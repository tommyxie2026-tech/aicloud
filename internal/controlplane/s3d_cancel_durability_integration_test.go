//go:build integration

package controlplane

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/identity"
	"github.com/tommyxie2026-tech/aicloud/internal/modelservice"
	outboxpkg "github.com/tommyxie2026-tech/aicloud/internal/outbox"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
	"github.com/tommyxie2026-tech/aicloud/internal/workflow"
)

type d8CancelEngine struct {
	mu       sync.Mutex
	requests []workflow.CancelRequest
}

func (*d8CancelEngine) Start(context.Context, workflow.StartRequest) (workflow.StartResult, error) {
	return workflow.StartResult{}, nil
}

func (e *d8CancelEngine) Cancel(_ context.Context, request workflow.CancelRequest) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.requests = append(e.requests, request)
	if len(e.requests) == 1 {
		return errors.New("Temporal cancel temporarily unavailable")
	}
	return nil
}

func TestS3DCancelBusinessStateSurvivesTemporalCancelRetry(t *testing.T) {
	db, ctx := openS3DCancelDB(t)
	defer db.Close()
	resetS3DCancelSchema(t, ctx, db)
	defer resetS3DCancelSchema(t, context.Background(), db)
	createS3DCancelSchema(t, ctx, db)

	projectCtx := identity.WithPrincipal(ctx, identity.Principal{
		Type: identity.PrincipalUser, SubjectID: "operator-a",
		TenantID: "tenant-a", ProjectID: "project-a", AuthnMethod: "integration-test", Issuer: "test",
	})
	tasks := repository.NewScopedPostgresTasks(db)
	commands := repository.NewScopedPostgresTaskCommands(db)
	task := createS3DCancelTask(t, projectCtx, commands)
	service := New(modelservice.New(repository.NewMemoryModels()), tasks, workflow.NoopEngine{})

	cancelResult, err := service.CancelTaskIdempotent(
		projectCtx,
		task.ID,
		1,
		"operator requested durable cancellation",
		CommandMetadata{
			IdempotencyKey: "cancel-task-d8",
			RequestDigest:  "sha256:cancel-task-d8",
			RequestID:      "request-cancel-d8",
		},
	)
	if err != nil {
		t.Fatalf("commit business cancellation: %v", err)
	}
	if cancelResult.Replayed || cancelResult.Task.Status != domain.TaskCancelled || cancelResult.Task.Version != 2 {
		t.Fatalf("unexpected business cancellation result: %+v", cancelResult)
	}
	assertS3DCancelledTask(t, ctx, db, task.ID, 2)

	var cancelEvents int
	if err := db.QueryRowContext(ctx, `
SELECT count(*) FROM task_events WHERE task_id=$1 AND event_type='TaskCancelled'`, task.ID).Scan(&cancelEvents); err != nil {
		t.Fatalf("read TaskCancelled evidence: %v", err)
	}
	if cancelEvents != 1 {
		t.Fatalf("TaskCancelled events=%d want=1", cancelEvents)
	}

	engine := &d8CancelEngine{}
	adapter := workflow.NewCancelDeliveryAdapter(engine)
	store := repository.NewScopedPostgresOutbox(db)
	dispatcher, err := outboxpkg.NewDispatcher(store, "d8-cancel-dispatcher", time.Second, 3, 5*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if err := dispatcher.Register(workflow.DestinationWorkflowCancel, adapter); err != nil {
		t.Fatal(err)
	}

	first, err := dispatcher.DispatchOnce(projectCtx, 10)
	if err != nil {
		t.Fatalf("first physical cancellation dispatch: %v", err)
	}
	if first.Leased != 1 || first.Retried != 1 || first.Delivered != 0 {
		t.Fatalf("unexpected first cancellation dispatch: %+v", first)
	}
	assertS3DCancelledTask(t, ctx, db, task.ID, 2)

	var status domain.OutboxStatus
	var attempts int
	var availableAt time.Time
	var lastError string
	if err := db.QueryRowContext(ctx, `
SELECT status, attempts, available_at, COALESCE(last_error,'')
FROM outbox_messages
WHERE task_id=$1 AND destination=$2`, task.ID, workflow.DestinationWorkflowCancel).Scan(
		&status, &attempts, &availableAt, &lastError,
	); err != nil {
		t.Fatalf("read retried cancellation Outbox: %v", err)
	}
	if status != domain.OutboxPending || attempts != 1 || !strings.Contains(lastError, "Temporal cancel temporarily unavailable") {
		t.Fatalf("cancel retry evidence status=%s attempts=%d lastError=%q", status, attempts, lastError)
	}
	if wait := time.Until(availableAt) + 20*time.Millisecond; wait > 0 {
		time.Sleep(wait)
	}

	second, err := dispatcher.DispatchOnce(projectCtx, 10)
	if err != nil {
		t.Fatalf("second physical cancellation dispatch: %v", err)
	}
	if second.Leased != 1 || second.Delivered != 1 || second.Retried != 0 {
		t.Fatalf("unexpected second cancellation dispatch: %+v", second)
	}
	assertS3DCancelledTask(t, ctx, db, task.ID, 2)

	engine.mu.Lock()
	requests := append([]workflow.CancelRequest(nil), engine.requests...)
	engine.mu.Unlock()
	if len(requests) != 2 {
		t.Fatalf("Temporal cancellation attempts=%d want=2", len(requests))
	}
	for index, request := range requests {
		if request.TenantID != "tenant-a" || request.ProjectID != "project-a" || request.TaskID != task.ID || request.TraceID != task.TraceID {
			t.Fatalf("cancel[%d] identity drifted: %+v", index, request)
		}
		if request.Reason != "operator requested durable cancellation" {
			t.Fatalf("cancel[%d] reason=%q", index, request.Reason)
		}
	}

	if err := db.QueryRowContext(ctx, `
SELECT status, attempts FROM outbox_messages WHERE task_id=$1 AND destination=$2`, task.ID, workflow.DestinationWorkflowCancel).Scan(
		&status, &attempts,
	); err != nil {
		t.Fatalf("read delivered cancellation Outbox: %v", err)
	}
	if status != domain.OutboxDelivered || attempts != 2 {
		t.Fatalf("delivered cancellation Outbox status=%s attempts=%d", status, attempts)
	}
	if err := db.QueryRowContext(ctx, `
SELECT count(*) FROM task_events WHERE task_id=$1 AND event_type='TaskCancelled'`, task.ID).Scan(&cancelEvents); err != nil {
		t.Fatal(err)
	}
	if cancelEvents != 1 {
		t.Fatalf("physical cancellation retry duplicated business event: %d", cancelEvents)
	}
}

func createS3DCancelTask(t *testing.T, ctx context.Context, commands *repository.ScopedPostgresTaskCommands) domain.Task {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)
	task, err := domain.NewTask(domain.NewTaskParams{
		ID: "task-d8-cancel", AgentID: "agent-d8", Input: "cancel durable task",
		TraceID: "trace-d8-cancel", Currency: "USD", Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"taskId": task.ID, "traceId": task.TraceID, "status": task.Status,
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := commands.CreateTask(ctx, repository.TaskCreateCommit{
		Task: task,
		Event: domain.TaskEvent{
			EventID: "event-task-d8-cancel", EventType: "TaskCreated",
			Actor: domain.TaskEventActor{PrincipalType: string(identity.PrincipalUser), SubjectID: "operator-a"},
			Payload: payload, SchemaVersion: 1, OccurredAt: now, CreatedAt: now,
		},
		Idempotency: domain.IdempotencyRecord{
			TenantID: "tenant-a", ProjectID: "project-a", Operation: "test.create.cancel-task",
			Key: "create-task-d8-cancel", RequestDigest: "sha256:create-task-d8-cancel",
			Status: domain.IdempotencyCompleted, ResponseCode: 202,
			ResponseDigest: "sha256:create-task-d8-cancel-response",
			ResponsePayload: json.RawMessage(`{"accepted":true}`),
			CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour),
		},
	})
	if err != nil {
		t.Fatalf("create durable cancellation Task: %v", err)
	}
	return result.Task
}

func assertS3DCancelledTask(t *testing.T, ctx context.Context, db *sql.DB, taskID string, wantVersion int64) {
	t.Helper()
	var status domain.TaskStatus
	var version int64
	if err := db.QueryRowContext(ctx, `SELECT status, version FROM tasks WHERE id=$1`, taskID).Scan(&status, &version); err != nil {
		t.Fatalf("read business Task: %v", err)
	}
	if status != domain.TaskCancelled || version != wantVersion {
		t.Fatalf("business Task status=%s version=%d want CANCELLED/%d", status, version, wantVersion)
	}
}

func openS3DCancelDB(t *testing.T) (*sql.DB, context.Context) {
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

func createS3DCancelSchema(t *testing.T, ctx context.Context, db *sql.DB) {
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
		t.Fatalf("create S3D cancellation schema: %v", err)
	}
}

func resetS3DCancelSchema(t *testing.T, ctx context.Context, db *sql.DB) {
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
