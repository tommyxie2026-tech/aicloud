//go:build integration

package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/identity"
)

func TestScopedPostgresTaskCommandsAtomicCreateReplayConflictAndRollback(t *testing.T) {
	dsn := os.Getenv("AICLOUD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("AICLOUD_TEST_DATABASE_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open PostgreSQL: %v", err)
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping PostgreSQL: %v", err)
	}
	cleanupTaskCreateFixture(t, ctx, db)
	defer cleanupTaskCreateFixture(t, context.Background(), db)
	createTaskCreateSchema(t, ctx, db)

	principal := identity.Principal{
		Type: identity.PrincipalUser, SubjectID: "user-a", TenantID: "tenant-a", ProjectID: "project-a",
		AuthnMethod: "integration-test", Issuer: "test",
	}
	projectCtx := identity.WithPrincipal(ctx, principal)
	repo := NewScopedPostgresTaskCommands(db)
	now := time.Now().UTC()

	task, err := domain.NewTask(domain.NewTaskParams{
		ID: "task-create-1", Input: "create atomically", TraceID: "trace-create-1", Currency: "USD", Now: now,
	})
	if err != nil {
		t.Fatalf("NewTask: %v", err)
	}
	command := TaskCreateCommit{
		Task: task,
		Event: domain.TaskEvent{
			EventID: "event-create-1", EventType: "TaskCreated",
			Actor:   domain.TaskEventActor{PrincipalType: "user", SubjectID: "user-a"},
			Payload: json.RawMessage(`{"status":"CREATED"}`), SchemaVersion: 1,
		},
		Outbox: []domain.OutboxMessage{{
			OutboxID: "outbox-create-1", AggregateType: "Task", AggregateID: task.ID,
			EventType: "TaskCreated", Payload: json.RawMessage(`{"taskId":"task-create-1"}`),
			Destination: "workflow", IdempotencyKey: "deliver-create-1",
		}},
		Idempotency: domain.IdempotencyRecord{
			TenantID: "tenant-a", ProjectID: "project-a", Operation: "POST /api/v1/tasks",
			Key: "create-key-1", RequestDigest: "sha256:create-1", Status: domain.IdempotencyCompleted,
			ResponseCode: 202, ResponseDigest: "sha256:create-response-1",
			ResponsePayload: json.RawMessage(`{"id":"task-create-1"}`),
			CreatedAt:       now, ExpiresAt: now.Add(24 * time.Hour),
		},
	}

	created, err := repo.CreateTask(projectCtx, command)
	if err != nil {
		t.Fatalf("CreateTask returned error: %v", err)
	}
	if created.Replayed || created.Task.ID != task.ID || created.Task.Version != 1 || created.Event.Sequence != 1 {
		t.Fatalf("unexpected create result: %#v", created)
	}
	if created.Task.TenantID != "tenant-a" || created.Task.ProjectID != "project-a" || created.Task.CreatedBy != "user-a" {
		t.Fatalf("Task identity was not derived from principal: %#v", created.Task)
	}
	assertTaskCreateCounts(t, ctx, db, 1, 1, 1, 1)

	replayed, err := repo.CreateTask(projectCtx, command)
	if err != nil {
		t.Fatalf("CreateTask replay returned error: %v", err)
	}
	if !replayed.Replayed || replayed.Task.ID != task.ID {
		t.Fatalf("expected replay of %s, got %#v", task.ID, replayed)
	}
	assertTaskCreateCounts(t, ctx, db, 1, 1, 1, 1)

	conflict := command
	conflict.Idempotency.RequestDigest = "sha256:changed-request"
	if _, err := repo.CreateTask(projectCtx, conflict); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("same key changed request error=%v want ErrIdempotencyConflict", err)
	}
	assertTaskCreateCounts(t, ctx, db, 1, 1, 1, 1)

	rollbackTask, err := domain.NewTask(domain.NewTaskParams{
		ID: "task-create-rollback", Input: "must rollback", TraceID: "trace-create-rollback", Now: now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("NewTask rollback fixture: %v", err)
	}
	rollback := TaskCreateCommit{
		Task: rollbackTask,
		Event: domain.TaskEvent{
			EventID: "event-create-rollback", EventType: "TaskCreated",
			Actor:   domain.TaskEventActor{PrincipalType: "user", SubjectID: "user-a"},
			Payload: json.RawMessage(`{"status":"CREATED"}`), SchemaVersion: 1,
		},
		Outbox: []domain.OutboxMessage{{
			OutboxID: "outbox-create-invalid", AggregateType: "Task", AggregateID: rollbackTask.ID,
			EventType: "TaskCreated", Payload: json.RawMessage(`{`),
			Destination: "workflow", IdempotencyKey: "deliver-create-invalid",
		}},
		Idempotency: domain.IdempotencyRecord{
			TenantID: "tenant-a", ProjectID: "project-a", Operation: "POST /api/v1/tasks",
			Key: "create-key-rollback", RequestDigest: "sha256:create-rollback", Status: domain.IdempotencyCompleted,
			CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour),
		},
	}
	if _, err := repo.CreateTask(projectCtx, rollback); !errors.Is(err, domain.ErrInvalidOutboxMessage) {
		t.Fatalf("invalid adjacent outbox error=%v want ErrInvalidOutboxMessage", err)
	}
	assertTaskCreateCounts(t, ctx, db, 1, 1, 1, 1)
	var rollbackTaskCount, rollbackKeyCount int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM tasks WHERE id='task-create-rollback'`).Scan(&rollbackTaskCount); err != nil {
		t.Fatalf("count rollback Task: %v", err)
	}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM idempotency_records WHERE idempotency_key='create-key-rollback'`).Scan(&rollbackKeyCount); err != nil {
		t.Fatalf("count rollback idempotency: %v", err)
	}
	if rollbackTaskCount != 0 || rollbackKeyCount != 0 {
		t.Fatalf("atomic rollback leaked task=%d idempotency=%d", rollbackTaskCount, rollbackKeyCount)
	}
}

func createTaskCreateSchema(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	_, err := db.ExecContext(ctx, `
		CREATE TABLE tasks (
			id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, project_id TEXT NOT NULL, created_by TEXT NOT NULL,
			agent_id TEXT NOT NULL DEFAULT '', input TEXT NOT NULL DEFAULT '', status TEXT NOT NULL,
			version BIGINT NOT NULL DEFAULT 1, result TEXT NOT NULL DEFAULT '', cost DOUBLE PRECISION NOT NULL DEFAULT 0,
			estimated_cost DOUBLE PRECISION NOT NULL DEFAULT 0, actual_cost DOUBLE PRECISION NOT NULL DEFAULT 0,
			currency TEXT NOT NULL DEFAULT 'USD', route_decision_id TEXT NOT NULL DEFAULT '', trace_id TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL, updated_at TIMESTAMPTZ NOT NULL, completed_at TIMESTAMPTZ
		);
		CREATE TABLE task_events (
			event_id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, project_id TEXT NOT NULL,
			task_id TEXT NOT NULL REFERENCES tasks(id), sequence BIGINT NOT NULL, event_type TEXT NOT NULL,
			actor_principal_type TEXT NOT NULL, actor_subject_id TEXT NOT NULL, payload JSONB NOT NULL,
			request_id TEXT, trace_id TEXT NOT NULL, schema_version INTEGER NOT NULL,
			occurred_at TIMESTAMPTZ NOT NULL, created_at TIMESTAMPTZ NOT NULL, UNIQUE(task_id, sequence)
		);
		CREATE TABLE outbox_messages (
			outbox_id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, project_id TEXT NOT NULL,
			task_id TEXT REFERENCES tasks(id), aggregate_type TEXT NOT NULL, aggregate_id TEXT NOT NULL,
			event_type TEXT NOT NULL, payload JSONB NOT NULL, destination TEXT NOT NULL, idempotency_key TEXT NOT NULL,
			status TEXT NOT NULL, attempts INTEGER NOT NULL, available_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL, delivered_at TIMESTAMPTZ,
			UNIQUE(tenant_id, project_id, destination, idempotency_key)
		);
		CREATE TABLE idempotency_records (
			tenant_id TEXT NOT NULL, project_id TEXT NOT NULL, operation TEXT NOT NULL, idempotency_key TEXT NOT NULL,
			request_digest TEXT NOT NULL, status TEXT NOT NULL, resource_id TEXT, response_code INTEGER,
			response_digest TEXT, response_payload JSONB, created_at TIMESTAMPTZ NOT NULL, expires_at TIMESTAMPTZ NOT NULL,
			PRIMARY KEY(tenant_id, project_id, operation, idempotency_key)
		);
	`)
	if err != nil {
		t.Fatalf("create Task creation schema: %v", err)
	}
}

func assertTaskCreateCounts(t *testing.T, ctx context.Context, db *sql.DB, tasks, events, outbox, idempotency int) {
	t.Helper()
	for table, want := range map[string]int{
		"tasks":               tasks,
		"task_events":         events,
		"outbox_messages":     outbox,
		"idempotency_records": idempotency,
	} {
		var got int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM `+table).Scan(&got); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if got != want {
			t.Fatalf("%s count=%d want=%d", table, got, want)
		}
	}
}

func cleanupTaskCreateFixture(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	if ctx == nil {
		ctx = context.Background()
	}
	_, _ = db.ExecContext(ctx, `
		DROP TABLE IF EXISTS idempotency_records CASCADE;
		DROP TABLE IF EXISTS outbox_messages CASCADE;
		DROP TABLE IF EXISTS task_events CASCADE;
		DROP TABLE IF EXISTS tasks CASCADE;
	`)
}
