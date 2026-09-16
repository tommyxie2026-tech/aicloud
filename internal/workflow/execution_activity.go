package workflow

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	execution "github.com/tommyxie2026-tech/aicloud/execution"
	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
	"go.temporal.io/sdk/temporal"
)

const (
	ExecutionActivityVersion              = "ecp-execute-v1"
	ErrorTypeExecutionBindingInconsistent = "EXECUTION_BINDING_INCONSISTENT"
	ErrorTypeExecutionRetryNotAllowed     = "EXECUTION_RETRY_NOT_ALLOWED"
)

type ExecutionActivityLineageStore interface {
	GetGoal(context.Context, execution.GoalID) (execution.Goal, error)
	GetPlan(context.Context, execution.PlanID, int64) (execution.ExecutionPlan, error)
	CreateExecution(context.Context, execution.Execution) (execution.Execution, error)
	GetExecution(context.Context, execution.ExecutionID) (execution.Execution, error)
}

type ExecutionRuntimeStore interface {
	Register(context.Context, execution.NodeRuntimeRecord) error
	Get(context.Context, execution.ExecutionID, int64, execution.NodeID) (execution.NodeRuntimeRecord, *execution.NodeLease, error)
	RequeueFailed(context.Context, execution.ExecutionID, int64, execution.NodeID) error
	ListAttempts(context.Context, execution.ExecutionID, int64, execution.NodeID) ([]execution.AttemptRecord, error)
}

type ExecutionNodeExecutor interface {
	Execute(context.Context, execution.Execution, execution.PreparedNode) (execution.WorkerExecutionResult, error)
}

var (
	_ ExecutionActivityLineageStore = (*repository.ScopedPostgresExecutionLineage)(nil)
	_ ExecutionRuntimeStore         = (*repository.PostgresExecutionNodeLeases)(nil)
	_ ExecutionNodeExecutor         = (*execution.PersistentWorker)(nil)
)

// DurableExecutionActivity consumes the exact route binding frozen during
// ROUTING. It never re-runs policy, routing or target resolution. Temporal
// retries recover from Execution/NodeRuntime/Attempt state instead.
type DurableExecutionActivity struct {
	Tasks     domain.TaskRepository
	Lineage   ExecutionActivityLineageStore
	Selection repository.RouteSelectionStore
	Runtime   ExecutionRuntimeStore
	Worker    ExecutionNodeExecutor
	Now       func() time.Time
}

func (a DurableExecutionActivity) Execute(ctx context.Context, input StepInput) error {
	if a.Tasks == nil || a.Lineage == nil || a.Selection == nil || a.Runtime == nil || a.Worker == nil {
		return fmt.Errorf("durable execution activity is not fully configured")
	}
	if err := validateExecutionStepInput(input); err != nil {
		return err
	}
	ctx = executionWorkflowProjectContext(ctx, input)

	task, err := a.Tasks.Get(ctx, input.TaskID)
	if err != nil {
		return fmt.Errorf("load task for execution: %w", err)
	}
	if task.TenantID != input.TenantID || task.ProjectID != input.ProjectID || task.TraceID != input.TraceID {
		return ErrLifecycleScope
	}
	if task.Status != domain.TaskExecuting {
		return fmt.Errorf("%w: cannot execute from %s", domain.ErrInvalidTaskTransition, task.Status)
	}

	binding, found, err := a.Selection.ResolveRouteSelection(ctx, routeSelectionLookup(input))
	if err != nil {
		return fmt.Errorf("resolve frozen route selection: %w", err)
	}
	if !found || !validExecutionBinding(task.ID, binding) {
		return temporal.NewNonRetryableApplicationError(
			"durable execution binding is missing or inconsistent",
			ErrorTypeExecutionBindingInconsistent,
			nil,
		)
	}

	plan, err := a.Lineage.GetPlan(ctx, binding.Plan.PlanID, binding.Plan.Revision)
	if err != nil {
		return fmt.Errorf("load frozen execution plan: %w", err)
	}
	if err := execution.ValidatePlan(plan); err != nil {
		return temporal.NewNonRetryableApplicationError(
			"persisted execution plan is invalid",
			ErrorTypeExecutionBindingInconsistent,
			err,
		)
	}
	if plan.ID != binding.Plan.PlanID || plan.Revision != binding.Plan.Revision {
		return bindingInconsistentError("persisted plan does not match frozen plan revision")
	}
	node, err := nodeByID(plan, binding.Node)
	if err != nil {
		return bindingInconsistentError(err.Error())
	}
	goal, err := a.Lineage.GetGoal(ctx, plan.GoalRef)
	if err != nil {
		return fmt.Errorf("load execution goal: %w", err)
	}
	if goal.TaskRef != task.ID {
		return bindingInconsistentError("execution goal does not belong to task")
	}

	now := time.Now().UTC()
	if a.Now != nil {
		now = a.Now().UTC()
	}
	item, err := a.ensureExecution(ctx, task, goal, plan, now)
	if err != nil {
		return err
	}
	if item.TaskRef != task.ID || item.GoalRef != goal.ID || item.PlanRef != binding.Plan {
		return bindingInconsistentError("durable Execution parent lineage mismatch")
	}

	runtime, err := a.ensureRuntime(ctx, item, node)
	if err != nil {
		return err
	}
	if runtime.State == execution.NodeSucceeded {
		return nil
	}
	if runtime.State == execution.NodeFailed {
		if err := a.prepareRetry(ctx, item, node); err != nil {
			return err
		}
		runtime, _, err = a.Runtime.Get(ctx, item.ID, item.PlanRef.Revision, node.ID)
		if err != nil {
			return fmt.Errorf("reload requeued execution node: %w", err)
		}
	}
	// RUNNING is intentionally handed back to PersistentWorker. Its lease fence
	// decides whether the previous worker still owns the node or an expired
	// read-only attempt can be safely reclaimed. The Activity must not invent a
	// second recovery authority.
	if runtime.State != execution.NodeReady && runtime.State != execution.NodeRunning {
		return fmt.Errorf("execution node %s is not executable from state %s", node.ID, runtime.State)
	}

	candidate, err := preparedNodeFromFrozenBinding(node, binding)
	if err != nil {
		return bindingInconsistentError(err.Error())
	}
	if _, err := a.Worker.Execute(ctx, item, candidate); err != nil {
		return err
	}
	completed, _, err := a.Runtime.Get(ctx, item.ID, item.PlanRef.Revision, node.ID)
	if err != nil {
		return fmt.Errorf("load completed execution node: %w", err)
	}
	if completed.State != execution.NodeSucceeded {
		return fmt.Errorf("worker returned success but node %s is %s", node.ID, completed.State)
	}
	return nil
}

func (a DurableExecutionActivity) ensureExecution(ctx context.Context, task domain.Task, goal execution.Goal, plan execution.ExecutionPlan, now time.Time) (execution.Execution, error) {
	id := execution.ExecutionID(fmt.Sprintf("execution/%s/%s@%d/%s", task.ID, plan.ID, plan.Revision, ExecutionActivityVersion))
	existing, err := a.Lineage.GetExecution(ctx, id)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return execution.Execution{}, fmt.Errorf("load durable Execution: %w", err)
	}
	item := execution.Execution{
		ID:      id,
		TaskRef: task.ID,
		GoalRef: goal.ID,
		PlanRef: execution.PlanRevisionRef{PlanID: plan.ID, Revision: plan.Revision},
		Identity: execution.Identity{
			Principal: "system:temporal-task-lifecycle",
			Tenant:    task.TenantID,
		},
		Budget:    execution.BudgetState{Limit: goal.Budget},
		Status:    execution.ExecutionStatus{Phase: execution.ExecutionRunning},
		CreatedAt: now,
		UpdatedAt: now,
	}
	created, createErr := a.Lineage.CreateExecution(ctx, item)
	if createErr == nil {
		return created, nil
	}
	// Commit-before-ack or a concurrent duplicate may have won. The deterministic
	// Execution ID is the replay key, so reload before surfacing the create error.
	existing, getErr := a.Lineage.GetExecution(ctx, id)
	if getErr == nil {
		return existing, nil
	}
	return execution.Execution{}, fmt.Errorf("create durable Execution: %w", createErr)
}

func (a DurableExecutionActivity) ensureRuntime(ctx context.Context, item execution.Execution, node execution.ExecutionNode) (execution.NodeRuntimeRecord, error) {
	record, _, err := a.Runtime.Get(ctx, item.ID, item.PlanRef.Revision, node.ID)
	if err == nil {
		return record, nil
	}
	if !errors.Is(err, repository.ErrNodeRuntimeNotFound) {
		return execution.NodeRuntimeRecord{}, fmt.Errorf("load execution node runtime: %w", err)
	}
	register := execution.NodeRuntimeRecord{
		ExecutionRef:   item.ID,
		PlanRevision:   item.PlanRef.Revision,
		NodeRef:        node.ID,
		State:          execution.NodeReady,
		EffectClass:    node.Effect.Class,
		RetrySafe:      node.Effect.RetrySafe,
		IdempotencyKey: node.Effect.IdempotencyKey,
	}
	if registerErr := a.Runtime.Register(ctx, register); registerErr != nil && !errors.Is(registerErr, repository.ErrNodeRuntimeExists) {
		return execution.NodeRuntimeRecord{}, fmt.Errorf("register execution node runtime: %w", registerErr)
	}
	record, _, err = a.Runtime.Get(ctx, item.ID, item.PlanRef.Revision, node.ID)
	if err != nil {
		return execution.NodeRuntimeRecord{}, fmt.Errorf("reload execution node runtime: %w", err)
	}
	return record, nil
}

func (a DurableExecutionActivity) prepareRetry(ctx context.Context, item execution.Execution, node execution.ExecutionNode) error {
	attempts, err := a.Runtime.ListAttempts(ctx, item.ID, item.PlanRef.Revision, node.ID)
	if err != nil {
		return fmt.Errorf("list execution attempts for retry: %w", err)
	}
	if len(attempts) == 0 {
		return retryNotAllowedError("failed node has no durable attempt evidence")
	}
	latest := attempts[len(attempts)-1]
	maxAttempts := node.RetryPolicy.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 1
	}
	if latest.Attempt.AttemptNumber >= maxAttempts {
		return retryNotAllowedError("execution node exhausted retry attempts")
	}
	if !node.Effect.RetrySafe || isMutationNode(node) {
		return retryNotAllowedError("execution node effect is not safe for automatic retry")
	}
	if !retryClassAllowed(node.RetryPolicy.RetryOn, latest.Attempt.ErrorClass) {
		return retryNotAllowedError("execution failure class is not retryable by the frozen plan")
	}
	if err := a.Runtime.RequeueFailed(ctx, item.ID, item.PlanRef.Revision, node.ID); err != nil {
		return fmt.Errorf("requeue failed execution node: %w", err)
	}
	return nil
}

func validExecutionBinding(taskID string, binding repository.RouteSelectionResult) bool {
	return binding.Decision.TaskID == taskID && binding.Plan.PlanID != "" && binding.Plan.Revision > 0 && binding.Node != "" &&
		binding.Policy.ID != "" && binding.Policy.Decision == execution.PolicyAllow &&
		binding.Policy.PlanRevision == binding.Plan.Revision && binding.Policy.NodeRef == binding.Node &&
		binding.Target.ID != "" && binding.Target.Revision > 0 && strings.TrimSpace(binding.Target.Snapshot.Digest) != ""
}

func preparedNodeFromFrozenBinding(node execution.ExecutionNode, binding repository.RouteSelectionResult) (execution.PreparedNode, error) {
	if binding.Policy.NodeRef != node.ID || binding.Policy.PlanRevision != binding.Plan.Revision {
		return execution.PreparedNode{}, fmt.Errorf("frozen policy does not match plan node")
	}
	if binding.Decision.Selected.EstimatedCost < 0 {
		return execution.PreparedNode{}, fmt.Errorf("frozen route estimated cost cannot be negative")
	}
	estimate := execution.BudgetEstimate{Cost: binding.Decision.Selected.EstimatedCost, NodeAttempts: 1}
	if binding.Target.Type == execution.TargetModel {
		estimate.FrontierCalls = 1
	}
	if binding.Target.Type == execution.TargetTool {
		estimate.ToolCalls = 1
	}
	return execution.PreparedNode{Node: node, Target: binding.Target, Policy: binding.Policy, Estimate: estimate}, nil
}

func nodeByID(plan execution.ExecutionPlan, id execution.NodeID) (execution.ExecutionNode, error) {
	for _, node := range plan.Nodes {
		if node.ID == id {
			return node, nil
		}
	}
	return execution.ExecutionNode{}, fmt.Errorf("frozen node %s is not present in plan %s@%d", id, plan.ID, plan.Revision)
}

func retryClassAllowed(allowed []execution.ErrorClass, actual execution.ErrorClass) bool {
	if actual == "" {
		return false
	}
	for _, item := range allowed {
		if item == actual {
			return true
		}
	}
	return false
}

func isMutationNode(node execution.ExecutionNode) bool {
	return node.Effect.Class == execution.EffectIdempotentMutation || node.Effect.Class == execution.EffectNonIdempotentMutation
}

func bindingInconsistentError(message string) error {
	return temporal.NewNonRetryableApplicationError(message, ErrorTypeExecutionBindingInconsistent, nil)
}

func retryNotAllowedError(message string) error {
	return temporal.NewNonRetryableApplicationError(message, ErrorTypeExecutionRetryNotAllowed, nil)
}

func validateExecutionStepInput(input StepInput) error {
	if strings.TrimSpace(input.TenantID) == "" || strings.TrimSpace(input.ProjectID) == "" || strings.TrimSpace(input.TaskID) == "" || strings.TrimSpace(input.TraceID) == "" {
		return fmt.Errorf("execution step requires tenant, project, task and trace identity")
	}
	return nil
}

// ExecutionLifecycleActivities progressively replaces only ExecuteStub while
// preserving all registered Temporal Activity names for workflow replay.
type ExecutionLifecycleActivities struct {
	Base      LifecycleActivities
	Execution DurableExecutionActivity
}

func (a ExecutionLifecycleActivities) LoadTask(ctx context.Context, input LoadTaskInput) (TaskSnapshot, error) {
	return a.Base.LoadTask(ctx, input)
}

func (a ExecutionLifecycleActivities) TransitionTask(ctx context.Context, input TransitionTaskInput) (TaskSnapshot, error) {
	return a.Base.TransitionTask(ctx, input)
}

func (a ExecutionLifecycleActivities) PlanStub(ctx context.Context, input StepInput) error {
	return a.Base.PlanStub(ctx, input)
}

func (a ExecutionLifecycleActivities) RouteStub(ctx context.Context, input StepInput) error {
	return a.Base.RouteStub(ctx, input)
}

func (a ExecutionLifecycleActivities) ExecuteStub(ctx context.Context, input StepInput) error {
	return a.Execution.Execute(ctx, input)
}

func (a ExecutionLifecycleActivities) ValidateStub(ctx context.Context, input StepInput) error {
	return a.Base.ValidateStub(ctx, input)
}
