package execution

import (
    "errors"
    "testing"
)

type staticPolicy struct {
    decisions map[NodeID]PolicyDecision
}

func (p staticPolicy) Evaluate(exec Execution, node ExecutionNode) (PolicyDecisionRecord, error) {
    d := p.decisions[node.ID]
    if d == "" {
        d = PolicyAllow
    }
    return PolicyDecisionRecord{
        ID:           PolicyDecisionID("pd-" + string(node.ID)),
        ExecutionRef: exec.ID,
        PlanRevision: exec.PlanRef.Revision,
        NodeRef:      node.ID,
        Decision:     d,
    }, nil
}

type staticResolver struct {
    failures map[NodeID]error
}

func (r staticResolver) Resolve(_ Execution, node ExecutionNode) (ExecutionTarget, error) {
    if err := r.failures[node.ID]; err != nil {
        return ExecutionTarget{}, err
    }
    return ExecutionTarget{
        ID:       TargetID("target-" + string(node.ID)),
        Revision: 1,
        Type:     TargetModel,
        Health:   TargetHealth{State: "HEALTHY"},
    }, nil
}

type staticEstimator struct {
    cost float64
}

func (e staticEstimator) Estimate(_ Execution, _ ExecutionNode, _ ExecutionTarget) (BudgetEstimate, error) {
    return BudgetEstimate{Cost: e.cost, NodeAttempts: 1}, nil
}

func testExecution() Execution {
    return Execution{
        ID:      "exec-1",
        GoalRef: "goal-1",
        PlanRef: PlanRevisionRef{PlanID: "plan-1", Revision: 1},
        Status:  ExecutionStatus{Phase: ExecutionRunning},
    }
}

func TestSupervisorSchedulesIndependentAllowedNodes(t *testing.T) {
    plan := ExecutionPlan{
        ID: "plan-1",
        Nodes: []ExecutionNode{
            {ID: "a", Type: NodeModel},
            {ID: "b", Type: NodeTool},
        },
    }
    ledger := NewBudgetLedger(BudgetState{Limit: BudgetLimit{MaxCost: 10, MaxNodeAttempts: 4}})
    s := NewSupervisor(staticPolicy{}, staticResolver{}, staticEstimator{cost: 2}, ledger)

    ready, conditions, err := s.SelectRunnable(testExecution(), plan, map[NodeID]NodeState{})
    if err != nil {
        t.Fatal(err)
    }
    if len(ready) != 2 {
        t.Fatalf("expected 2 ready nodes, got %d", len(ready))
    }
    if len(conditions) != 0 {
        t.Fatalf("expected no conditions, got %+v", conditions)
    }
    snap := ledger.Snapshot()
    if snap.Reserved.Cost != 4 || snap.Reserved.NodeAttempts != 2 {
        t.Fatalf("unexpected reserved budget: %+v", snap.Reserved)
    }
}

func TestSupervisorApprovalOnOneNodeDoesNotBlockIndependentNode(t *testing.T) {
    plan := ExecutionPlan{
        ID: "plan-1",
        Nodes: []ExecutionNode{
            {ID: "approval", Type: NodeWorkflow},
            {ID: "safe", Type: NodeModel},
        },
    }
    ledger := NewBudgetLedger(BudgetState{Limit: BudgetLimit{MaxCost: 10, MaxNodeAttempts: 4}})
    s := NewSupervisor(
        staticPolicy{decisions: map[NodeID]PolicyDecision{"approval": PolicyRequireApproval}},
        staticResolver{},
        staticEstimator{cost: 1},
        ledger,
    )

    ready, conditions, err := s.SelectRunnable(testExecution(), plan, map[NodeID]NodeState{})
    if err != nil {
        t.Fatal(err)
    }
    if len(ready) != 1 || ready[0].Node.ID != "safe" {
        t.Fatalf("expected safe node to remain runnable, got %+v", ready)
    }
    if len(conditions) != 1 || conditions[0].Type != ConditionApprovalRequired || conditions[0].NodeRef != "approval" {
        t.Fatalf("unexpected conditions: %+v", conditions)
    }
}

func TestSupervisorDoesNotRunDependentNodeBeforeDependencySucceeds(t *testing.T) {
    plan := ExecutionPlan{
        ID: "plan-1",
        Nodes: []ExecutionNode{
            {ID: "collect", Type: NodeTool},
            {ID: "analyze", Type: NodeModel, DependsOn: []NodeID{"collect"}},
        },
    }
    ledger := NewBudgetLedger(BudgetState{Limit: BudgetLimit{MaxCost: 10, MaxNodeAttempts: 4}})
    s := NewSupervisor(staticPolicy{}, staticResolver{}, staticEstimator{cost: 1}, ledger)

    ready, _, err := s.SelectRunnable(testExecution(), plan, map[NodeID]NodeState{"collect": NodeRunning})
    if err != nil {
        t.Fatal(err)
    }
    if len(ready) != 0 {
        t.Fatalf("expected no runnable nodes while dependency is running, got %+v", ready)
    }
}

func TestSupervisorBudgetPressureDoesNotReserveOverLimit(t *testing.T) {
    plan := ExecutionPlan{
        ID: "plan-1",
        Nodes: []ExecutionNode{
            {ID: "a", Type: NodeModel},
            {ID: "b", Type: NodeModel},
        },
    }
    ledger := NewBudgetLedger(BudgetState{Limit: BudgetLimit{MaxCost: 5, MaxNodeAttempts: 4}})
    s := NewSupervisor(staticPolicy{}, staticResolver{}, staticEstimator{cost: 3}, ledger)

    ready, conditions, err := s.SelectRunnable(testExecution(), plan, map[NodeID]NodeState{})
    if err != nil {
        t.Fatal(err)
    }
    if len(ready) != 1 {
        t.Fatalf("expected one node within budget, got %d", len(ready))
    }
    if len(conditions) != 1 || conditions[0].Type != ConditionBudgetPressure {
        t.Fatalf("expected one budget pressure condition, got %+v", conditions)
    }
    snap := ledger.Snapshot()
    if snap.Reserved.Cost != 3 {
        t.Fatalf("expected only 3 cost reserved, got %+v", snap.Reserved)
    }
}

func TestSupervisorTargetFailureProducesDegradedCondition(t *testing.T) {
    plan := ExecutionPlan{ID: "plan-1", Nodes: []ExecutionNode{{ID: "a", Type: NodeModel}}}
    ledger := NewBudgetLedger(BudgetState{Limit: BudgetLimit{MaxCost: 5, MaxNodeAttempts: 2}})
    s := NewSupervisor(staticPolicy{}, staticResolver{failures: map[NodeID]error{"a": errors.New("unhealthy")}}, staticEstimator{cost: 1}, ledger)

    ready, conditions, err := s.SelectRunnable(testExecution(), plan, map[NodeID]NodeState{})
    if err != nil {
        t.Fatal(err)
    }
    if len(ready) != 0 || len(conditions) != 1 || conditions[0].Type != ConditionTargetDegraded {
        t.Fatalf("unexpected result ready=%+v conditions=%+v", ready, conditions)
    }
}
