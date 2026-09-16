package workflow

import (
	"context"
	"fmt"
	"testing"
	"time"

	execution "github.com/tommyxie2026-tech/aicloud/execution"
	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
)

type planningTaskStore struct {
	task domain.Task
}

func (s *planningTaskStore) List(context.Context) ([]domain.Task, error) {
	return []domain.Task{s.task}, nil
}
func (s *planningTaskStore) Get(_ context.Context, id string) (domain.Task, error) {
	if s.task.ID != id {
		return domain.Task{}, repository.ErrNotFound
	}
	return s.task, nil
}
func (s *planningTaskStore) Create(_ context.Context, task domain.Task) (domain.Task, error) {
	s.task = task
	return task, nil
}
func (s *planningTaskStore) Update(_ context.Context, task domain.Task) (domain.Task, error) {
	s.task = task
	return task, nil
}

type memoryPlanningLineage struct {
	goals       map[execution.GoalID]execution.Goal
	plans       map[string]execution.ExecutionPlan
	goalCreates int
	planCreates int
}

func newMemoryPlanningLineage() *memoryPlanningLineage {
	return &memoryPlanningLineage{
		goals: make(map[execution.GoalID]execution.Goal),
		plans: make(map[string]execution.ExecutionPlan),
	}
}

func (m *memoryPlanningLineage) CreateGoal(_ context.Context, goal execution.Goal) (execution.Goal, error) {
	if _, exists := m.goals[goal.ID]; exists {
		return execution.Goal{}, fmt.Errorf("goal already exists")
	}
	m.goalCreates++
	m.goals[goal.ID] = goal
	return goal, nil
}

func (m *memoryPlanningLineage) GetGoal(_ context.Context, id execution.GoalID) (execution.Goal, error) {
	goal, ok := m.goals[id]
	if !ok {
		return execution.Goal{}, repository.ErrNotFound
	}
	return goal, nil
}

func (m *memoryPlanningLineage) CreatePlan(_ context.Context, plan execution.ExecutionPlan) (execution.ExecutionPlan, error) {
	key := planningPlanKey(plan.ID, plan.Revision)
	if _, exists := m.plans[key]; exists {
		return execution.ExecutionPlan{}, fmt.Errorf("plan already exists")
	}
	m.planCreates++
	m.plans[key] = plan
	return plan, nil
}

func (m *memoryPlanningLineage) GetPlan(_ context.Context, id execution.PlanID, revision int64) (execution.ExecutionPlan, error) {
	plan, ok := m.plans[planningPlanKey(id, revision)]
	if !ok {
		return execution.ExecutionPlan{}, repository.ErrNotFound
	}
	return plan, nil
}

func planningPlanKey(id execution.PlanID, revision int64) string {
	return fmt.Sprintf("%s@%d", id, revision)
}

func TestDurablePlanningActivityCreatesTaskGoalAndPlanLineage(t *testing.T) {
	now := time.Date(2026, 9, 16, 7, 30, 0, 0, time.UTC)
	tasks := &planningTaskStore{task: domain.Task{
		ID: "task-1", TenantID: "tenant-a", ProjectID: "project-a",
		Input: "analyze incident", TraceID: "trace-1", Status: domain.TaskPlanning,
	}}
	lineage := newMemoryPlanningLineage()
	activity := DurablePlanningActivity{
		Tasks: tasks, Lineage: lineage,
		Builder: BaselineModelPlanBuilder{Now: func() time.Time { return now }},
	}
	input := StepInput{TenantID: "tenant-a", ProjectID: "project-a", TaskID: "task-1", TraceID: "trace-1"}
	if err := activity.Plan(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	if lineage.goalCreates != 1 || lineage.planCreates != 1 {
		t.Fatalf("expected one goal and plan create, got goals=%d plans=%d", lineage.goalCreates, lineage.planCreates)
	}
	goal := lineage.goals["goal/task-1/v1"]
	if goal.TaskRef != "task-1" || goal.Objective != "analyze incident" {
		t.Fatalf("unexpected goal lineage: %+v", goal)
	}
	plan := lineage.plans["plan/task-1/model/v1@1"]
	if plan.GoalRef != goal.ID || len(plan.Nodes) != 1 || plan.Nodes[0].Type != execution.NodeModel {
		t.Fatalf("unexpected baseline plan: %+v", plan)
	}
	if err := execution.ValidatePlan(plan); err != nil {
		t.Fatalf("persisted plan must satisfy execution contract: %v", err)
	}
}

func TestDurablePlanningActivityRetryUsesPersistedPlanAfterTaskAdvanced(t *testing.T) {
	tasks := &planningTaskStore{task: domain.Task{
		ID: "task-2", TenantID: "tenant-a", ProjectID: "project-a",
		Input: "analyze logs", TraceID: "trace-2", Status: domain.TaskPlanning,
	}}
	lineage := newMemoryPlanningLineage()
	activity := DurablePlanningActivity{Tasks: tasks, Lineage: lineage, Builder: BaselineModelPlanBuilder{}}
	input := StepInput{TenantID: "tenant-a", ProjectID: "project-a", TaskID: "task-2", TraceID: "trace-2"}
	if err := activity.Plan(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	tasks.task.Status = domain.TaskRouting
	if err := activity.Plan(context.Background(), input); err != nil {
		t.Fatalf("retry after Task advanced must replay persisted plan: %v", err)
	}
	if lineage.goalCreates != 1 || lineage.planCreates != 1 {
		t.Fatalf("retry must not duplicate lineage: goals=%d plans=%d", lineage.goalCreates, lineage.planCreates)
	}
}

func TestDurablePlanningActivityRejectsAdvancedTaskWithoutPlan(t *testing.T) {
	tasks := &planningTaskStore{task: domain.Task{
		ID: "task-3", TenantID: "tenant-a", ProjectID: "project-a",
		Input: "analyze logs", TraceID: "trace-3", Status: domain.TaskRouting,
	}}
	activity := DurablePlanningActivity{Tasks: tasks, Lineage: newMemoryPlanningLineage(), Builder: BaselineModelPlanBuilder{}}
	err := activity.Plan(context.Background(), StepInput{
		TenantID: "tenant-a", ProjectID: "project-a", TaskID: "task-3", TraceID: "trace-3",
	})
	if err == nil {
		t.Fatal("advanced Task without durable plan must fail closed")
	}
}

func TestDurablePlanningActivityRejectsScopeMismatch(t *testing.T) {
	tasks := &planningTaskStore{task: domain.Task{
		ID: "task-4", TenantID: "tenant-a", ProjectID: "project-a",
		Input: "analyze logs", TraceID: "trace-4", Status: domain.TaskPlanning,
	}}
	activity := DurablePlanningActivity{Tasks: tasks, Lineage: newMemoryPlanningLineage(), Builder: BaselineModelPlanBuilder{}}
	err := activity.Plan(context.Background(), StepInput{
		TenantID: "tenant-a", ProjectID: "project-b", TaskID: "task-4", TraceID: "trace-4",
	})
	if err == nil {
		t.Fatal("cross-project planning must fail")
	}
}
