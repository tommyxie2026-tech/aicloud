//go:build integration

package migrations

import (
	"context"
	"testing"
)

func TestExecutionAttemptMigrationAllowsPreStartCancellation(t *testing.T) {
	db, ctx := openIntegrationDB(t)
	defer db.Close()
	cleanupExecutionAttemptMigrationFixture(t, context.Background(), db)
	defer cleanupExecutionAttemptMigrationFixture(t, context.Background(), db)

	migration017, err := migrationFiles.ReadFile("017_execution_node_runtime_leases.sql")
	if err != nil {
		t.Fatalf("read migration 017: %v", err)
	}
	if _, err := db.ExecContext(ctx, string(migration017)); err != nil {
		t.Fatalf("execute migration 017: %v", err)
	}
	migration018, err := migrationFiles.ReadFile("018_execution_attempts.sql")
	if err != nil {
		t.Fatalf("read migration 018: %v", err)
	}
	if _, err := db.ExecContext(ctx, string(migration018)); err != nil {
		t.Fatalf("execute migration 018: %v", err)
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO execution_attempts(
		tenant_id, project_id, attempt_id, execution_id, plan_revision, node_id,
		attempt_number, lease_fence, lease_owner, status, claimed_at, finished_at
	) VALUES (
		'tenant-a','project-a','att-cancelled','exec-cancel',1,'node-cancel',1,1,
		'worker-a','CANCELLED',NOW(),NOW()
	)`); err != nil {
		t.Fatalf("pre-start CANCELLED attempt should be valid: %v", err)
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO execution_attempts(
		tenant_id, project_id, attempt_id, execution_id, plan_revision, node_id,
		attempt_number, lease_fence, lease_owner, status, claimed_at, finished_at
	) VALUES (
		'tenant-a','project-a','att-success-no-start','exec-success',1,'node-success',1,1,
		'worker-a','SUCCEEDED',NOW(),NOW()
	)`); err == nil {
		t.Fatal("pre-start SUCCEEDED attempt must be rejected")
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO execution_attempts(
		tenant_id, project_id, attempt_id, execution_id, plan_revision, node_id,
		attempt_number, lease_fence, lease_owner, status, error_class, claimed_at, finished_at
	) VALUES (
		'tenant-a','project-a','att-failed-no-start','exec-failed',1,'node-failed',1,1,
		'worker-a','FAILED','TIMEOUT',NOW(),NOW()
	)`); err == nil {
		t.Fatal("pre-start FAILED attempt must be rejected")
	}
}
