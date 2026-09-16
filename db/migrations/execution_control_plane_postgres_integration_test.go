//go:build integration

package migrations

import (
	"context"
	"database/sql"
	"testing"
)

func TestExecutionControlPlaneMigrationPostgres(t *testing.T) {
	db, ctx := openIntegrationDB(t)
	defer db.Close()
	cleanupExecutionControlPlaneFixture(t, ctx, db)
	defer cleanupExecutionControlPlaneFixture(t, context.Background(), db)

	if _, err := db.ExecContext(ctx, `
		CREATE TABLE tasks (id TEXT PRIMARY KEY);
		INSERT INTO tasks(id) VALUES ('task-a'), ('task-b');
	`); err != nil {
		t.Fatalf("create execution control-plane fixture: %v", err)
	}
	body, err := migrationFiles.ReadFile("019_execution_control_plane.sql")
	if err != nil {
		t.Fatalf("read migration 019: %v", err)
	}
	if _, err := db.ExecContext(ctx, string(body)); err != nil {
		t.Fatalf("execute migration 019: %v", err)
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO execution_goals (
		tenant_id, project_id, goal_id, task_id, objective
	) VALUES ('tenant-a','project-a','goal-a','task-a','diagnose incident')`); err != nil {
		t.Fatalf("insert execution goal: %v", err)
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO execution_plans (
		tenant_id, project_id, plan_id, task_id, goal_id, revision, plan_json
	) VALUES ('tenant-a','project-a','plan-bad','task-b','goal-a',1,'{}'::jsonb)`); err == nil {
		t.Fatal("plan must not bind a goal to a different Task")
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO execution_plans (
		tenant_id, project_id, plan_id, task_id, goal_id, revision, plan_json
	) VALUES ('tenant-a','project-a','plan-a','task-a','goal-a',1,'{}'::jsonb)`); err != nil {
		t.Fatalf("insert execution plan: %v", err)
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO executions (
		tenant_id, project_id, execution_id, task_id, goal_id, plan_id,
		plan_revision, principal, phase, created_at, updated_at
	) VALUES (
		'tenant-a','project-a','exec-bad','task-b','goal-a','plan-a',
		1,'user-a','CREATED',NOW(),NOW()
	)`); err == nil {
		t.Fatal("execution must not cross Task/Goal/Plan lineage")
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO executions (
		tenant_id, project_id, execution_id, task_id, goal_id, plan_id,
		plan_revision, principal, phase, created_at, updated_at
	) VALUES (
		'tenant-a','project-a','exec-a','task-a','goal-a','plan-a',
		1,'user-a','CREATED',NOW(),NOW()
	)`); err != nil {
		t.Fatalf("insert valid execution lineage: %v", err)
	}

	var count int
	if err := db.QueryRowContext(ctx, `SELECT count(*)
		FROM executions e
		JOIN execution_plans p
		  ON p.tenant_id=e.tenant_id AND p.project_id=e.project_id
		 AND p.plan_id=e.plan_id AND p.revision=e.plan_revision
		JOIN execution_goals g
		  ON g.tenant_id=e.tenant_id AND g.project_id=e.project_id
		 AND g.goal_id=e.goal_id
		WHERE e.task_id='task-a' AND p.task_id=e.task_id AND g.task_id=e.task_id`).Scan(&count); err != nil {
		t.Fatalf("query execution lineage: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one reconstructable execution lineage, got %d", count)
	}

	for _, table := range []string{"execution_goals", "execution_plans", "executions"} {
		var enabled, forced bool
		if err := db.QueryRowContext(ctx, `SELECT relrowsecurity, relforcerowsecurity
			FROM pg_class WHERE oid=$1::regclass`, table).Scan(&enabled, &forced); err != nil {
			t.Fatalf("read RLS flags for %s: %v", table, err)
		}
		if !enabled || !forced {
			t.Fatalf("RLS must be enabled and forced for %s", table)
		}
	}
}

func cleanupExecutionControlPlaneFixture(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	_, _ = db.ExecContext(ctx, `
		DROP TABLE IF EXISTS executions CASCADE;
		DROP TABLE IF EXISTS execution_plans CASCADE;
		DROP TABLE IF EXISTS execution_goals CASCADE;
		DROP TABLE IF EXISTS tasks CASCADE;
	`)
}
