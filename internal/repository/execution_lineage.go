package repository

import (
	"context"

	execution "github.com/tommyxie2026-tech/aicloud/execution"
)

// ExecutionLineageStore persists the immutable parent lineage above the
// existing node-runtime and attempt stores. Task business truth remains owned by
// internal/domain and task_events.
type ExecutionLineageStore interface {
	CreateGoal(context.Context, execution.Goal) (execution.Goal, error)
	GetGoal(context.Context, execution.GoalID) (execution.Goal, error)
	ListGoalsByTask(context.Context, string) ([]execution.Goal, error)

	CreatePlan(context.Context, execution.ExecutionPlan) (execution.ExecutionPlan, error)
	GetPlan(context.Context, execution.PlanID, int64) (execution.ExecutionPlan, error)
	ListPlansByTask(context.Context, string) ([]execution.ExecutionPlan, error)

	CreateExecution(context.Context, execution.Execution) (execution.Execution, error)
	GetExecution(context.Context, execution.ExecutionID) (execution.Execution, error)
	ListExecutionsByTask(context.Context, string) ([]execution.Execution, error)
}
