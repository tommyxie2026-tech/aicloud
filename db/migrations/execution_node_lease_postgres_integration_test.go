//go:build integration

package migrations

import (
	"context"
	"database/sql"
	"testing"
)

func TestExecutionNodeLeaseMigrationPostgresConstraints(t *testing.T) {
	db, ctx := openIntegrationDB(t)
	defer db.Close()
	cleanupExecutionNodeLeaseMigrationFixture(t, context.Background(), db)
	defer cleanupExecutionNodeLeaseMigrationFixture(t, context.Background(), db)

	body, err := migrationFiles.ReadFile("017_execution_node_runtime_leases.sql")
	if err != nil {
		t.Fatalf("read migration 017: %v", err)
	}
	if _, err := db.ExecContext(ctx, string(body)); err != nil {
		t.Fatalf("execute migration 017: %v", err)
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO execution_node_runtime(
		tenant_id, project_id, execution_id, plan_revision, node_id, state,
		effect_class, retry_safe
	) VALUES ('tenant-a','project-a','exec-1',1,'ready','READY','PURE',true)`); err != nil {
		t.Fatalf("valid READY row rejected: %v", err)
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO execution_node_runtime(
		tenant_id, project_id, execution_id, plan_revision, node_id, state,
		effect_class, retry_safe
	) VALUES ('tenant-a','project-a','exec-1',1,'running-no-lease','RUNNING','PURE',true)`); err == nil {
		t.Fatal("RUNNING without a complete lease must be rejected")
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO execution_node_runtime(
		tenant_id, project_id, execution_id, plan_revision, node_id, state,
		effect_class, retry_safe, lease_owner, lease_token, claimed_at, heartbeat_at, lease_expires_at
	) VALUES (
		'tenant-a','project-a','exec-1',1,'ready-with-lease','READY','PURE',true,
		'worker-a','token-a',NOW(),NOW(),NOW()+INTERVAL '1 minute'
	)`); err == nil {
		t.Fatal("non-RUNNING row with a lease must be rejected")
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO execution_node_runtime(
		tenant_id, project_id, execution_id, plan_revision, node_id, state,
		effect_class, retry_safe, lease_owner, lease_token, lease_fence,
		claimed_at, heartbeat_at, lease_expires_at
	) VALUES (
		'tenant-a','project-a','exec-1',1,'running','RUNNING','PURE',true,
		'worker-a','token-a',1,NOW(),NOW(),NOW()+INTERVAL '1 minute'
	)`); err != nil {
		t.Fatalf("valid RUNNING lease row rejected: %v", err)
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO execution_node_runtime(
		tenant_id, project_id, execution_id, plan_revision, node_id, state,
		effect_class, retry_safe
	) VALUES (
		'tenant-a','project-a','exec-1',1,'unsafe-mutation','READY','NON_IDEMPOTENT_MUTATION',true
	)`); err == nil {
		t.Fatal("non-idempotent mutation must not be retry-safe")
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO execution_node_runtime(
		tenant_id, project_id, execution_id, plan_revision, node_id, state,
		effect_class, retry_safe
	) VALUES (
		'tenant-a','project-a','exec-1',1,'missing-key','READY','IDEMPOTENT_MUTATION',true
	)`); err == nil {
		t.Fatal("idempotent mutation without idempotency key must be rejected")
	}
}

func cleanupExecutionNodeLeaseMigrationFixture(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	_, _ = db.ExecContext(ctx, `DROP TABLE IF EXISTS execution_node_runtime CASCADE`)
}
