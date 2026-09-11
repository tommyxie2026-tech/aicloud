//go:build integration

package migrations

import (
	"context"
	"database/sql"
	"testing"
)

func TestExecutionAttemptMigrationPostgresConstraints(t *testing.T) {
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

	// First application of 018 must refuse to inherit a RUNNING pre-attempt
	// fence because no durable attempt history exists for that old lease.
	if _, err := db.ExecContext(ctx, `INSERT INTO execution_node_runtime(
		tenant_id, project_id, execution_id, plan_revision, node_id, state,
		effect_class, retry_safe, attempt_number, lease_owner, lease_token,
		lease_fence, claimed_at, heartbeat_at, lease_expires_at
	) VALUES (
		'tenant-a','project-a','legacy-exec',1,'legacy-node','RUNNING',
		'PURE',true,1,'legacy-worker','legacy-token',1,
		NOW(),NOW(),NOW()+INTERVAL '1 minute'
	)`); err != nil {
		t.Fatalf("create legacy running lease: %v", err)
	}

	migration018, err := migrationFiles.ReadFile("018_execution_attempts.sql")
	if err != nil {
		t.Fatalf("read migration 018: %v", err)
	}
	if _, err := db.ExecContext(ctx, string(migration018)); err == nil {
		t.Fatal("first apply of migration 018 must reject an undrained RUNNING lease")
	}
	if _, err := db.ExecContext(ctx, `UPDATE execution_node_runtime
		SET state='READY', lease_owner=NULL, lease_token=NULL,
			claimed_at=NULL, heartbeat_at=NULL, lease_expires_at=NULL
		WHERE execution_id='legacy-exec'`); err != nil {
		t.Fatalf("drain legacy lease: %v", err)
	}
	if _, err := db.ExecContext(ctx, string(migration018)); err != nil {
		t.Fatalf("execute migration 018 after drain: %v", err)
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO execution_attempts(
		tenant_id, project_id, attempt_id, execution_id, plan_revision, node_id,
		attempt_number, lease_fence, lease_owner, status, claimed_at
	) VALUES ('tenant-a','project-a','att-pending','exec-1',1,'node-1',1,1,'worker-a','PENDING',NOW())`); err != nil {
		t.Fatalf("valid PENDING attempt rejected: %v", err)
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO execution_attempts(
		tenant_id, project_id, attempt_id, execution_id, plan_revision, node_id,
		attempt_number, lease_fence, lease_owner, status, claimed_at, started_at,
		target_id, target_revision, target_snapshot_digest,
		policy_decision_id, budget_reservation_id
	) VALUES (
		'tenant-a','project-a','att-bad-pending','exec-2',1,'node-1',1,1,'worker-a','PENDING',
		NOW(),NOW(),'target-1',1,'sha256:x','policy-1','budget-1'
	)`); err == nil {
		t.Fatal("PENDING attempt with started_at must be rejected")
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO execution_attempts(
		tenant_id, project_id, attempt_id, execution_id, plan_revision, node_id,
		attempt_number, lease_fence, lease_owner, status, claimed_at,
		target_id, target_revision, target_snapshot_digest,
		policy_decision_id, budget_reservation_id
	) VALUES (
		'tenant-a','project-a','att-bad-running','exec-3',1,'node-1',1,1,'worker-a','RUNNING',
		NOW(),'target-1',1,'sha256:x','policy-1','budget-1'
	)`); err == nil {
		t.Fatal("RUNNING attempt without started_at must be rejected")
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO execution_attempts(
		tenant_id, project_id, attempt_id, execution_id, plan_revision, node_id,
		attempt_number, lease_fence, lease_owner, status, claimed_at, started_at,
		target_id, target_revision, target_snapshot_digest
	) VALUES (
		'tenant-a','project-a','att-missing-governance','exec-gov',1,'node-1',1,1,'worker-a','RUNNING',
		NOW(),NOW(),'target-1',1,'sha256:x'
	)`); err == nil {
		t.Fatal("RUNNING attempt without policy/budget governance refs must be rejected")
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO execution_attempts(
		tenant_id, project_id, attempt_id, execution_id, plan_revision, node_id,
		attempt_number, lease_fence, lease_owner, status, claimed_at, started_at,
		target_id, target_revision, target_snapshot_digest,
		policy_decision_id, budget_reservation_id
	) VALUES (
		'tenant-a','project-a','att-running','exec-4',1,'node-1',1,1,'worker-a','RUNNING',
		NOW(),NOW(),'target-1',1,'sha256:x','policy-1','budget-1'
	)`); err != nil {
		t.Fatalf("valid RUNNING attempt rejected: %v", err)
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO execution_attempts(
		tenant_id, project_id, attempt_id, execution_id, plan_revision, node_id,
		attempt_number, lease_fence, lease_owner, status, claimed_at, finished_at
	) VALUES ('tenant-a','project-a','att-bad-failed','exec-5',1,'node-1',1,1,'worker-a','FAILED',NOW(),NOW())`); err == nil {
		t.Fatal("FAILED attempt without error_class must be rejected")
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO execution_attempts(
		tenant_id, project_id, attempt_id, execution_id, plan_revision, node_id,
		attempt_number, lease_fence, lease_owner, status, claimed_at, started_at,
		finished_at, target_id, target_revision, target_snapshot_digest,
		policy_decision_id, budget_reservation_id, error_class
	) VALUES (
		'tenant-a','project-a','att-failed','exec-5b',1,'node-1',1,1,'worker-a','FAILED',
		NOW(),NOW(),NOW(),'target-1',1,'sha256:x','policy-1','budget-1','TIMEOUT'
	)`); err != nil {
		t.Fatalf("valid FAILED attempt rejected: %v", err)
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO execution_attempts(
		tenant_id, project_id, attempt_id, execution_id, plan_revision, node_id,
		attempt_number, lease_fence, lease_owner, status, claimed_at, finished_at
	) VALUES (
		'tenant-a','project-a','att-abandoned','exec-6',1,'node-1',1,1,'worker-a','ABANDONED',NOW(),NOW()
	)`); err != nil {
		t.Fatalf("ABANDONED attempt before StartAttempt should be valid: %v", err)
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO execution_attempts(
		tenant_id, project_id, attempt_id, execution_id, plan_revision, node_id,
		attempt_number, lease_fence, lease_owner, status, claimed_at, cost, finished_at
	) VALUES (
		'tenant-a','project-a','att-negative','exec-7',1,'node-1',1,1,'worker-a','ABANDONED',NOW(),-1,NOW()
	)`); err == nil {
		t.Fatal("negative attempt usage must be rejected")
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO execution_attempts(
		tenant_id, project_id, attempt_id, execution_id, plan_revision, node_id,
		attempt_number, lease_fence, lease_owner, status, claimed_at
	) VALUES ('tenant-a','project-a','att-duplicate','exec-1',1,'node-1',1,2,'worker-b','PENDING',NOW())`); err == nil {
		t.Fatal("duplicate scoped attempt_number must be rejected")
	}

	var enabled, forced bool
	if err := db.QueryRowContext(ctx, `SELECT relrowsecurity, relforcerowsecurity
		FROM pg_class WHERE oid='execution_attempts'::regclass`).Scan(&enabled, &forced); err != nil {
		t.Fatalf("read execution_attempts RLS flags: %v", err)
	}
	if !enabled || !forced {
		t.Fatalf("execution_attempts RLS must be enabled and forced: enabled=%v forced=%v", enabled, forced)
	}
}

func cleanupExecutionAttemptMigrationFixture(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	_, _ = db.ExecContext(ctx, `DROP TABLE IF EXISTS execution_attempts CASCADE; DROP TABLE IF EXISTS execution_node_runtime CASCADE`)
}
