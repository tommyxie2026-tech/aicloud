//go:build integration

package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/identity"
	outboxpkg "github.com/tommyxie2026-tech/aicloud/internal/outbox"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

type d8StartOutcome struct {
	result temporalStartResult
	err    error
}

type d8TemporalBackend struct {
	mu         sync.Mutex
	starts     []temporalStartRequest
	outcomes   []d8StartOutcome
	cancelIDs  []string
	cancelErrs []error
}

func (b *d8TemporalBackend) Start(_ context.Context, request temporalStartRequest) (temporalStartResult, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.starts = append(b.starts, request)
	index := len(b.starts) - 1
	if index < len(b.outcomes) {
		return b.outcomes[index].result, b.outcomes[index].err
	}
	return temporalStartResult{RunID: fmt.Sprintf("run-%d", index+1)}, nil
}

func (b *d8TemporalBackend) Cancel(_ context.Context, workflowID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.cancelIDs = append(b.cancelIDs, workflowID)
	index := len(b.cancelIDs) - 1
	if index < len(b.cancelErrs) {
		return b.cancelErrs[index]
	}
	return nil
}

func TestS3DStartOutboxRetriesThenNormalizesAlreadyStarted(t *testing.T) {
	db, ctx := openS3DWorkflowDB(t)
	defer db.Close()
	resetS3DWorkflowSchema(t, ctx, db)
	defer resetS3DWorkflowSchema(t, context.Background(), db)
	createS3DWorkflowSchema(t, ctx, db)

	projectCtx := s3dProjectContext(ctx)
	commands := repository.NewScopedPostgresTaskCommands(db)
	task := createS3DDurableTask(t, projectCtx, commands, "task-d8-start", "trace-d8-start")

	backend := &d8TemporalBackend{outcomes: []d8StartOutcome{
		{err: errors.New("Temporal temporarily unavailable")},
		{result: temporalStartResult{RunID: "run-existing", AlreadyStarted: true}},
	}}
	engine, err := newTemporalEngineWithBackend(backend, "aicloud-task-v1")
	if err != nil {
		t.Fatal(err)
	}
	adapter := NewStartDeliveryAdapter(engine)
	store := repository.NewScopedPostgresOutbox(db)
	dispatcher, err := outboxpkg.NewDispatcher(store, "d8-dispatcher", time.Second, 3, 5*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if err := dispatcher.Register(DestinationWorkflowStart, adapter); err != nil {
		t.Fatal(err)
	}

	first, err := dispatcher.DispatchOnce(projectCtx, 10)
	if err != nil {
		t.Fatalf("first dispatch returned infrastructure error: %v", err)
	}
	if first.Leased != 1 || first.Retried != 1 || first.Delivered != 0 {
		t.Fatalf("unexpected first dispatch result: %+v", first)
	}

	var status domain.OutboxStatus
	var attempts int
	var availableAt time.Time
	var lastError string
	if err := db.QueryRowContext(ctx, `
SELECT status, attempts, available_at, COALESCE(last_error,'')
FROM outbox_messages
WHERE task_id=$1 AND destination=$2`, task.ID, DestinationWorkflowStart).Scan(
		&status, &attempts, &availableAt, &lastError,
	); err != nil {
		t.Fatalf("read retry Outbox: %v", err)
	}
	if status != domain.OutboxPending || attempts != 1 || !strings.Contains(lastError, "Temporal temporarily unavailable") {
		t.Fatalf("retry evidence status=%s attempts=%d lastError=%q", status, attempts, lastError)
	}
	if wait := time.Until(availableAt) + 20*time.Millisecond; wait > 0 {
		time.Sleep(wait)
	}

	second, err := dispatcher.DispatchOnce(projectCtx, 10)
	if err != nil {
		t.Fatalf("second dispatch returned error: %v", err)
	}
	if second.Leased != 1 || second.Delivered != 1 || second.Retried != 0 {
		t.Fatalf("unexpected second dispatch result: %+v", second)
	}

	backend.mu.Lock()
	starts := append([]temporalStartRequest(nil), backend.starts...)
	backend.mu.Unlock()
	if len(starts) != 2 {
		t.Fatalf("Temporal start attempts=%d want=2", len(starts))
	}
	for index, request := range starts {
		if request.WorkflowID != "task/"+task.ID || request.WorkflowType != TaskExecutionWorkflowType || request.TaskQueue != "aicloud-task-v1" {
			t.Fatalf("start[%d] identity drifted: %+v", index, request)
		}
		if request.Input.TaskID != task.ID || request.Input.TraceID != task.TraceID || request.Input.TenantID != "tenant-a" || request.Input.ProjectID != "project-a" {
			t.Fatalf("start[%d] scope drifted: %+v", index, request.Input)
		}
	}

	if err := db.QueryRowContext(ctx, `SELECT status, attempts FROM outbox_messages WHERE task_id=$1 AND destination=$2`, task.ID, DestinationWorkflowStart).Scan(&status, &attempts); err != nil {
		t.Fatalf("read delivered Outbox: %v", err)
	}
	if status != domain.OutboxDelivered || attempts != 2 {
		t.Fatalf("delivered Outbox status=%s attempts=%d", status, attempts)
	}
	var taskStatus domain.TaskStatus
	var taskVersion int64
	if err := db.QueryRowContext(ctx, `SELECT status, version FROM tasks WHERE id=$1`, task.ID).Scan(&taskStatus, &taskVersion); err != nil {
		t.Fatalf("read Task after start delivery: %v", err)
	}
	if taskStatus != domain.TaskCreated || taskVersion != 1 {
		t.Fatalf("Temporal starter delivery mutated business Task: status=%s version=%d", taskStatus, taskVersion)
	}
}

func TestS3DActivityLostResponseRetryDoesNotDuplicateTaskEvent(t *testing.T) {
	db, ctx := openS3DWorkflowDB(t)
	defer db.Close()
	resetS3DWorkflowSchema(t, ctx, db)
	defer resetS3DWorkflowSchema(t, context.Background(), db)
	createS3DWorkflowSchema(t, ctx, db)

	commands := repository.NewScopedPostgresTaskCommands(db)
	task := createS3DDurableTask(t, s3dProjectContext(ctx), commands, "task-d8-replay", "trace-d8-replay")
	tasks := repository.NewScopedPostgresTasks(db)
	activities, err := newPostgresLifecycleActivities(
		tasks,
		commands,
		ActivityTrustConfig{Namespace: "d8", TaskQueue: "aicloud-task-v1"},
		func(context.Context) ActivityExecutionInfo {
			return d8ExecutionInfo(task.ID, ActivityTransition)
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	input := TransitionTaskInput{
		TenantID: "tenant-a", ProjectID: "project-a", TaskID: task.ID, TraceID: task.TraceID,
		ExpectedVersion: 1, To: domain.TaskPlanning, Cause: TaskLifecycleVersion,
		OperationKey: TransitionOperationKey(task.ID, domain.TaskPlanning),
	}
	first, err := activities.TransitionTask(context.Background(), input)
	if err != nil {
		t.Fatalf("first Activity transition: %v", err)
	}
	if first.Status != domain.TaskPlanning || first.Version != 2 {
		t.Fatalf("first transition result=%+v", first)
	}

	// Simulate the worker losing the first Activity response after PostgreSQL
	// committed. The same Temporal Activity input is then retried.
	second, err := activities.TransitionTask(context.Background(), input)
	if err != nil {
		t.Fatalf("lost-response retry: %v", err)
	}
	if second != first {
		t.Fatalf("idempotent replay changed snapshot: first=%+v second=%+v", first, second)
	}

	var version int64
	var status domain.TaskStatus
	if err := db.QueryRowContext(ctx, `SELECT status, version FROM tasks WHERE id=$1`, task.ID).Scan(&status, &version); err != nil {
		t.Fatal(err)
	}
	if status != domain.TaskPlanning || version != 2 {
		t.Fatalf("replayed Activity rewrote Task: status=%s version=%d", status, version)
	}
	var planningEvents int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM task_events WHERE task_id=$1 AND event_type='TaskPlanningStarted'`, task.ID).Scan(&planningEvents); err != nil {
		t.Fatal(err)
	}
	if planningEvents != 1 {
		t.Fatalf("lost-response retry produced %d TaskPlanningStarted events", planningEvents)
	}
	var operationRecords int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM idempotency_records WHERE operation=$1 AND resource_id=$2`, ActivityTransitionOperationV1, task.ID).Scan(&operationRecords); err != nil {
		t.Fatal(err)
	}
	if operationRecords != 1 {
		t.Fatalf("lost-response retry produced %d Activity idempotency records", operationRecords)
	}
}

func TestS3DTemporalWorkflowCommitsPostgresPlanningThenFailsClosed(t *testing.T) {
	db, ctx := openS3DWorkflowDB(t)
	defer db.Close()
	resetS3DWorkflowSchema(t, ctx, db)
	defer resetS3DWorkflowSchema(t, context.Background(), db)
	createS3DWorkflowSchema(t, ctx, db)

	commands := repository.NewScopedPostgresTaskCommands(db)
	task := createS3DDurableTask(t, s3dProjectContext(ctx), commands, "task-d8-workflow", "trace-d8-workflow")
	activities, err := newPostgresLifecycleActivities(
		repository.NewScopedPostgresTasks(db),
		commands,
		ActivityTrustConfig{Namespace: "d8", TaskQueue: "aicloud-task-v1"},
		func(activityCtx context.Context) ActivityExecutionInfo {
			info := temporalActivityExecutionInfo(activityCtx)
			info.WorkflowID = "task/" + task.ID
			info.WorkflowRunID = "run-d8-workflow"
			info.WorkflowType = TaskExecutionWorkflowType
			info.Namespace = "d8"
			info.TaskQueue = "aicloud-task-v1"
			if info.ActivityID == "" {
				info.ActivityID = "activity-d8"
			}
			return info
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	if err := RegisterLifecycle(env, activities); err != nil {
		t.Fatal(err)
	}
	env.ExecuteWorkflow(TaskLifecycleWorkflow, TaskWorkflowInput{
		SchemaVersion: TaskWorkflowSchemaVersion,
		TenantID:      "tenant-a",
		ProjectID:     "project-a",
		TaskID:        task.ID,
		TraceID:       task.TraceID,
	})
	if !env.IsWorkflowCompleted() {
		t.Fatal("Temporal test workflow did not complete with fail-closed error")
	}
	workflowErr := env.GetWorkflowError()
	if workflowErr == nil {
		t.Fatal("PostgreSQL lifecycle unexpectedly completed through fail-closed execution stubs")
	}
	var applicationErr *temporal.ApplicationError
	if !errors.As(workflowErr, &applicationErr) || applicationErr.Type() != ErrorTypeLifecycleBackendDisabled {
		t.Fatalf("workflow error=%v want application type %s", workflowErr, ErrorTypeLifecycleBackendDisabled)
	}

	var status domain.TaskStatus
	var version int64
	if err := db.QueryRowContext(ctx, `SELECT status, version FROM tasks WHERE id=$1`, task.ID).Scan(&status, &version); err != nil {
		t.Fatal(err)
	}
	if status != domain.TaskPlanning || version != 2 {
		t.Fatalf("fail-closed Workflow business state status=%s version=%d want PLANNING/2", status, version)
	}
	var eventCount, planningEvents int
	if err := db.QueryRowContext(ctx, `SELECT count(*), count(*) FILTER (WHERE event_type='TaskPlanningStarted') FROM task_events WHERE task_id=$1`, task.ID).Scan(&eventCount, &planningEvents); err != nil {
		t.Fatal(err)
	}
	if eventCount != 2 || planningEvents != 1 {
		t.Fatalf("fail-closed Workflow events total=%d planning=%d want 2/1", eventCount, planningEvents)
	}
}

func openS3DWorkflowDB(t *testing.T) (*sql.DB, context.Context) {
	t.Helper()
	dsn := os.Getenv("AICLOUD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("AICLOUD_TEST_DATABASE_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open PostgreSQL: %v", err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		t.Fatalf("ping PostgreSQL: %v", err)
	}
	return db, ctx
}

func s3dProjectContext(ctx context.Context) context.Context {
	return identity.WithPrincipal(ctx, identity.Principal{
		Type: identity.PrincipalUser, SubjectID: "user-a", TenantID: "tenant-a", ProjectID: "project-a",
		AuthnMethod: "integration-test", Issuer: "test",
	})
}

func d8ExecutionInfo(taskID, activityType string) ActivityExecutionInfo {
	return ActivityExecutionInfo{
		WorkflowID: "task/" + taskID, WorkflowRunID: "run-d8", WorkflowType: TaskExecutionWorkflowType,
		Namespace: "d8", TaskQueue: "aicloud-task-v1", ActivityID: "activity-d8", ActivityType: activityType, Attempt: 1,
	}
}

func createS3DDurableTask(t *testing.T, ctx context.Context, commands *repository.ScopedPostgresTaskCommands, taskID, traceID string) domain.Task {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)
	task, err := domain.NewTask(domain.NewTaskParams{
		ID: taskID, AgentID: "agent-d8", Input: "S3D durability matrix", TraceID: traceID, Currency: "USD", Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"taskId": task.ID, "traceId": task.TraceID, "status": task.Status, "agentId": task.AgentID,
	})
	if err != nil {
		t.Fatal(err)
	}
	command := repository.TaskCreateCommit{
		Task: task,
		Event: domain.TaskEvent{
			EventID: "event-" + taskID, EventType: "TaskCreated",
			Actor: domain.TaskEventActor{PrincipalType: string(identity.PrincipalUser), SubjectID: "user-a"},
			Payload: payload, SchemaVersion: 1, OccurredAt: now, CreatedAt: now,
		},
		Outbox: []domain.OutboxMessage{{
			OutboxID: "outbox-" + taskID, AggregateType: "Task", AggregateID: task.ID,
			EventType: "TaskCreated", Payload: payload, Destination: DestinationWorkflowStart,
			IdempotencyKey: "workflow-start:tenant-a:" + task.ID, Status: domain.OutboxPending,
			AvailableAt: now.Add(-time.Second), CreatedAt: now,
		}},
		Idempotency: domain.IdempotencyRecord{
			TenantID: "tenant-a", ProjectID: "project-a", Operation: "POST /api/v1/tasks",
			Key: "create-" + taskID, RequestDigest: "sha256:create-" + taskID, Status: domain.IdempotencyCompleted,
			ResponseCode: 202, ResponseDigest: "sha256:response-" + taskID,
			ResponsePayload: json.RawMessage(`{"accepted":true}`), CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour),
		},
	}
	result, err := commands.CreateTask(ctx, command)
	if err != nil {
		t.Fatalf("create durable Task: %v", err)
	}
	return result.Task
}

func createS3DWorkflowSchema(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	_, err := db.ExecContext(ctx, `
CREATE TABLE tasks (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    created_by TEXT NOT NULL,
    agent_id TEXT NOT NULL DEFAULT '',
    input TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    version BIGINT NOT NULL DEFAULT 1,
    result TEXT NOT NULL DEFAULT '',
    cost DOUBLE PRECISION NOT NULL DEFAULT 0,
    estimated_cost DOUBLE PRECISION NOT NULL DEFAULT 0,
    actual_cost DOUBLE PRECISION NOT NULL DEFAULT 0,
    currency TEXT NOT NULL DEFAULT 'USD',
    route_decision_id TEXT NOT NULL DEFAULT '',
    trace_id TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ
);
CREATE TABLE task_events (
    event_id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    task_id TEXT NOT NULL REFERENCES tasks(id),
    sequence BIGINT NOT NULL,
    event_type TEXT NOT NULL,
    actor_principal_type TEXT NOT NULL,
    actor_subject_id TEXT NOT NULL,
    payload JSONB NOT NULL,
    request_id TEXT,
    trace_id TEXT NOT NULL,
    schema_version INTEGER NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    UNIQUE(task_id, sequence)
);
CREATE TABLE outbox_messages (
    outbox_id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    task_id TEXT REFERENCES tasks(id),
    aggregate_type TEXT NOT NULL,
    aggregate_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    destination TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    status TEXT NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    available_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    delivered_at TIMESTAMPTZ,
    lease_owner TEXT,
    lease_expires_at TIMESTAMPTZ,
    last_error TEXT,
    UNIQUE(tenant_id, project_id, destination, idempotency_key)
);
CREATE TABLE idempotency_records (
    tenant_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    operation TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    request_digest TEXT NOT NULL,
    status TEXT NOT NULL,
    resource_id TEXT,
    response_code INTEGER,
    response_digest TEXT,
    response_payload JSONB,
    created_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY(tenant_id, project_id, operation, idempotency_key)
);
`)
	if err != nil {
		t.Fatalf("create S3D workflow schema: %v", err)
	}
}

func resetS3DWorkflowSchema(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	if ctx == nil {
		ctx = context.Background()
	}
	for _, table := range []string{"task_events", "outbox_messages", "idempotency_records", "tasks"} {
		if _, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS `+table+` CASCADE`); err != nil {
			t.Fatalf("drop %s: %v", table, err)
		}
	}
}
