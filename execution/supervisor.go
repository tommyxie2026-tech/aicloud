package execution

import (
	"errors"
	"fmt"
	"sort"
)

var (
	ErrExecutionNotRunnable = errors.New("execution is not runnable")
	ErrPlanNotReady         = errors.New("plan has no runnable nodes")
	ErrPolicyApproval       = errors.New("policy requires approval")
	ErrPolicyRejected       = errors.New("policy denied action")
)

type NodeRuntime struct {
	NodeRef NodeID
	State   NodeState
}

type PolicyEvaluator interface {
	Evaluate(execution Execution, node ExecutionNode) (PolicyDecisionRecord, error)
}

type TargetResolver interface {
	Resolve(execution Execution, node ExecutionNode) (ExecutionTarget, error)
}

type CostEstimator interface {
	Estimate(execution Execution, node ExecutionNode, target ExecutionTarget) (BudgetEstimate, error)
}

// PreparedNode is a side-effect-free execution candidate. Policy eligibility,
// target selection and cost estimation are resolved, but no budget reservation
// has been created yet. Distributed workers must Claim first, then reserve
// budget using the durable AttemptRef so losing workers cannot leak reservations.
type PreparedNode struct {
	Node     ExecutionNode
	Target   ExecutionTarget
	Policy   PolicyDecisionRecord
	Estimate BudgetEstimate
}

// ReadyNode is the legacy in-memory bootstrap shape where budget is reserved by
// the Supervisor before execution. Persistent workers should use PreparedNode.
type ReadyNode struct {
	Node        ExecutionNode
	Target      ExecutionTarget
	Policy      PolicyDecisionRecord
	Reservation *BudgetReservation
}

type Supervisor struct {
	policy    PolicyEvaluator
	resolver  TargetResolver
	estimator CostEstimator
	budget    *BudgetLedger
}

func NewSupervisor(policy PolicyEvaluator, resolver TargetResolver, estimator CostEstimator, budget *BudgetLedger) *Supervisor {
	return &Supervisor{policy: policy, resolver: resolver, estimator: estimator, budget: budget}
}

// PrepareRunnable performs only deterministic/side-effect-free scheduling work.
// It deliberately does not reserve budget. This method is the entry point for
// the persistent worker path.
func (s *Supervisor) PrepareRunnable(execution Execution, plan ExecutionPlan, runtimes map[NodeID]NodeState) ([]PreparedNode, []ExecutionCondition, error) {
	if execution.Status.Phase != ExecutionRunning && execution.Status.Phase != ExecutionReady {
		return nil, nil, ErrExecutionNotRunnable
	}

	nodes := append([]ExecutionNode(nil), plan.Nodes...)
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })

	var prepared []PreparedNode
	var conditions []ExecutionCondition

	for _, node := range nodes {
		state := runtimes[node.ID]
		if state == "" {
			state = NodePending
		}
		if state != NodePending && state != NodeReady {
			continue
		}
		if !dependenciesSucceeded(node, runtimes) {
			continue
		}

		policy, err := s.policy.Evaluate(execution, node)
		if err != nil {
			return nil, conditions, fmt.Errorf("policy evaluate node %s: %w", node.ID, err)
		}
		switch policy.Decision {
		case PolicyDeny:
			conditions = append(conditions, ExecutionCondition{
				Type:    ConditionPartiallyBlocked,
				Status:  true,
				NodeRef: node.ID,
				Reason:  ErrPolicyRejected.Error(),
			})
			continue
		case PolicyRequireApproval:
			conditions = append(conditions, ExecutionCondition{
				Type:    ConditionApprovalRequired,
				Status:  true,
				NodeRef: node.ID,
				Reason:  ErrPolicyApproval.Error(),
			})
			continue
		case PolicyAllow:
		default:
			return nil, conditions, fmt.Errorf("node %s returned unknown policy decision %q", node.ID, policy.Decision)
		}

		target, err := s.resolver.Resolve(execution, node)
		if err != nil {
			conditions = append(conditions, ExecutionCondition{
				Type:    ConditionTargetDegraded,
				Status:  true,
				NodeRef: node.ID,
				Reason:  err.Error(),
			})
			continue
		}

		estimate, err := s.estimator.Estimate(execution, node, target)
		if err != nil {
			return nil, conditions, fmt.Errorf("estimate node %s: %w", node.ID, err)
		}

		prepared = append(prepared, PreparedNode{
			Node:     node,
			Target:   target,
			Policy:   policy,
			Estimate: estimate,
		})
	}

	if len(prepared) == 0 && len(conditions) == 0 {
		if hasActiveOrWaitingNodes(runtimes) {
			return nil, nil, nil
		}
		return nil, nil, ErrPlanNotReady
	}
	return prepared, conditions, nil
}

// SelectRunnable preserves the original in-memory bootstrap behavior. It is not
// safe as a distributed scheduling primitive because reservation happens before
// durable Claim. New worker code must use PrepareRunnable instead.
func (s *Supervisor) SelectRunnable(execution Execution, plan ExecutionPlan, runtimes map[NodeID]NodeState) ([]ReadyNode, []ExecutionCondition, error) {
	prepared, conditions, err := s.PrepareRunnable(execution, plan, runtimes)
	if err != nil {
		return nil, conditions, err
	}

	var ready []ReadyNode
	for _, candidate := range prepared {
		reservationID := fmt.Sprintf("%s/%s/%d", execution.ID, candidate.Node.ID, nextAttemptNumber(candidate.Node.ID, runtimes))
		reservation, err := s.budget.Reserve(reservationID, execution.ID, candidate.Node.ID, candidate.Estimate)
		if err != nil {
			if errors.Is(err, ErrBudgetExceeded) {
				conditions = append(conditions, ExecutionCondition{
					Type:    ConditionBudgetPressure,
					Status:  true,
					NodeRef: candidate.Node.ID,
					Reason:  err.Error(),
				})
				continue
			}
			return nil, conditions, err
		}
		ready = append(ready, ReadyNode{
			Node:        candidate.Node,
			Target:      candidate.Target,
			Policy:      candidate.Policy,
			Reservation: reservation,
		})
	}
	return ready, conditions, nil
}

func dependenciesSucceeded(node ExecutionNode, runtimes map[NodeID]NodeState) bool {
	for _, dep := range node.DependsOn {
		if runtimes[dep] != NodeSucceeded {
			return false
		}
	}
	return true
}

func hasActiveOrWaitingNodes(runtimes map[NodeID]NodeState) bool {
	for _, state := range runtimes {
		switch state {
		case NodeRunning, NodeWaitingApproval, NodeBlocked:
			return true
		}
	}
	return false
}

// Attempt numbering is a persistence concern for the persistent worker path.
// The legacy in-memory bootstrap still assumes attempt 1.
func nextAttemptNumber(_ NodeID, _ map[NodeID]NodeState) int {
	return 1
}
