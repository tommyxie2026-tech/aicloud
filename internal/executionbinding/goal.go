package executionbinding

import (
	"fmt"
	"strings"

	execution "github.com/tommyxie2026-tech/aicloud/execution"
	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

// GoalSnapshotOptions adds execution-only constraints without changing the
// canonical Task aggregate. Defaults remain fail-closed for production writes.
type GoalSnapshotOptions struct {
	GoalID             execution.GoalID
	AcceptanceCriteria []execution.AcceptanceCriterion
	Constraints        execution.GoalConstraints
	Budget             execution.BudgetLimit
}

// GoalSnapshotFromTask converts canonical Task intent into an immutable
// execution.Goal. Goal is an execution snapshot, not a second business Task.
func GoalSnapshotFromTask(task domain.Task, options GoalSnapshotOptions) (execution.Goal, error) {
	if strings.TrimSpace(task.ID) == "" {
		return execution.Goal{}, fmt.Errorf("task id is required")
	}
	objective := strings.TrimSpace(task.Input)
	if objective == "" {
		return execution.Goal{}, fmt.Errorf("task input is required to build goal snapshot")
	}
	goalID := options.GoalID
	if goalID == "" {
		goalID = execution.GoalID("goal/" + task.ID + "/v1")
	}
	return execution.Goal{
		ID:                 goalID,
		TaskRef:            task.ID,
		Objective:          objective,
		AcceptanceCriteria: append([]execution.AcceptanceCriterion(nil), options.AcceptanceCriteria...),
		Constraints: execution.GoalConstraints{
			ProductionWriteAllowed: options.Constraints.ProductionWriteAllowed,
			AllowedDataScopes:      append([]string(nil), options.Constraints.AllowedDataScopes...),
		},
		Budget: options.Budget,
	}, nil
}
