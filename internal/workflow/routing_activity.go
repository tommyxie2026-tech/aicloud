package workflow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	execution "github.com/tommyxie2026-tech/aicloud/execution"
	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/executionbinding"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
	"github.com/tommyxie2026-tech/aicloud/internal/router"
	tracepkg "github.com/tommyxie2026-tech/aicloud/internal/trace"
	"go.temporal.io/sdk/temporal"
)

const (
	RoutingActivityVersion              = "ecp-route-v1"
	ErrorTypeRoutePolicyDenied          = "ROUTE_POLICY_DENIED"
	ErrorTypeRouteApprovalRequired      = "ROUTE_POLICY_APPROVAL_REQUIRED"
	ErrorTypeRouteSelectionInconsistent = "ROUTE_SELECTION_INCONSISTENT"
)

type RoutingLineageStore interface {
	ListPlansByTask(context.Context, string) ([]execution.ExecutionPlan, error)
	GetGoal(context.Context, execution.GoalID) (execution.Goal, error)
}

type RoutePlanner interface {
	Plan(context.Context, router.Request) (domain.RouteDecision, error)
}

type RoutingPolicyGate interface {
	Evaluate(context.Context, domain.Task, execution.Goal, execution.ExecutionPlan, execution.ExecutionNode) (execution.PolicyDecisionRecord, error)
}

type RouteRequestBuilder interface {
	Build(domain.Task, execution.Goal, execution.ExecutionPlan, execution.ExecutionNode) (router.Request, error)
}

type BaselineRoutingPolicy struct{}

func (BaselineRoutingPolicy) Evaluate(_ context.Context, task domain.Task, goal execution.Goal, plan execution.ExecutionPlan, node execution.ExecutionNode) (execution.PolicyDecisionRecord, error) {
	if strings.TrimSpace(task.ID) == "" || goal.TaskRef != task.ID || plan.GoalRef != goal.ID {
		return execution.PolicyDecisionRecord{}, fmt.Errorf("routing policy lineage is invalid")
	}
	decision := execution.PolicyAllow
	reasons := []string{"BASELINE_MODEL_ROUTE"}
	if node.Type != execution.NodeModel {
		decision = execution.PolicyDeny
		reasons = []string{"UNSUPPORTED_NODE_TYPE"}
	}
	if node.Effect.Class != execution.EffectPure && node.Effect.Class != execution.EffectReadOnly {
		decision = execution.PolicyDeny
		reasons = []string{"MUTATION_REQUIRES_EXPLICIT_POLICY"}
	}
	return execution.PolicyDecisionRecord{
		ID:            execution.PolicyDecisionID("policy/" + task.ID + "/" + string(node.ID) + "/route/v1"),
		PolicySetRef:  "ecp/baseline-routing",
		PolicyVersion: "v1",
		PlanRevision:  plan.Revision,
		NodeRef:       node.ID,
		Principal:     "system:temporal-task-lifecycle",
		Action:        "route:model-target",
		Resource:      task.ID + "/" + string(node.ID),
		Decision:      decision,
		ReasonCodes:   reasons,
	}, nil
}

type BaselineRouteRequestBuilder struct{}

func (BaselineRouteRequestBuilder) Build(task domain.Task, goal execution.Goal, plan execution.ExecutionPlan, node execution.ExecutionNode) (router.Request, error) {
	if task.ID == "" || plan.GoalRef != goal.ID || node.Type != execution.NodeModel {
		return router.Request{}, fmt.Errorf("model routing request lineage is invalid")
	}
	currency := strings.TrimSpace(task.Currency)
	if currency == "" {
		currency = "USD"
	}
	required := append([]string(nil), node.CapabilityRequirement.Modality...)
	if node.CapabilityRequirement.Coding != "" {
		required = append(required, "coding")
	}
	if node.CapabilityRequirement.Reasoning != "" {
		required = append(required, "reasoning")
	}
	if node.CapabilityRequirement.ToolCalling != "" {
		required = append(required, "tool-calling")
	}
	return router.Request{
		TaskID:               task.ID,
		RouteClass:           domain.RouteEfficient,
		RequiredCapabilities: required,
		Budget:               goal.Budget.MaxCost,
		Currency:             currency,
		EvidenceVersion:      fmt.Sprintf("plan:%s@%d", plan.ID, plan.Revision),
	}, nil
}

type DurableRoutingActivity struct {
	Tasks          domain.TaskRepository
	Lineage        RoutingLineageStore
	Policy         RoutingPolicyGate
	Planner        RoutePlanner
	Deployments    domain.DeploymentRepository
	Selection      repository.RouteSelectionStore
	RequestBuilder RouteRequestBuilder
	Now            func() time.Time
}

func (a DurableRoutingActivity) Route(ctx context.Context, input StepInput) error {
	if a.Tasks == nil || a.Lineage == nil || a.Policy == nil || a.Planner == nil || a.Selection == nil || a.RequestBuilder == nil {
		return fmt.Errorf("durable routing activity is not fully configured")
	}
	if err := validateRoutingStepInput(input); err != nil {
		return err
	}
	ctx = executionWorkflowProjectContext(ctx, input)
	lookup := routeSelectionLookup(input)
	if replay, found, err := a.Selection.ResolveRouteSelection(ctx, lookup); err != nil {
		return err
	} else if found {
		if replay.Decision.TaskID != input.TaskID || replay.Plan.PlanID == "" || replay.Plan.Revision < 1 || replay.Node == "" || replay.Policy.ID == "" || replay.Policy.Decision != execution.PolicyAllow || replay.Policy.PlanRevision != replay.Plan.Revision || replay.Policy.NodeRef != replay.Node || replay.Target.ID == "" || replay.Target.Snapshot.Digest == "" {
			return temporal.NewNonRetryableApplicationError("durable route selection replay does not match workflow execution binding", ErrorTypeRouteSelectionInconsistent, nil)
		}
		return nil
	}

	task, err := a.Tasks.Get(ctx, input.TaskID)
	if err != nil {
		return fmt.Errorf("load task for routing: %w", err)
	}
	if task.TenantID != input.TenantID || task.ProjectID != input.ProjectID || task.TraceID != input.TraceID {
		return ErrLifecycleScope
	}
	if task.Status != domain.TaskRouting {
		return fmt.Errorf("%w: cannot select route from %s", domain.ErrInvalidTaskTransition, task.Status)
	}
	if task.RouteDecisionID != "" {
		return temporal.NewNonRetryableApplicationError("Task already references route evidence but durable workflow replay is missing", ErrorTypeRouteSelectionInconsistent, nil)
	}

	plan, err := a.currentPlan(ctx, task.ID)
	if err != nil {
		return err
	}
	goal, err := a.Lineage.GetGoal(ctx, plan.GoalRef)
	if err != nil {
		return fmt.Errorf("load routing goal: %w", err)
	}
	if goal.TaskRef != task.ID {
		return fmt.Errorf("routing goal does not belong to task")
	}
	node, err := selectTaskRoutingNode(plan)
	if err != nil {
		return err
	}

	policy, err := a.Policy.Evaluate(ctx, task, goal, plan, node)
	if err != nil {
		return fmt.Errorf("evaluate routing policy: %w", err)
	}
	switch policy.Decision {
	case execution.PolicyAllow:
	case execution.PolicyDeny:
		return temporal.NewNonRetryableApplicationError("routing policy denied target selection", ErrorTypeRoutePolicyDenied, nil)
	case execution.PolicyRequireApproval:
		return temporal.NewNonRetryableApplicationError("routing policy requires approval before target selection", ErrorTypeRouteApprovalRequired, nil)
	default:
		return fmt.Errorf("routing policy returned unknown decision %q", policy.Decision)
	}

	request, err := a.RequestBuilder.Build(task, goal, plan, node)
	if err != nil {
		return fmt.Errorf("build route request: %w", err)
	}
	request.PolicyVersion = policy.PolicyVersion
	decision, err := a.Planner.Plan(ctx, request)
	if err != nil {
		return fmt.Errorf("plan route: %w", err)
	}
	if decision.TaskID != task.ID {
		return fmt.Errorf("router returned decision for a different task")
	}

	var deployment *domain.Deployment
	if decision.Selected.DeploymentID != "" {
		if a.Deployments == nil {
			return fmt.Errorf("selected deployment requires deployment repository")
		}
		item, err := a.Deployments.Get(ctx, decision.Selected.DeploymentID)
		if err != nil {
			return fmt.Errorf("load selected deployment snapshot: %w", err)
		}
		deployment = &item
	}
	target, err := executionbinding.TargetFromRouteDecision(decision, deployment)
	if err != nil {
		return fmt.Errorf("freeze execution target: %w", err)
	}

	now := time.Now().UTC()
	if a.Now != nil {
		now = a.Now().UTC()
	}
	payload, err := json.Marshal(map[string]any{
		"routeDecisionId": decision.ID,
		"planId":          plan.ID,
		"planRevision":    plan.Revision,
		"nodeId":          node.ID,
		"policyDecision":  policy,
		"target":          target,
	})
	if err != nil {
		return fmt.Errorf("encode route selection event: %w", err)
	}
	task.UpdatedAt = now
	planRef := execution.PlanRevisionRef{PlanID: plan.ID, Revision: plan.Revision}
	result, err := a.Selection.CommitRouteSelection(ctx, repository.RouteSelectionCommit{
		Task: task, Plan: planRef, Node: node.ID, Decision: decision, Policy: policy, Target: target,
		Event: domain.TaskEvent{
			EventID: tracepkg.NewID("task-event"), EventType: "TaskRouteSelected",
			Actor: domain.TaskEventActor{PrincipalType: "system", SubjectID: "temporal-task-lifecycle"},
			Payload: payload, SchemaVersion: 1, OccurredAt: now, CreatedAt: now,
		},
		Idempotency: domain.IdempotencyRecord{
			TenantID: lookup.TenantID, ProjectID: lookup.ProjectID, Operation: lookup.Operation,
			Key: lookup.Key, RequestDigest: lookup.RequestDigest, Status: domain.IdempotencyCompleted,
			CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour),
		},
	})
	if err != nil {
		if replay, found, replayErr := a.Selection.ResolveRouteSelection(ctx, lookup); replayErr == nil && found && replay.Decision.TaskID == task.ID && replay.Plan == planRef && replay.Node == node.ID && replay.Policy.ID != "" && replay.Policy.Decision == execution.PolicyAllow {
			return nil
		}
		return fmt.Errorf("commit route selection: %w", err)
	}
	if result.Decision.TaskID != task.ID || result.Plan != planRef || result.Node != node.ID || result.Policy.ID == "" || result.Policy.Decision != execution.PolicyAllow || result.Target.Snapshot.Digest == "" {
		return temporal.NewNonRetryableApplicationError("committed route selection is incomplete", ErrorTypeRouteSelectionInconsistent, nil)
	}
	return nil
}

func (a DurableRoutingActivity) currentPlan(ctx context.Context, taskID string) (execution.ExecutionPlan, error) {
	plans, err := a.Lineage.ListPlansByTask(ctx, taskID)
	if err != nil {
		return execution.ExecutionPlan{}, fmt.Errorf("list task execution plans: %w", err)
	}
	if len(plans) == 0 {
		return execution.ExecutionPlan{}, fmt.Errorf("task has no durable execution plan")
	}
	current := plans[0]
	for _, candidate := range plans[1:] {
		if candidate.CreatedAt.After(current.CreatedAt) || (candidate.CreatedAt.Equal(current.CreatedAt) && candidate.Revision > current.Revision) || (candidate.CreatedAt.Equal(current.CreatedAt) && candidate.Revision == current.Revision && string(candidate.ID) > string(current.ID)) {
			current = candidate
		}
	}
	if err := execution.ValidatePlan(current); err != nil {
		return execution.ExecutionPlan{}, fmt.Errorf("validate persisted execution plan: %w", err)
	}
	return current, nil
}

func selectTaskRoutingNode(plan execution.ExecutionPlan) (execution.ExecutionNode, error) {
	var selected *execution.ExecutionNode
	for i := range plan.Nodes {
		if plan.Nodes[i].Type != execution.NodeModel {
			continue
		}
		if selected != nil {
			return execution.ExecutionNode{}, fmt.Errorf("task-level route activity cannot resolve multiple model nodes")
		}
		selected = &plan.Nodes[i]
	}
	if selected == nil {
		return execution.ExecutionNode{}, fmt.Errorf("execution plan has no model node to route")
	}
	return *selected, nil
}

func routeSelectionLookup(input StepInput) repository.IdempotencyLookup {
	body := strings.Join([]string{RoutingActivityVersion, strings.TrimSpace(input.TenantID), strings.TrimSpace(input.ProjectID), strings.TrimSpace(input.TaskID), strings.TrimSpace(input.TraceID)}, "\x00")
	sum := sha256.Sum256([]byte(body))
	return repository.IdempotencyLookup{
		TenantID: strings.TrimSpace(input.TenantID), ProjectID: strings.TrimSpace(input.ProjectID),
		Operation: "workflow:task-route-select:" + RoutingActivityVersion,
		Key: strings.TrimSpace(input.TaskID) + ":route:" + RoutingActivityVersion,
		RequestDigest: "sha256:" + hex.EncodeToString(sum[:]),
	}
}

func validateRoutingStepInput(input StepInput) error {
	if strings.TrimSpace(input.TenantID) == "" || strings.TrimSpace(input.ProjectID) == "" || strings.TrimSpace(input.TaskID) == "" || strings.TrimSpace(input.TraceID) == "" {
		return fmt.Errorf("routing step requires tenant, project, task and trace identity")
	}
	return nil
}

type RoutingLifecycleActivities struct {
	Base    LifecycleActivities
	Routing DurableRoutingActivity
}

func (a RoutingLifecycleActivities) LoadTask(ctx context.Context, input LoadTaskInput) (TaskSnapshot, error) {
	return a.Base.LoadTask(ctx, input)
}
func (a RoutingLifecycleActivities) TransitionTask(ctx context.Context, input TransitionTaskInput) (TaskSnapshot, error) {
	return a.Base.TransitionTask(ctx, input)
}
func (a RoutingLifecycleActivities) PlanStub(ctx context.Context, input StepInput) error {
	return a.Base.PlanStub(ctx, input)
}
func (a RoutingLifecycleActivities) RouteStub(ctx context.Context, input StepInput) error {
	return a.Routing.Route(ctx, input)
}
func (a RoutingLifecycleActivities) ExecuteStub(ctx context.Context, input StepInput) error {
	return a.Base.ExecuteStub(ctx, input)
}
func (a RoutingLifecycleActivities) ValidateStub(ctx context.Context, input StepInput) error {
	return a.Base.ValidateStub(ctx, input)
}
