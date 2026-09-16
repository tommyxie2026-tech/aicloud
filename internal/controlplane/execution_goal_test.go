package controlplane

import (
	"testing"

	execution "github.com/tommyxie2026-tech/aicloud/execution"
	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

func TestGoalSnapshotFromTaskPreservesTaskLineageAndDefaultsFailClosed(t *testing.T) {
	task := domain.Task{ID: "task-1", Input: "diagnose the incident"}
	goal, err := GoalSnapshotFromTask(task, GoalSnapshotOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if goal.ID != "goal/task-1/v1" {
		t.Fatalf("unexpected goal id %q", goal.ID)
	}
	if goal.TaskRef != task.ID || goal.Objective != task.Input {
		t.Fatalf("task lineage not preserved: %+v", goal)
	}
	if goal.Constraints.ProductionWriteAllowed {
		t.Fatal("production writes must be fail-closed by default")
	}
}

func TestGoalSnapshotFromTaskCopiesExecutionConstraints(t *testing.T) {
	task := domain.Task{ID: "task-2", Input: "analyze logs"}
	options := GoalSnapshotOptions{
		GoalID: "goal-custom",
		AcceptanceCriteria: []execution.AcceptanceCriterion{{
			ID: "evidence", Text: "cite evidence", Required: true,
		}},
		Constraints: execution.GoalConstraints{
			AllowedDataScopes: []string{"logs:read"},
		},
		Budget: execution.BudgetLimit{MaxCost: 5, MaxNodeAttempts: 4},
	}
	goal, err := GoalSnapshotFromTask(task, options)
	if err != nil {
		t.Fatal(err)
	}
	if goal.ID != options.GoalID || len(goal.AcceptanceCriteria) != 1 || goal.Budget.MaxCost != 5 {
		t.Fatalf("execution options not preserved: %+v", goal)
	}
	options.Constraints.AllowedDataScopes[0] = "mutated"
	if goal.Constraints.AllowedDataScopes[0] != "logs:read" {
		t.Fatal("goal snapshot must not alias caller data scopes")
	}
}

func TestGoalSnapshotFromTaskRejectsMissingIntent(t *testing.T) {
	if _, err := GoalSnapshotFromTask(domain.Task{ID: "task-3"}, GoalSnapshotOptions{}); err == nil {
		t.Fatal("expected missing task intent error")
	}
}
