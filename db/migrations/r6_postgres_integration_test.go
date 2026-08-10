//go:build integration

package migrations

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

func TestTaskEventOutboxIdempotencyMigrationPostgres(t *testing.T) {
	db, ctx := openIntegrationDB(t)
	defer db.Close()
	cleanupR6Fixture(t, ctx, db)
	defer cleanupR6Fixture(t, context.Background(), db)

	_, err := db.ExecContext(ctx, `
		CREATE TABLE tasks (
			id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			project_id TEXT NOT NULL,
			created_by TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'CREATED',
			version BIGINT NOT NULL DEFAULT 1,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		INSERT INTO tasks(id, tenant_id, project_id, created_by)
		VALUES ('task-a', 'tenant-a', 'project-a', 'user-a');
	`)
	if err != nil {
		t.Fatalf("create R6 migration fixture: %v", err)
	}

	body, err := migrationFiles.ReadFile("007_task_event_outbox_idempotency.sql")
	if err != nil {
		t.Fatalf("read migration 007: %v", err)
	}
	if _, err := db.ExecContext(ctx, string(body)); err != nil {
		t.Fatalf("execute migration 007: %v", err)
	}

	for _, table := range []string{"task_events", "outbox_messages", "idempotency_records"} {
		var enabled, forced bool
		if err := db.QueryRowContext(ctx, `SELECT relrowsecurity, relforcerowsecurity FROM pg_class WHERE oid=$1::regclass`, table).Scan(&enabled, &forced); err != nil {
			t.Fatalf("read RLS flags for %s: %v", table, err)
		}
		if !enabled || !forced {
			t.Fatalf("RLS must be enabled and forced for %s: enabled=%v forced=%v", table, enabled, forced)
		}
	}

	if _, err := db.ExecContext(ctx, `
		CREATE ROLE aicloud_r6_runtime NOLOGIN;
		GRANT SELECT, INSERT, UPDATE, DELETE ON task_events TO aicloud_r6_runtime;
		GRANT SELECT, INSERT, UPDATE, DELETE ON outbox_messages TO aicloud_r6_runtime;
		GRANT SELECT, INSERT, UPDATE, DELETE ON idempotency_records TO aicloud_r6_runtime;
	`); err != nil {
		t.Fatalf("create R6 runtime role: %v", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin R6 verification: %v", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `SET LOCAL ROLE aicloud_r6_runtime`); err != nil {
		t.Fatalf("set R6 runtime role: %v", err)
	}
	if _, err := tx.ExecContext(ctx, `SELECT set_config('aicloud.tenant_id','tenant-a',true), set_config('aicloud.project_id','project-a',true)`); err != nil {
		t.Fatalf("set R6 scope: %v", err)
	}

	now := time.Now().UTC()
	if _, err := tx.ExecContext(ctx, `INSERT INTO task_events(
		event_id, tenant_id, project_id, task_id, sequence, event_type,
		actor_principal_type, actor_subject_id, payload, trace_id, schema_version,
		occurred_at, created_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,$10,$11,$12,$13)`,
		"event-1", "tenant-a", "project-a", "task-a", 1, "TaskCreated",
		"user", "user-a", `{"status":"CREATED"}`, "trace-a", 1, now, now); err != nil {
		t.Fatalf("insert TaskEvent: %v", err)
	}

	expectConstraintError(t, ctx, tx, "duplicate task event sequence", `INSERT INTO task_events(
		event_id, tenant_id, project_id, task_id, sequence, event_type,
		actor_principal_type, actor_subject_id, payload, trace_id, schema_version,
		occurred_at, created_at
	) VALUES ('event-2','tenant-a','project-a','task-a',1,'TaskPlanningStarted','user','user-a','{}'::jsonb,'trace-a',1,NOW(),NOW())`)

	result, err := tx.ExecContext(ctx, `UPDATE task_events SET event_type='rewritten' WHERE event_id='event-1'`)
	if err != nil {
		t.Fatalf("TaskEvent UPDATE should be filtered by RLS, got error: %v", err)
	}
	if affected, _ := result.RowsAffected(); affected != 0 {
		t.Fatalf("runtime TaskEvent UPDATE must affect zero rows, got %d", affected)
	}
	result, err = tx.ExecContext(ctx, `DELETE FROM task_events WHERE event_id='event-1'`)
	if err != nil {
		t.Fatalf("TaskEvent DELETE should be filtered by RLS, got error: %v", err)
	}
	if affected, _ := result.RowsAffected(); affected != 0 {
		t.Fatalf("runtime TaskEvent DELETE must affect zero rows, got %d", affected)
	}

	if _, err := tx.ExecContext(ctx, `INSERT INTO outbox_messages(
		outbox_id, tenant_id, project_id, task_id, aggregate_type, aggregate_id,
		event_type, payload, destination, idempotency_key, status, available_at, created_at
	) VALUES ('outbox-1','tenant-a','project-a','task-a','Task','task-a','TaskCreated','{}'::jsonb,'temporal','delivery-1','pending',NOW(),NOW())`); err != nil {
		t.Fatalf("insert outbox message: %v", err)
	}
	expectConstraintError(t, ctx, tx, "duplicate outbox idempotency key", `INSERT INTO outbox_messages(
		outbox_id, tenant_id, project_id, task_id, aggregate_type, aggregate_id,
		event_type, payload, destination, idempotency_key, status, available_at, created_at
	) VALUES ('outbox-2','tenant-a','project-a','task-a','Task','task-a','TaskCreated','{}'::jsonb,'temporal','delivery-1','pending',NOW(),NOW())`)

	if _, err := tx.ExecContext(ctx, `INSERT INTO idempotency_records(
		tenant_id, project_id, operation, idempotency_key, request_digest, status,
		created_at, expires_at
	) VALUES ('tenant-a','project-a','POST /tasks','command-1','sha256:a','in_progress',NOW(),NOW()+INTERVAL '1 hour')`); err != nil {
		t.Fatalf("insert idempotency record: %v", err)
	}
	expectConstraintError(t, ctx, tx, "duplicate command idempotency scope", `INSERT INTO idempotency_records(
		tenant_id, project_id, operation, idempotency_key, request_digest, status,
		created_at, expires_at
	) VALUES ('tenant-a','project-a','POST /tasks','command-1','sha256:a','in_progress',NOW(),NOW()+INTERVAL '1 hour')`)

	if _, err := tx.ExecContext(ctx, `SELECT set_config('aicloud.project_id','project-b',true)`); err != nil {
		t.Fatalf("switch R6 project scope: %v", err)
	}
	var visible int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM task_events`).Scan(&visible); err != nil {
		t.Fatalf("query cross-project TaskEvents: %v", err)
	}
	if visible != 0 {
		t.Fatalf("cross-project scope must not see TaskEvents, got %d", visible)
	}
}

func expectConstraintError(t *testing.T, ctx context.Context, tx *sql.Tx, name, statement string) {
	t.Helper()
	if _, err := tx.ExecContext(ctx, `SAVEPOINT r6_expected_error`); err != nil {
		t.Fatalf("%s: create savepoint: %v", name, err)
	}
	_, execErr := tx.ExecContext(ctx, statement)
	if _, err := tx.ExecContext(ctx, `ROLLBACK TO SAVEPOINT r6_expected_error`); err != nil {
		t.Fatalf("%s: rollback savepoint: %v", name, err)
	}
	if _, err := tx.ExecContext(ctx, `RELEASE SAVEPOINT r6_expected_error`); err != nil {
		t.Fatalf("%s: release savepoint: %v", name, err)
	}
	if execErr == nil {
		t.Fatalf("%s must be rejected", name)
	}
}

func cleanupR6Fixture(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	if ctx == nil {
		ctx = context.Background()
	}
	_, _ = db.ExecContext(ctx, `
		DROP TABLE IF EXISTS idempotency_records CASCADE;
		DROP TABLE IF EXISTS outbox_messages CASCADE;
		DROP TABLE IF EXISTS task_events CASCADE;
		DROP TABLE IF EXISTS tasks CASCADE;
		DROP ROLE IF EXISTS aicloud_r6_runtime;
	`)
}
