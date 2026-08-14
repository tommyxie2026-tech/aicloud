package workflow

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"go.temporal.io/sdk/testsuite"
)

func TestTaskWorkflowInputValidation(t *testing.T) {
	valid := TaskWorkflowInput{
		SchemaVersion: TaskWorkflowSchemaVersion,
		TenantID:      "tenant-a",
		ProjectID:     "project-a",
		TaskID:        "task-a",
		TraceID:       "trace-a",
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid workflow input rejected: %v", err)
	}

	cases := []struct {
		name   string
		mutate func(*TaskWorkflowInput)
	}{
		{name: "schema", mutate: func(i *TaskWorkflowInput) { i.SchemaVersion = 0 }},
		{name: "tenant", mutate: func(i *TaskWorkflowInput) { i.TenantID = "" }},
		{name: "project", mutate: func(i *TaskWorkflowInput) { i.ProjectID = "" }},
		{name: "task", mutate: func(i *TaskWorkflowInput) { i.TaskID = "" }},
		{name: "trace", mutate: func(i *TaskWorkflowInput) { i.TraceID = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := valid
			tc.mutate(&input)
			if err := input.Validate(); !errors.Is(err, ErrInvalidTaskWorkflowInput) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestTransitionOperationKeyIsDeterministic(t *testing.T) {
	first := TransitionOperationKey(" task-a ", domain.TaskPlanning)
	second := TransitionOperationKey("task-a", domain.TaskPlanning)
	if first != "task-a:transition:PLANNING:lifecycle-v1" || first != second {
		t.Fatalf("keys first=%q second=%q", first, second)
	}
}

func TestTaskLifecycleWorkflowHappyPath(t *testing.T) {
	activities := NewMemoryLifecycleActivities(newLifecycleTask(domain.TaskCreated))
	env := newLifecycleTestEnvironment(t, activities)
	env.ExecuteWorkflow(TaskLifecycleWorkflow, lifecycleInput())
	assertWorkflowSucceeded(t, env)

	var result TaskWorkflowResult
	if err := env.GetWorkflowResult(&result); err != nil {
		t.Fatal(err)
	}
	if result.AlreadyTerminal {
		t.Fatalf("normal completion was marked already terminal: %+v", result)
	}
	if result.ObservedStatus != domain.TaskCompleted {
		t.Fatalf("observed status=%s", result.ObservedStatus)
	}
	expectedSteps := []string{
		"PLANNING", "plan", "ROUTING", "route", "EXECUTING", "execute", "VALIDATING", "validate", "COMPLETED",
	}
	if !reflect.DeepEqual(result.Steps, expectedSteps) {
		t.Fatalf("steps=%v want=%v", result.Steps, expectedSteps)
	}
	task, ok := activities.Task("task-a")
	if !ok || task.Status != domain.TaskCompleted || task.Version != 6 {
		t.Fatalf("final Task=%+v ok=%v", task, ok)
	}
	if activities.OperationCount() != 5 {
		t.Fatalf("operation count=%d", activities.OperationCount())
	}
}

func TestTaskLifecycleWorkflowRetriesTransientStubWithoutDuplicateTransitions(t *testing.T) {
	activities := NewMemoryLifecycleActivities(newLifecycleTask(domain.TaskCreated))
	activities.SetTransientFailures(ActivityPlanStub, 1)
	env := newLifecycleTestEnvironment(t, activities)
	env.ExecuteWorkflow(TaskLifecycleWorkflow, lifecycleInput())
	assertWorkflowSucceeded(t, env)

	if activities.StepCalls(ActivityPlanStub) != 2 {
		t.Fatalf("plan calls=%d", activities.StepCalls(ActivityPlanStub))
	}
	if activities.OperationCount() != 5 {
		t.Fatalf("retry duplicated transition operations: %d", activities.OperationCount())
	}
}

func TestTaskLifecycleWorkflowShortCircuitsTerminalBeforeStart(t *testing.T) {
	task := newLifecycleTask(domain.TaskCancelled)
	task.Version = 2
	activities := NewMemoryLifecycleActivities(task)
	env := newLifecycleTestEnvironment(t, activities)
	env.ExecuteWorkflow(TaskLifecycleWorkflow, lifecycleInput())
	assertWorkflowSucceeded(t, env)

	var result TaskWorkflowResult
	if err := env.GetWorkflowResult(&result); err != nil {
		t.Fatal(err)
	}
	if !result.AlreadyTerminal || result.ObservedStatus != domain.TaskCancelled {
		t.Fatalf("unexpected terminal result: %+v", result)
	}
	if activities.OperationCount() != 0 || activities.StepCalls(ActivityPlanStub) != 0 {
		t.Fatal("terminal Task produced lifecycle side effects")
	}
}

func TestTaskLifecycleWorkflowShortCircuitsTerminalRace(t *testing.T) {
	activities := NewMemoryLifecycleActivities(newLifecycleTask(domain.TaskCreated))
	activities.SetTerminalOnStep(ActivityPlanStub, domain.TaskCancelled)
	env := newLifecycleTestEnvironment(t, activities)
	env.ExecuteWorkflow(TaskLifecycleWorkflow, lifecycleInput())
	assertWorkflowSucceeded(t, env)

	var result TaskWorkflowResult
	if err := env.GetWorkflowResult(&result); err != nil {
		t.Fatal(err)
	}
	if !result.AlreadyTerminal || result.ObservedStatus != domain.TaskCancelled {
		t.Fatalf("unexpected race result: %+v", result)
	}
	if activities.OperationCount() != 1 {
		t.Fatalf("only CREATED->PLANNING should commit before race, operations=%d", activities.OperationCount())
	}
}

func TestTaskLifecycleWorkflowReloadsAfterStaleVersion(t *testing.T) {
	activities := NewMemoryLifecycleActivities(newLifecycleTask(domain.TaskCreated))
	activities.SetStaleOnce(TransitionOperationKey("task-a", domain.TaskPlanning))
	env := newLifecycleTestEnvironment(t, activities)
	env.ExecuteWorkflow(TaskLifecycleWorkflow, lifecycleInput())
	assertWorkflowSucceeded(t, env)

	var result TaskWorkflowResult
	if err := env.GetWorkflowResult(&result); err != nil {
		t.Fatal(err)
	}
	if result.AlreadyTerminal || result.ObservedStatus != domain.TaskCompleted {
		t.Fatalf("unexpected stale-version result: %+v", result)
	}
	if activities.OperationCount() != 5 {
		t.Fatalf("operation count=%d", activities.OperationCount())
	}
}

func newLifecycleTestEnvironment(t *testing.T, activities LifecycleActivities) *testsuite.TestWorkflowEnvironment {
	t.Helper()
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	if err := RegisterLifecycle(env, activities); err != nil {
		t.Fatalf("register lifecycle: %v", err)
	}
	return env
}

func assertWorkflowSucceeded(t *testing.T, env *testsuite.TestWorkflowEnvironment) {
	t.Helper()
	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow did not complete")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("workflow failed: %v", err)
	}
}

func lifecycleInput() TaskWorkflowInput {
	return TaskWorkflowInput{
		SchemaVersion: TaskWorkflowSchemaVersion,
		TenantID:      "tenant-a",
		ProjectID:     "project-a",
		TaskID:        "task-a",
		TraceID:       "trace-a",
	}
}

func newLifecycleTask(status domain.TaskStatus) domain.Task {
	now := time.Now().UTC()
	completedAt := (*time.Time)(nil)
	if isTerminalStatus(status) {
		completedAt = &now
	}
	return domain.Task{
		ID:          "task-a",
		TenantID:    "tenant-a",
		ProjectID:   "project-a",
		CreatedBy:   "user-a",
		AgentID:     "agent-a",
		Input:       "test lifecycle",
		Status:      status,
		Version:     1,
		TraceID:     "trace-a",
		Currency:    "USD",
		CreatedAt:   now,
		UpdatedAt:   now,
		CompletedAt: completedAt,
	}
}
