package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/identity"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
	"go.temporal.io/sdk/temporal"
)

type fakeActivityTasks struct {
	domain.TaskRepository
	task      domain.Task
	err       error
	getCalls  int
	principal identity.Principal
}

func (f *fakeActivityTasks) Get(ctx context.Context, id string) (domain.Task, error) {
	f.getCalls++
	principal, err := identity.RequireProject(ctx)
	if err != nil {
		return domain.Task{}, err
	}
	f.principal = principal
	if f.err != nil {
		return domain.Task{}, f.err
	}
	if f.task.ID != id {
		return domain.Task{}, repository.ErrNotFound
	}
	return f.task, nil
}

type fakeActivityCommands struct {
	repository.TaskCommandStore
	resolveRecord domain.IdempotencyRecord
	resolveFound  bool
	resolveErr    error
	resolveCalls  int
	resolveLookup repository.IdempotencyLookup
	commitResult  repository.TaskCommandCommitResult
	commitErr     error
	commitCalls   int
	commit        repository.TaskCommandCommit
	principal     identity.Principal
}

func (f *fakeActivityCommands) ResolveIdempotency(ctx context.Context, lookup repository.IdempotencyLookup) (domain.IdempotencyRecord, bool, error) {
	f.resolveCalls++
	principal, err := identity.RequireProject(ctx)
	if err != nil {
		return domain.IdempotencyRecord{}, false, err
	}
	f.principal = principal
	f.resolveLookup = lookup
	return f.resolveRecord, f.resolveFound, f.resolveErr
}

func (f *fakeActivityCommands) CommitTransition(ctx context.Context, commit repository.TaskCommandCommit) (repository.TaskCommandCommitResult, error) {
	f.commitCalls++
	principal, err := identity.RequireProject(ctx)
	if err != nil {
		return repository.TaskCommandCommitResult{}, err
	}
	f.principal = principal
	f.commit = commit
	return f.commitResult, f.commitErr
}

func TestNewPostgresLifecycleActivitiesRequiresTrustedDependencies(t *testing.T) {
	tasks := &fakeActivityTasks{}
	commands := &fakeActivityCommands{}
	trust := ActivityTrustConfig{Namespace: "default", TaskQueue: "aicloud-task-v1"}
	info := func(context.Context) ActivityExecutionInfo { return validActivityExecutionInfo(ActivityLoadTask) }

	if _, err := newPostgresLifecycleActivities(nil, commands, trust, info); err == nil {
		t.Fatal("nil Task repository accepted")
	}
	if _, err := newPostgresLifecycleActivities(tasks, nil, trust, info); err == nil {
		t.Fatal("nil Task command store accepted")
	}
	if _, err := newPostgresLifecycleActivities(tasks, commands, trust, nil); err == nil {
		t.Fatal("nil execution-info provider accepted")
	}
	if _, err := newPostgresLifecycleActivities(tasks, commands, ActivityTrustConfig{TaskQueue: "aicloud-task-v1"}, info); err == nil {
		t.Fatal("missing trusted namespace accepted")
	}
	if _, err := newPostgresLifecycleActivities(tasks, commands, ActivityTrustConfig{Namespace: "default"}, info); err == nil {
		t.Fatal("missing trusted Task Queue accepted")
	}
}

func TestPostgresLoadTaskAttestsThenUsesScopedServiceAccount(t *testing.T) {
	task := postgresActivityTask(domain.TaskCreated, 1)
	tasks := &fakeActivityTasks{task: task}
	commands := &fakeActivityCommands{}
	activities := mustPostgresActivities(t, tasks, commands, ActivityLoadTask)

	snapshot, err := activities.LoadTask(context.Background(), LoadTaskInput{
		TenantID: "tenant-a", ProjectID: "project-a", TaskID: "task-a", TraceID: "trace-a",
	})
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.TaskID != task.ID || snapshot.TraceID != task.TraceID || snapshot.Status != task.Status || snapshot.Version != task.Version {
		t.Fatalf("unexpected snapshot: %+v", snapshot)
	}
	if tasks.principal.Type != identity.PrincipalServiceAccount || tasks.principal.SubjectID != WorkflowWorkerSubject {
		t.Fatalf("unexpected workload principal: %+v", tasks.principal)
	}
	if tasks.principal.TenantID != "tenant-a" || tasks.principal.ProjectID != "project-a" || tasks.principal.AuthnMethod != WorkflowWorkerAuthnMethod {
		t.Fatalf("unexpected scoped workload principal: %+v", tasks.principal)
	}
}

func TestPostgresLoadTaskRejectsForgedExecutionBeforeRepositoryRead(t *testing.T) {
	tasks := &fakeActivityTasks{task: postgresActivityTask(domain.TaskCreated, 1)}
	commands := &fakeActivityCommands{}
	info := validActivityExecutionInfo(ActivityLoadTask)
	info.WorkflowID = "task/task-other"
	activities := mustPostgresActivitiesWithInfo(t, tasks, commands, info)

	_, err := activities.LoadTask(context.Background(), LoadTaskInput{
		TenantID: "tenant-a", ProjectID: "project-a", TaskID: "task-a", TraceID: "trace-a",
	})
	assertApplicationErrorType(t, err, ErrorTypeActivityAttestation)
	if tasks.getCalls != 0 {
		t.Fatalf("repository read occurred before attestation, calls=%d", tasks.getCalls)
	}
}

func TestPostgresActivityAttestationRejectsBoundaryMismatch(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*ActivityExecutionInfo)
	}{
		{name: "workflow type", mutate: func(info *ActivityExecutionInfo) { info.WorkflowType = "other-workflow" }},
		{name: "namespace", mutate: func(info *ActivityExecutionInfo) { info.Namespace = "other" }},
		{name: "task queue", mutate: func(info *ActivityExecutionInfo) { info.TaskQueue = "other" }},
		{name: "activity type", mutate: func(info *ActivityExecutionInfo) { info.ActivityType = ActivityTransition }},
		{name: "run id", mutate: func(info *ActivityExecutionInfo) { info.WorkflowRunID = "" }},
		{name: "activity id", mutate: func(info *ActivityExecutionInfo) { info.ActivityID = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tasks := &fakeActivityTasks{task: postgresActivityTask(domain.TaskCreated, 1)}
			commands := &fakeActivityCommands{}
			info := validActivityExecutionInfo(ActivityLoadTask)
			tc.mutate(&info)
			activities := mustPostgresActivitiesWithInfo(t, tasks, commands, info)
			_, err := activities.LoadTask(context.Background(), LoadTaskInput{
				TenantID: "tenant-a", ProjectID: "project-a", TaskID: "task-a", TraceID: "trace-a",
			})
			assertApplicationErrorType(t, err, ErrorTypeActivityAttestation)
			if tasks.getCalls != 0 {
				t.Fatalf("repository read occurred for invalid attestation, calls=%d", tasks.getCalls)
			}
		})
	}
}

func TestPostgresLoadTaskRejectsCorrelationMismatch(t *testing.T) {
	task := postgresActivityTask(domain.TaskCreated, 1)
	task.TraceID = "trace-other"
	tasks := &fakeActivityTasks{task: task}
	activities := mustPostgresActivities(t, tasks, &fakeActivityCommands{}, ActivityLoadTask)

	_, err := activities.LoadTask(context.Background(), LoadTaskInput{
		TenantID: "tenant-a", ProjectID: "project-a", TaskID: "task-a", TraceID: "trace-a",
	})
	assertApplicationErrorType(t, err, ErrorTypeActivityScope)
}

func TestPostgresTransitionCommitsCanonicalEventAndIdempotencyEvidence(t *testing.T) {
	task := postgresActivityTask(domain.TaskCreated, 1)
	tasks := &fakeActivityTasks{task: task}
	commands := &fakeActivityCommands{}
	activities := mustPostgresActivities(t, tasks, commands, ActivityTransition)
	fixedNow := time.Date(2026, 8, 14, 3, 0, 0, 0, time.UTC)
	activities.now = func() time.Time { return fixedNow }

	committed := task
	if _, err := committed.Transition(domain.TaskTransitionCommand{
		To: domain.TaskPlanning, Actor: WorkflowWorkerSubject, Cause: TaskLifecycleVersion, At: fixedNow,
	}); err != nil {
		t.Fatal(err)
	}
	committed.Version = 2
	commands.commitResult = repository.TaskCommandCommitResult{Task: committed}

	input := transitionInput(domain.TaskPlanning, 1)
	snapshot, err := activities.TransitionTask(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Status != domain.TaskPlanning || snapshot.Version != 2 {
		t.Fatalf("unexpected committed snapshot: %+v", snapshot)
	}
	if commands.resolveCalls != 1 || commands.commitCalls != 1 || tasks.getCalls != 1 {
		t.Fatalf("resolve=%d get=%d commit=%d", commands.resolveCalls, tasks.getCalls, commands.commitCalls)
	}
	if commands.commit.ExpectedVersion != 1 || commands.commit.Event.EventType != "TaskPlanningStarted" {
		t.Fatalf("unexpected commit: %+v", commands.commit)
	}
	if commands.commit.Event.Actor.PrincipalType != string(identity.PrincipalServiceAccount) || commands.commit.Event.Actor.SubjectID != WorkflowWorkerSubject {
		t.Fatalf("unexpected event actor: %+v", commands.commit.Event.Actor)
	}
	if commands.commit.Idempotency.Operation != ActivityTransitionOperationV1 || commands.commit.Idempotency.Key != input.OperationKey {
		t.Fatalf("unexpected idempotency identity: %+v", commands.commit.Idempotency)
	}
	if commands.commit.Idempotency.RequestDigest == "" || commands.commit.Idempotency.ResponseDigest == "" || len(commands.commit.Idempotency.ResponsePayload) == 0 {
		t.Fatalf("incomplete idempotency evidence: %+v", commands.commit.Idempotency)
	}
	if commands.commit.Idempotency.ExpiresAt.Sub(fixedNow) != 30*24*time.Hour {
		t.Fatalf("idempotency retention=%s", commands.commit.Idempotency.ExpiresAt.Sub(fixedNow))
	}
	var replay TaskSnapshot
	if err := json.Unmarshal(commands.commit.Idempotency.ResponsePayload, &replay); err != nil {
		t.Fatal(err)
	}
	if replay.Status != domain.TaskPlanning || replay.Version != 2 || replay.TaskID != "task-a" {
		t.Fatalf("unexpected replay payload: %+v", replay)
	}
	var eventPayload map[string]any
	if err := json.Unmarshal(commands.commit.Event.Payload, &eventPayload); err != nil {
		t.Fatal(err)
	}
	if eventPayload["fromStatus"] != string(domain.TaskCreated) || eventPayload["toStatus"] != string(domain.TaskPlanning) || eventPayload["workflowId"] != "task/task-a" {
		t.Fatalf("unexpected TaskEvent payload: %#v", eventPayload)
	}
}

func TestPostgresTransitionLostResponseRetryResolvesIdempotencyBeforeOCCRead(t *testing.T) {
	current := postgresActivityTask(domain.TaskPlanning, 2)
	tasks := &fakeActivityTasks{task: current}
	stored := TaskSnapshot{TaskID: "task-a", TraceID: "trace-a", Status: domain.TaskPlanning, Version: 2}
	storedPayload, _ := json.Marshal(stored)
	commands := &fakeActivityCommands{
		resolveFound: true,
		resolveRecord: domain.IdempotencyRecord{
			TenantID: "tenant-a", ProjectID: "project-a", Operation: ActivityTransitionOperationV1,
			Key: TransitionOperationKey("task-a", domain.TaskPlanning), ResponsePayload: storedPayload,
		},
	}
	activities := mustPostgresActivities(t, tasks, commands, ActivityTransition)

	result, err := activities.TransitionTask(context.Background(), transitionInput(domain.TaskPlanning, 1))
	if err != nil {
		t.Fatal(err)
	}
	if result != stored {
		t.Fatalf("replay result=%+v want=%+v", result, stored)
	}
	if commands.resolveCalls != 1 {
		t.Fatalf("resolve calls=%d", commands.resolveCalls)
	}
	if tasks.getCalls != 0 {
		t.Fatalf("Task OCC read happened before idempotency replay, getCalls=%d", tasks.getCalls)
	}
	if commands.commitCalls != 0 {
		t.Fatalf("replayed transition committed again, commitCalls=%d", commands.commitCalls)
	}
}

func TestPostgresTransitionIdempotencyConflictIsNonRetryable(t *testing.T) {
	tasks := &fakeActivityTasks{task: postgresActivityTask(domain.TaskCreated, 1)}
	commands := &fakeActivityCommands{resolveErr: repository.ErrIdempotencyConflict}
	activities := mustPostgresActivities(t, tasks, commands, ActivityTransition)

	_, err := activities.TransitionTask(context.Background(), transitionInput(domain.TaskPlanning, 1))
	assertApplicationErrorType(t, err, ErrorTypeActivityIdempotency)
	if tasks.getCalls != 0 || commands.commitCalls != 0 {
		t.Fatalf("idempotency conflict touched Task state: get=%d commit=%d", tasks.getCalls, commands.commitCalls)
	}
}

func TestPostgresTransitionReturnsTypedStaleVersion(t *testing.T) {
	tasks := &fakeActivityTasks{task: postgresActivityTask(domain.TaskCreated, 2)}
	commands := &fakeActivityCommands{}
	activities := mustPostgresActivities(t, tasks, commands, ActivityTransition)

	_, err := activities.TransitionTask(context.Background(), transitionInput(domain.TaskPlanning, 1))
	assertApplicationErrorType(t, err, ErrorTypeStaleTaskVersion)
	if commands.commitCalls != 0 {
		t.Fatalf("stale transition committed, calls=%d", commands.commitCalls)
	}
}

func TestPostgresTransitionTerminalTaskShortCircuitsWithoutCommit(t *testing.T) {
	tasks := &fakeActivityTasks{task: postgresActivityTask(domain.TaskCancelled, 2)}
	commands := &fakeActivityCommands{}
	activities := mustPostgresActivities(t, tasks, commands, ActivityTransition)

	result, err := activities.TransitionTask(context.Background(), transitionInput(domain.TaskPlanning, 1))
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != domain.TaskCancelled || !result.Terminal {
		t.Fatalf("unexpected terminal snapshot: %+v", result)
	}
	if commands.commitCalls != 0 {
		t.Fatalf("terminal Task committed again, calls=%d", commands.commitCalls)
	}
}

func TestPostgresTransitionInvalidAggregateTransitionIsNonRetryable(t *testing.T) {
	tasks := &fakeActivityTasks{task: postgresActivityTask(domain.TaskCreated, 1)}
	commands := &fakeActivityCommands{}
	activities := mustPostgresActivities(t, tasks, commands, ActivityTransition)

	_, err := activities.TransitionTask(context.Background(), transitionInput(domain.TaskCompleted, 1))
	assertApplicationErrorType(t, err, ErrorTypeActivityTransition)
	if commands.commitCalls != 0 {
		t.Fatalf("invalid aggregate transition committed, calls=%d", commands.commitCalls)
	}
}

func TestPostgresActivityRetryableRepositoryErrorPropagates(t *testing.T) {
	sentinel := errors.New("database temporarily unavailable")
	tasks := &fakeActivityTasks{task: postgresActivityTask(domain.TaskCreated, 1), err: sentinel}
	commands := &fakeActivityCommands{}
	activities := mustPostgresActivities(t, tasks, commands, ActivityTransition)

	_, err := activities.TransitionTask(context.Background(), transitionInput(domain.TaskPlanning, 1))
	if !errors.Is(err, sentinel) {
		t.Fatalf("error=%v", err)
	}
}

func mustPostgresActivities(t *testing.T, tasks domain.TaskRepository, commands repository.TaskCommandStore, activityType string) *PostgresLifecycleActivities {
	t.Helper()
	return mustPostgresActivitiesWithInfo(t, tasks, commands, validActivityExecutionInfo(activityType))
}

func mustPostgresActivitiesWithInfo(t *testing.T, tasks domain.TaskRepository, commands repository.TaskCommandStore, info ActivityExecutionInfo) *PostgresLifecycleActivities {
	t.Helper()
	activities, err := newPostgresLifecycleActivities(
		tasks,
		commands,
		ActivityTrustConfig{Namespace: "default", TaskQueue: "aicloud-task-v1"},
		func(context.Context) ActivityExecutionInfo { return info },
	)
	if err != nil {
		t.Fatal(err)
	}
	return activities
}

func validActivityExecutionInfo(activityType string) ActivityExecutionInfo {
	return ActivityExecutionInfo{
		WorkflowID: "task/task-a", WorkflowRunID: "run-a", WorkflowType: TaskExecutionWorkflowType,
		Namespace: "default", TaskQueue: "aicloud-task-v1", ActivityID: "activity-a", ActivityType: activityType, Attempt: 1,
	}
}

func transitionInput(to domain.TaskStatus, expectedVersion int64) TransitionTaskInput {
	return TransitionTaskInput{
		TenantID: "tenant-a", ProjectID: "project-a", TaskID: "task-a", TraceID: "trace-a",
		ExpectedVersion: expectedVersion, To: to, Cause: TaskLifecycleVersion,
		OperationKey: TransitionOperationKey("task-a", to),
	}
}

func postgresActivityTask(status domain.TaskStatus, version int64) domain.Task {
	now := time.Date(2026, 8, 14, 1, 0, 0, 0, time.UTC)
	completedAt := (*time.Time)(nil)
	if isTerminalStatus(status) {
		completedAt = &now
	}
	return domain.Task{
		ID: "task-a", TenantID: "tenant-a", ProjectID: "project-a", CreatedBy: "user-a",
		AgentID: "agent-a", Input: "test", Status: status, Version: version, TraceID: "trace-a",
		Currency: "USD", CreatedAt: now, UpdatedAt: now, CompletedAt: completedAt,
	}
}

func assertApplicationErrorType(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected application error type %s", want)
	}
	var applicationErr *temporal.ApplicationError
	if !errors.As(err, &applicationErr) {
		t.Fatalf("error is not Temporal ApplicationError: %T %v", err, err)
	}
	if applicationErr.Type() != want {
		t.Fatalf("error type=%q want=%q err=%v", applicationErr.Type(), want, err)
	}
	if !applicationErr.NonRetryable() {
		t.Fatalf("error type=%q must be non-retryable", want)
	}
}
