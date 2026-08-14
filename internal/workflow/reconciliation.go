package workflow

import (
	"context"
	"fmt"
	"strings"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/identity"
)

type ExecutionCondition string

const (
	ExecutionOpen             ExecutionCondition = "open"
	ExecutionClosedSuccessful ExecutionCondition = "closed_successful"
	ExecutionClosedFailed     ExecutionCondition = "closed_failed"
	ExecutionMissing          ExecutionCondition = "missing"
)

type ExecutionObservation struct {
	WorkflowID string
	RunID      string
	Condition  ExecutionCondition
}

func (o ExecutionObservation) Validate() error {
	if strings.TrimSpace(o.WorkflowID) == "" {
		return fmt.Errorf("workflow ID is required for reconciliation")
	}
	switch o.Condition {
	case ExecutionOpen, ExecutionClosedSuccessful, ExecutionClosedFailed, ExecutionMissing:
		return nil
	default:
		return fmt.Errorf("unsupported workflow execution condition %q", o.Condition)
	}
}

type ExecutionObserver interface {
	ObserveExecution(context.Context, string) (ExecutionObservation, error)
}

type ReconciliationClassification string

const (
	ReconciliationHealthy          ReconciliationClassification = "healthy"
	ReconciliationRecoveryRequired ReconciliationClassification = "recovery_required"
	ReconciliationEnsureCancel     ReconciliationClassification = "ensure_cancel"
	ReconciliationConsistent       ReconciliationClassification = "consistent"
)

type ReconciliationDecision struct {
	TenantID       string
	ProjectID      string
	TaskID         string
	TraceID        string
	TaskStatus     domain.TaskStatus
	WorkflowID     string
	WorkflowRunID  string
	Execution      ExecutionCondition
	Classification ReconciliationClassification
	Reason         string
}

type TaskReconciler struct {
	tasks    domain.TaskRepository
	observer ExecutionObserver
}

func NewTaskReconciler(tasks domain.TaskRepository, observer ExecutionObserver) (*TaskReconciler, error) {
	if tasks == nil {
		return nil, fmt.Errorf("Task repository is required")
	}
	if observer == nil {
		return nil, fmt.Errorf("workflow execution observer is required")
	}
	return &TaskReconciler{tasks: tasks, observer: observer}, nil
}

// Inspect is deliberately diagnosis-only. PostgreSQL remains the business
// source of truth: this method has no Task mutation or Outbox write capability.
func (r *TaskReconciler) Inspect(ctx context.Context, taskID string) (ReconciliationDecision, error) {
	if r == nil || r.tasks == nil || r.observer == nil {
		return ReconciliationDecision{}, fmt.Errorf("Task reconciler is not configured")
	}
	principal, err := identity.RequireProject(ctx)
	if err != nil {
		return ReconciliationDecision{}, err
	}
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return ReconciliationDecision{}, fmt.Errorf("Task ID is required")
	}
	task, err := r.tasks.Get(ctx, taskID)
	if err != nil {
		return ReconciliationDecision{}, err
	}
	if task.TenantID != principal.TenantID || task.ProjectID != principal.ProjectID {
		return ReconciliationDecision{}, fmt.Errorf("Task is not available in the current project scope")
	}
	workflowID, err := WorkflowID(task.ID)
	if err != nil {
		return ReconciliationDecision{}, err
	}
	observation, err := r.observer.ObserveExecution(ctx, workflowID)
	if err != nil {
		return ReconciliationDecision{}, err
	}
	if err := observation.Validate(); err != nil {
		return ReconciliationDecision{}, err
	}
	if observation.WorkflowID != workflowID {
		return ReconciliationDecision{}, fmt.Errorf("workflow observation does not match Task identity")
	}
	return classifyReconciliation(task, observation), nil
}

func classifyReconciliation(task domain.Task, observation ExecutionObservation) ReconciliationDecision {
	decision := ReconciliationDecision{
		TenantID:      task.TenantID,
		ProjectID:     task.ProjectID,
		TaskID:        task.ID,
		TraceID:       task.TraceID,
		TaskStatus:    task.Status,
		WorkflowID:    observation.WorkflowID,
		WorkflowRunID: observation.RunID,
		Execution:     observation.Condition,
	}

	if task.IsTerminal() {
		if observation.Condition == ExecutionOpen {
			decision.Classification = ReconciliationEnsureCancel
			decision.Reason = "terminal PostgreSQL Task still has an open workflow execution"
			return decision
		}
		decision.Classification = ReconciliationConsistent
		decision.Reason = "terminal PostgreSQL Task has no active workflow execution"
		return decision
	}

	if observation.Condition == ExecutionOpen {
		decision.Classification = ReconciliationHealthy
		decision.Reason = "nonterminal PostgreSQL Task has an open workflow execution"
		return decision
	}
	decision.Classification = ReconciliationRecoveryRequired
	decision.Reason = "nonterminal PostgreSQL Task has no active workflow execution; explicit recovery review is required"
	return decision
}
