package workflow

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	execution "github.com/tommyxie2026-tech/aicloud/execution"
	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/executionbinding"
	"github.com/tommyxie2026-tech/aicloud/internal/identity"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
)

// PlanningLineageStore is the narrow ECP-C2 persistence boundary used by the
// Temporal planning Activity. The canonical PostgreSQL implementation is
// repository.ScopedPostgresExecutionLineage.
type PlanningLineageStore interface {
	CreateGoal(context.Context, execution.Goal) (execution.Goal, error)
	GetGoal(context.Context, execution.GoalID) (execution.Goal, error)
	CreatePlan(context.Context, execution.ExecutionPlan) (execution.ExecutionPlan, error)
	GetPlan(context.Context, execution.PlanID, int64) (execution.ExecutionPlan, error)
}

// ExecutionPlanBuilder converts one immutable Goal snapshot into a versioned,
// bounded execution DAG. It has no persistence or routing authority.
type ExecutionPlanBuilder interface {
	Build(context.Context, domain.Task, execution.Goal) (execution.ExecutionPlan, error)
}

// BaselineModelPlanBuilder bridges the current model-only product path into the
// canonical execution domain. ECP-C2 route convergence resolves the model node
// to an approved target later; planning itself does not select a Provider.
type BaselineModelPlanBuilder struct {
	Now func() time.Time
}

func (b BaselineModelPlanBuilder) Build(_ context.Context, task domain.Task, goal execution.Goal) (execution.ExecutionPlan, error) {
	if strings.TrimSpace(task.ID) == "" || goal.ID == "" || goal.TaskRef != task.ID {
		return execution.ExecutionPlan{}, fmt.Errorf("task and goal lineage is required")
	}
	now := time.Now().UTC()
	if b.Now != nil {
		now = b.Now().UTC()
	}
	plan := execution.ExecutionPlan{
		ID:       execution.PlanID("plan/" + task.ID + "/model/v1"),
		GoalRef:  goal.ID,
		Revision: 1,
		Nodes: []execution.ExecutionNode{{
			ID:   execution.NodeID("model/main"),
			Type: execution.NodeModel,
			RetryPolicy: execution.RetryPolicy{
				MaxAttempts: 3,
				RetryOn: []execution.ErrorClass{
					execution.ErrorTransient,
					execution.ErrorTargetUnavailable,
					execution.ErrorTimeout,
				},
			},
			Effect:               execution.EffectSpec{Class: execution.EffectPure, RetrySafe: true},
			VerificationRequired: true,
		}},
		CreatedAt: now,
	}
	if err := execution.ValidatePlan(plan); err != nil {
		return execution.ExecutionPlan{}, err
	}
	return plan, nil
}

// DurablePlanningActivity replaces the semantic behavior of PlanStub without
// changing the registered Temporal Activity name. Keeping the activity name
// stable protects existing replay histories while new executions gain durable
// Goal/Plan lineage.
type DurablePlanningActivity struct {
	Tasks   domain.TaskRepository
	Lineage PlanningLineageStore
	Builder ExecutionPlanBuilder
}

func (a DurablePlanningActivity) Plan(ctx context.Context, input StepInput) error {
	if a.Tasks == nil || a.Lineage == nil || a.Builder == nil {
		return fmt.Errorf("durable planning activity is not fully configured")
	}
	if err := validatePlanningStepInput(input); err != nil {
		return err
	}
	ctx = executionWorkflowProjectContext(ctx, input)
	task, err := a.Tasks.Get(ctx, input.TaskID)
	if err != nil {
		return fmt.Errorf("load task for planning: %w", err)
	}
	if task.TenantID != input.TenantID || task.ProjectID != input.ProjectID || task.TraceID != input.TraceID {
		return ErrLifecycleScope
	}

	goal, err := executionbinding.GoalSnapshotFromTask(task, executionbinding.GoalSnapshotOptions{})
	if err != nil {
		return err
	}
	plan, err := a.Builder.Build(ctx, task, goal)
	if err != nil {
		return fmt.Errorf("build execution plan: %w", err)
	}

	// A retry can observe a Task that already moved to ROUTING after the first
	// Activity completed. The immutable persisted plan is the idempotency record.
	if existing, getErr := a.Lineage.GetPlan(ctx, plan.ID, plan.Revision); getErr == nil {
		if existing.GoalRef != goal.ID {
			return fmt.Errorf("persisted plan goal lineage mismatch")
		}
		return nil
	} else if !errors.Is(getErr, repository.ErrNotFound) {
		return fmt.Errorf("load persisted execution plan: %w", getErr)
	}

	if task.Status != domain.TaskPlanning {
		return fmt.Errorf("%w: cannot create execution plan from %s", domain.ErrInvalidTaskTransition, task.Status)
	}
	persistedGoal, err := a.ensureGoal(ctx, goal)
	if err != nil {
		return err
	}
	if persistedGoal.TaskRef != task.ID || persistedGoal.Objective != goal.Objective {
		return fmt.Errorf("persisted goal does not match canonical Task intent")
	}
	plan.GoalRef = persistedGoal.ID
	if _, err := a.Lineage.CreatePlan(ctx, plan); err != nil {
		// Handle commit-before-ack/retry and concurrent duplicate Activities by
		// accepting an already persisted immutable revision with matching lineage.
		existing, getErr := a.Lineage.GetPlan(ctx, plan.ID, plan.Revision)
		if getErr == nil && existing.GoalRef == plan.GoalRef {
			return nil
		}
		return fmt.Errorf("persist execution plan: %w", err)
	}
	return nil
}

func (a DurablePlanningActivity) ensureGoal(ctx context.Context, goal execution.Goal) (execution.Goal, error) {
	existing, err := a.Lineage.GetGoal(ctx, goal.ID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return execution.Goal{}, fmt.Errorf("load persisted execution goal: %w", err)
	}
	created, createErr := a.Lineage.CreateGoal(ctx, goal)
	if createErr == nil {
		return created, nil
	}
	// Same commit-before-ack/concurrent duplicate rule as plans.
	existing, getErr := a.Lineage.GetGoal(ctx, goal.ID)
	if getErr == nil {
		return existing, nil
	}
	return execution.Goal{}, fmt.Errorf("persist execution goal: %w", createErr)
}

func validatePlanningStepInput(input StepInput) error {
	if strings.TrimSpace(input.TenantID) == "" || strings.TrimSpace(input.ProjectID) == "" ||
		strings.TrimSpace(input.TaskID) == "" || strings.TrimSpace(input.TraceID) == "" {
		return fmt.Errorf("planning step requires tenant, project, task and trace identity")
	}
	return nil
}

func executionWorkflowProjectContext(ctx context.Context, input StepInput) context.Context {
	return identity.WithPrincipal(ctx, identity.Principal{
		Type:         identity.PrincipalSystem,
		SubjectID:    "temporal-task-lifecycle",
		TenantID:     input.TenantID,
		ProjectID:    input.ProjectID,
		Capabilities: []string{identity.CapabilityTaskSystemAccess},
		AuthnMethod:  "internal_temporal",
		Issuer:       "aicloud",
		Purpose:      "task-lifecycle",
	})
}

// PlanningLifecycleActivities is a progressive migration adapter. It replaces
// only PlanStub while delegating Load/Transition/Route/Execute/Validate to the
// existing backend. This allows ECP-C2 to land without changing Temporal replay
// names or forcing an all-at-once lifecycle rewrite.
type PlanningLifecycleActivities struct {
	Base     LifecycleActivities
	Planning DurablePlanningActivity
}

func (a PlanningLifecycleActivities) LoadTask(ctx context.Context, input LoadTaskInput) (TaskSnapshot, error) {
	return a.Base.LoadTask(ctx, input)
}

func (a PlanningLifecycleActivities) TransitionTask(ctx context.Context, input TransitionTaskInput) (TaskSnapshot, error) {
	return a.Base.TransitionTask(ctx, input)
}

func (a PlanningLifecycleActivities) PlanStub(ctx context.Context, input StepInput) error {
	return a.Planning.Plan(ctx, input)
}

func (a PlanningLifecycleActivities) RouteStub(ctx context.Context, input StepInput) error {
	return a.Base.RouteStub(ctx, input)
}

func (a PlanningLifecycleActivities) ExecuteStub(ctx context.Context, input StepInput) error {
	return a.Base.ExecuteStub(ctx, input)
}

func (a PlanningLifecycleActivities) ValidateStub(ctx context.Context, input StepInput) error {
	return a.Base.ValidateStub(ctx, input)
}
