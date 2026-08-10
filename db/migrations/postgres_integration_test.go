//go:build integration

package migrations

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

func TestTaskScopeMigrationPostgres(t *testing.T) {
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

	cleanupTaskScopeFixture(t, ctx, db)
	defer cleanupTaskScopeFixture(t, context.Background(), db)

	_, err = db.ExecContext(ctx, `
		CREATE TABLE tasks (
			id TEXT PRIMARY KEY,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE TABLE task_ownership (
			task_id TEXT PRIMARY KEY,
			tenant_id TEXT NOT NULL,
			project_id TEXT NOT NULL,
			subject_id TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		INSERT INTO tasks(id) VALUES ('task-a'), ('task-b');
		INSERT INTO task_ownership(task_id, tenant_id, project_id, subject_id) VALUES
			('task-a', 'tenant-a', 'project-a', 'user-a'),
			('task-b', 'tenant-a', 'project-b', 'user-b');
	`)
	if err != nil {
		t.Fatalf("create migration fixture: %v", err)
	}

	body, err := migrationFiles.ReadFile("005_task_scope_identity.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	if _, err := db.ExecContext(ctx, string(body)); err != nil {
		t.Fatalf("execute migration 005: %v", err)
	}

	var tenantID, projectID, createdBy string
	if err := db.QueryRowContext(ctx, `SELECT tenant_id, project_id, created_by FROM tasks WHERE id='task-a'`).Scan(&tenantID, &projectID, &createdBy); err != nil {
		t.Fatalf("read migrated Task identity: %v", err)
	}
	if tenantID != "tenant-a" || projectID != "project-a" || createdBy != "user-a" {
		t.Fatalf("unexpected migrated identity tenant=%q project=%q createdBy=%q", tenantID, projectID, createdBy)
	}

	for _, column := range []string{"tenant_id", "project_id", "created_by"} {
		var nullable string
		if err := db.QueryRowContext(ctx, `SELECT is_nullable FROM information_schema.columns
			WHERE table_schema=current_schema() AND table_name='tasks' AND column_name=$1`, column).Scan(&nullable); err != nil {
			t.Fatalf("read column %s: %v", column, err)
		}
		if nullable != "NO" {
			t.Fatalf("column %s must be NOT NULL, got is_nullable=%s", column, nullable)
		}
	}

	for _, table := range []string{"tasks", "task_ownership"} {
		var enabled, forced bool
		if err := db.QueryRowContext(ctx, `SELECT relrowsecurity, relforcerowsecurity FROM pg_class WHERE oid=$1::regclass`, table).Scan(&enabled, &forced); err != nil {
			t.Fatalf("read RLS flags for %s: %v", table, err)
		}
		if !enabled || !forced {
			t.Fatalf("RLS must be enabled and forced for %s: enabled=%v forced=%v", table, enabled, forced)
		}
	}

	if _, err := db.ExecContext(ctx, `CREATE ROLE aicloud_rls_test NOLOGIN; GRANT SELECT ON tasks TO aicloud_rls_test`); err != nil {
		t.Fatalf("create RLS test role: %v", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin RLS verification: %v", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `SET LOCAL ROLE aicloud_rls_test`); err != nil {
		t.Fatalf("set RLS test role: %v", err)
	}
	if _, err := tx.ExecContext(ctx, `SELECT set_config('aicloud.tenant_id','tenant-a',true), set_config('aicloud.project_id','project-a',true)`); err != nil {
		t.Fatalf("set project-a RLS context: %v", err)
	}
	var visible int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM tasks`).Scan(&visible); err != nil {
		t.Fatalf("query project-a tasks: %v", err)
	}
	if visible != 1 {
		t.Fatalf("project-a should see exactly one Task, got %d", visible)
	}
	if _, err := tx.ExecContext(ctx, `SELECT set_config('aicloud.project_id','project-c',true)`); err != nil {
		t.Fatalf("set project-c RLS context: %v", err)
	}
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM tasks`).Scan(&visible); err != nil {
		t.Fatalf("query project-c tasks: %v", err)
	}
	if visible != 0 {
		t.Fatalf("project-c must not see tasks from other projects, got %d", visible)
	}
}

func cleanupTaskScopeFixture(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	if ctx == nil {
		ctx = context.Background()
	}
	_, _ = db.ExecContext(ctx, `DROP TABLE IF EXISTS task_ownership CASCADE; DROP TABLE IF EXISTS tasks CASCADE; DROP ROLE IF EXISTS aicloud_rls_test`)
}
