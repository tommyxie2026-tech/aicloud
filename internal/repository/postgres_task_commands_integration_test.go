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

func TestScopedPostgresTaskCommandsAtomicCommitReplayAndRollback(t *testing.T) {
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
	cleanupTaskCommandFixture(t, ctx, db)
	defer cleanupTaskCommandFixture(t, context.Background(), db)
	createTaskCommandFixture(t, ctx, db)

	principal := identity.Principal{
		Type: identity.PrincipalUser, SubjectID: "user-a", TenantID: "tenant-a", ProjectID: "project-a",
		AuthnMethod: "integration-test", Issuer: "test",
	}
	projectCtx := identity.WithPrincipal(ctx, principal)
	repo := NewScopedPostgresTaskCommands(db)
	now := time.Now().UTC()

	planningTask := fixtureTask(now)
	planningTask.Status = domain.TaskPlanning
	planningTask.UpdatedAt = now.Add(time.Second)
	planning := TaskCommandCommit{
		Task: planningTask,
		Transition: domain.TaskTransition{
			From: domain.TaskCreated, To: domain.TaskPlanning,
			Actor: "user:user-a", Cause: "prepare task plan", At: planningTask.UpdatedAt,
		},
		Event: domain.TaskEvent{
			EventID: "event-1", EventType: "TaskPlanningStarted",
			Actor: domain.TaskEventActor{PrincipalType: "user", SubjectID: "user-a"},
			Payload: json.RawMessage(`{"from":"CREATED","to":"PLANNING"}`),
			RequestID: "request-1", SchemaVersion: 1,
		},
		Outbox: []domain.OutboxMessage{{
			OutboxID: "outbox-1", AggregateType: "Task", AggregateID: "task-1",
			EventType: "TaskPlanningStarted", Payload: json.RawMessage(`{"taskId":"task-1"}`),
			Destination: "temporal", IdempotencyKey: "delivery-task-1-1",
		}},
		Idempotency: domain.IdempotencyRecord{
			TenantID: "tenant-a", ProjectID: "project-a", Operation: "task:plan", Key: "command-1",
			RequestDigest: "sha256:plan", Status: domain.IdempotencyCompleted,
			ResponseCode: 200, ResponseDigest: "sha256:response-plan",
			ResponsePayload: json.RawMessage(`{"taskId":"task-1","status":"PLANNING"}`),
			CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour),
		},
	}

	result, err := repo.CommitTransition(projectCtx, planning)
	if err != nil {
		t.Fatalf("CommitTransition returned error: %v", err)
	}
	if result.Replayed || result.Task.Version != 2 || result.Event.Sequence != 1 {
		t.Fatalf("unexpected first commit result: %#v", result)
	}
	assertTaskCommandFixture(t, ctx, db, domain.TaskPlanning, 2, 1, 1, 1)

	replay, err := repo.CommitTransition(projectCtx, planning)
	if err != nil {
		t.Fatalf("idempotent replay returned error: %v", err)
	}
	if !replay.Replayed || replay.Idempotency.ResourceID != "task-1" {
		t.Fatalf("expected replay of task-1, got %#v", replay)
	}
	assertTaskCommandFixture(t, ctx, db, domain.TaskPlanning, 2, 1, 1, 1)

	conflict := planning
	conflict.Idempotency.RequestDigest = "sha256:different"
	if _, err := repo.CommitTransition(projectCtx, conflict); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("same key with changed digest error=%v want ErrIdempotencyConflict", err)
	}

	routingTask := planningTask
	routingTask.Version = 2
	routingTask.Status = domain.TaskRouting
	routingTask.UpdatedAt = now.Add(2 * time.Second)
	rollbackCommand := TaskCommandCommit{
		Task: routingTask,
		Transition: domain.TaskTransition{
			From: domain.TaskPlanning, To: domain.TaskRouting,
			Actor: "user:user-a", Cause: "route task", At: routingTask.UpdatedAt,
		},
		Event: domain.TaskEvent{
			EventID: "event-rollback", EventType: "TaskRoutingStarted",
			Actor: domain.TaskEventActor{PrincipalType: "user", SubjectID: "user-a"},
			Payload: json.RawMessage(`{"to":"ROUTING"}`), SchemaVersion: 1,
		},
		Outbox: []domain.OutboxMessage{{
			OutboxID: "outbox-invalid", AggregateType: "Task", AggregateID: "task-1",
			EventType: "TaskRoutingStarted", Payload: json.RawMessage(`{`),
			Destination: "temporal", IdempotencyKey: "delivery-task-1-invalid",
		}},
		Idempotency: domain.IdempotencyRecord{
			TenantID: "tenant-a", ProjectID: "project-a", Operation: "task:route", Key: "command-rollback",
			RequestDigest: "sha256:rollback", Status: domain.IdempotencyCompleted,
			CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour),
		},
	}
	if _, err := repo.CommitTransition(projectCtx, rollbackCommand); !errors.Is(err, domain.ErrInvalidOutboxMessage) {
		t.Fatalf("invalid outbox error=%v want ErrInvalidOutboxMessage", err)
	}
	assertTaskCommandFixture(t, ctx, db, domain.TaskPlanning, 2, 1, 1, 1)
	var rollbackKeys int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM idempotency_records WHERE idempotency_key='command-rollback'`).Scan(&rollbackKeys); err != nil {
		t.Fatalf("count rolled-back idempotency records: %v", err)
	}
	if rollbackKeys != 0 {
		t.Fatalf("rolled-back command left %d idempotency records", rollbackKeys)
	}

	routing := rollbackCommand
	routing.Event.EventID = "event-2"
	routing.Outbox[0].OutboxID = "outbox-2"
	routing.Outbox[0].Payload = json.RawMessage(`{"taskId":"task-1"}`)
	routing.Outbox[0].IdempotencyKey = "delivery-task-1-2"
	routing.Idempotency.Key = "command-2"
	routing.Idempotency.RequestDigest = "sha256:route"
	routed, err := repo.CommitTransition(projectCtx, routing)
	if err != nil {
		t.Fatalf("second successful command returned error: %v", err)
	}
	if routed.Task.Version != 3 || routed.Event.Sequence != 2 {
		t.Fatalf("unexpected routed result: %#v", routed)
	}
	assertTaskCommandFixture(t, ctx, db, domain.TaskRouting, 3, 2, 2, 2)
}

func fixtureTask(now time.Time) domain.Task {
	return domain.Task{
		ID: "task-1", TenantID: "tenant-a", ProjectID: "project-a", CreatedBy: "user-a",
		Input: "integration task", Status: domain.TaskCreated, Version: 1, Currency: "USD",
		TraceID: "trace-1", CreatedAt: now, UpdatedAt: now,
	}
}

func createTaskCommandFixture(t *testing.T, ctx context.Context, db *sql.DB) {
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
			version BIGINT NOT NULL DEFAULT 1 CHECK (version >= 1),
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
			attempts INTEGER NOT NULL,
			available_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			delivered_at TIMESTAMPTZ,
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
		t.Fatalf("create task command fixture schema: %v", err)
	}
	now := time.Now().UTC()
	task := fixtureTask(now)
	_, err = db.ExecContext(ctx, `INSERT INTO tasks(
		id, tenant_id, project_id, created_by, agent_id, input, status, version,
		result, cost, estimated_cost, actual_cost, currency, route_decision_id,
		trace_id, created_at, updated_at, completed_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`,
		task.ID, task.TenantID, task.ProjectID, task.CreatedBy, task.AgentID, task.Input,
		task.Status, task.Version, task.Result, task.Cost, task.EstimatedCost,
		task.ActualCost, task.Currency, task.RouteDecisionID, task.TraceID,
		task.CreatedAt, task.UpdatedAt, task.CompletedAt)
	if err != nil {
		t.Fatalf("insert task command fixture: %v", err)
	}
}

func assertTaskCommandFixture(t *testing.T, ctx context.Context, db *sql.DB, status domain.TaskStatus, version int64, eventCount, outboxCount, idempotencyCount int) {
	t.Helper()
	var gotStatus domain.TaskStatus
	var gotVersion int64
	if err := db.QueryRowContext(ctx, `SELECT status, version FROM tasks WHERE id='task-1'`).Scan(&gotStatus, &gotVersion); err != nil {
		t.Fatalf("read task projection: %v", err)
	}
	if gotStatus != status || gotVersion != version {
		t.Fatalf("task projection status=%s version=%d want status=%s version=%d", gotStatus, gotVersion, status, version)
	}
	for table, want := range map[string]int{
		"task_events": eventCount,
		"outbox_messages": outboxCount,
		"idempotency_records": idempotencyCount,
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

func cleanupTaskCommandFixture(t *testing.T, ctx context.Context, db *sql.DB) {
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
