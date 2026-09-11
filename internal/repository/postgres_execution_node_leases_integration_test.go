//go:build integration

package repository

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	_ "github.com/lib/pq"

	execution "github.com/tommyxie2026-tech/aicloud/execution"
	"github.com/tommyxie2026-tech/aicloud/internal/identity"
)

func TestPostgresExecutionNodeLeaseConcurrentClaimAndFencing(t *testing.T) {
	db, ctx := openExecutionLeaseTestDB(t)
	defer db.Close()
	cleanupExecutionLeaseFixture(t, context.Background(), db)
	defer cleanupExecutionLeaseFixture(t, context.Background(), db)
	createExecutionLeaseFixture(t, ctx, db)

	projectCtx := executionLeaseProjectContext(ctx)
	repo := NewPostgresExecutionNodeLeases(db)
	if err := repo.Register(projectCtx, execution.NodeRuntimeRecord{
		ExecutionRef: "exec-1", PlanRevision: 1, NodeRef: "node-1",
		State: execution.NodeReady, EffectClass: execution.EffectReadOnly, RetrySafe: true,
	}); err != nil {
		t.Fatalf("register node: %v", err)
	}

	var wg sync.WaitGroup
	type result struct {
		lease execution.NodeLease
		err   error
	}
	results := make(chan result, 2)
	for _, owner := range []string{"worker-a", "worker-b"} {
		wg.Add(1)
		go func(owner string) {
			defer wg.Done()
			lease, err := repo.Claim(projectCtx, "exec-1", 1, "node-1", owner, time.Minute)
			results <- result{lease: lease, err: err}
		}(owner)
	}
	wg.Wait()
	close(results)

	var winner execution.NodeLease
	successes := 0
	held := 0
	for got := range results {
		if got.err == nil {
			successes++
			winner = got.lease
			continue
		}
		if errors.Is(got.err, ErrNodeLeaseHeld) {
			held++
			continue
		}
		t.Fatalf("unexpected claim error: %v", got.err)
	}
	if successes != 1 || held != 1 {
		t.Fatalf("expected one winner and one held error, successes=%d held=%d", successes, held)
	}
	if winner.Fence != 1 || winner.AttemptNumber != 1 || winner.Token == "" {
		t.Fatalf("unexpected first lease: %+v", winner)
	}

	expireExecutionNodeLease(t, ctx, db, "exec-1", 1, "node-1")
	reclaimed, err := repo.Claim(projectCtx, "exec-1", 1, "node-1", "worker-c", time.Minute)
	if err != nil {
		t.Fatalf("reclaim expired lease: %v", err)
	}
	if reclaimed.Fence != 2 || reclaimed.AttemptNumber != 2 || reclaimed.Token == winner.Token {
		t.Fatalf("reclaim must issue new token/fence/attempt: old=%+v new=%+v", winner, reclaimed)
	}

	if err := repo.Complete(projectCtx, winner, execution.NodeSucceeded); !errors.Is(err, ErrNodeLeaseLost) {
		t.Fatalf("stale worker completion error=%v want ErrNodeLeaseLost", err)
	}
	if err := repo.Complete(projectCtx, reclaimed, execution.NodeSucceeded); err != nil {
		t.Fatalf("current lease completion: %v", err)
	}

	record, lease, err := repo.Get(projectCtx, "exec-1", 1, "node-1")
	if err != nil {
		t.Fatalf("get completed node: %v", err)
	}
	if record.State != execution.NodeSucceeded || lease != nil {
		t.Fatalf("completed runtime state=%+v lease=%+v", record, lease)
	}
}

func TestPostgresExecutionNodeLeaseNonIdempotentExpiryRequiresRecovery(t *testing.T) {
	db, ctx := openExecutionLeaseTestDB(t)
	defer db.Close()
	cleanupExecutionLeaseFixture(t, context.Background(), db)
	defer cleanupExecutionLeaseFixture(t, context.Background(), db)
	createExecutionLeaseFixture(t, ctx, db)

	projectCtx := executionLeaseProjectContext(ctx)
	repo := NewPostgresExecutionNodeLeases(db)
	if err := repo.Register(projectCtx, execution.NodeRuntimeRecord{
		ExecutionRef: "exec-2", PlanRevision: 1, NodeRef: "restart-prod",
		State: execution.NodeReady, EffectClass: execution.EffectNonIdempotentMutation, RetrySafe: false,
	}); err != nil {
		t.Fatalf("register node: %v", err)
	}

	lease, err := repo.Claim(projectCtx, "exec-2", 1, "restart-prod", "worker-a", time.Minute)
	if err != nil {
		t.Fatalf("claim non-idempotent node: %v", err)
	}
	if err := repo.MarkEffectStarted(projectCtx, lease); err != nil {
		t.Fatalf("mark effect started: %v", err)
	}
	if err := repo.ReleaseBeforeEffect(projectCtx, lease); !errors.Is(err, ErrNodeLeaseLost) {
		t.Fatalf("release after effect started error=%v want ErrNodeLeaseLost", err)
	}

	expireExecutionNodeLease(t, ctx, db, "exec-2", 1, "restart-prod")
	_, err = repo.Claim(projectCtx, "exec-2", 1, "restart-prod", "worker-b", time.Minute)
	if !errors.Is(err, ErrNodeRecoveryRequired) {
		t.Fatalf("expired non-idempotent claim error=%v want ErrNodeRecoveryRequired", err)
	}
}

func TestPostgresExecutionNodeLeaseMutationRequiresCommittedEffectForSuccess(t *testing.T) {
	db, ctx := openExecutionLeaseTestDB(t)
	defer db.Close()
	cleanupExecutionLeaseFixture(t, context.Background(), db)
	defer cleanupExecutionLeaseFixture(t, context.Background(), db)
	createExecutionLeaseFixture(t, ctx, db)

	projectCtx := executionLeaseProjectContext(ctx)
	repo := NewPostgresExecutionNodeLeases(db)
	if err := repo.Register(projectCtx, execution.NodeRuntimeRecord{
		ExecutionRef: "exec-4", PlanRevision: 1, NodeRef: "apply-change",
		State: execution.NodeReady, EffectClass: execution.EffectIdempotentMutation,
		RetrySafe: true, IdempotencyKey: "exec-4/apply-change",
	}); err != nil {
		t.Fatalf("register mutation node: %v", err)
	}

	lease, err := repo.Claim(projectCtx, "exec-4", 1, "apply-change", "worker-a", time.Minute)
	if err != nil {
		t.Fatalf("claim mutation: %v", err)
	}
	if err := repo.MarkEffectStarted(projectCtx, lease); err != nil {
		t.Fatalf("mark effect started: %v", err)
	}
	if err := repo.Complete(projectCtx, lease, execution.NodeSucceeded); !errors.Is(err, ErrNodeEffectNotCommitted) {
		t.Fatalf("success before commit error=%v want ErrNodeEffectNotCommitted", err)
	}
	if err := repo.MarkEffectCommitted(projectCtx, lease); err != nil {
		t.Fatalf("mark effect committed: %v", err)
	}
	if err := repo.Complete(projectCtx, lease, execution.NodeSucceeded); err != nil {
		t.Fatalf("complete committed mutation: %v", err)
	}
}

func TestPostgresExecutionNodeLeaseRenewCannotReviveExpiredLease(t *testing.T) {
	db, ctx := openExecutionLeaseTestDB(t)
	defer db.Close()
	cleanupExecutionLeaseFixture(t, context.Background(), db)
	defer cleanupExecutionLeaseFixture(t, context.Background(), db)
	createExecutionLeaseFixture(t, ctx, db)

	projectCtx := executionLeaseProjectContext(ctx)
	repo := NewPostgresExecutionNodeLeases(db)
	if err := repo.Register(projectCtx, execution.NodeRuntimeRecord{
		ExecutionRef: "exec-3", PlanRevision: 2, NodeRef: "analyze",
		State: execution.NodeReady, EffectClass: execution.EffectPure, RetrySafe: true,
	}); err != nil {
		t.Fatalf("register node: %v", err)
	}

	lease, err := repo.Claim(projectCtx, "exec-3", 2, "analyze", "worker-a", time.Minute)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	renewed, err := repo.Renew(projectCtx, lease, 2*time.Minute)
	if err != nil {
		t.Fatalf("renew active lease: %v", err)
	}
	if !renewed.ExpiresAt.After(lease.ExpiresAt) {
		t.Fatalf("renewal must extend expiry: old=%s new=%s", lease.ExpiresAt, renewed.ExpiresAt)
	}

	expireExecutionNodeLease(t, ctx, db, "exec-3", 2, "analyze")
	if _, err := repo.Renew(projectCtx, renewed, time.Minute); !errors.Is(err, ErrNodeLeaseLost) {
		t.Fatalf("expired renewal error=%v want ErrNodeLeaseLost", err)
	}
}

func expireExecutionNodeLease(t *testing.T, ctx context.Context, db *sql.DB, executionID string, planRevision int64, nodeID string) {
	t.Helper()
	if _, err := db.ExecContext(ctx, `UPDATE execution_node_runtime
		SET claimed_at=clock_timestamp()-INTERVAL '2 minutes',
			heartbeat_at=clock_timestamp()-INTERVAL '90 seconds',
			lease_expires_at=clock_timestamp()-INTERVAL '1 minute'
		WHERE execution_id=$1 AND plan_revision=$2 AND node_id=$3`, executionID, planRevision, nodeID); err != nil {
		t.Fatalf("expire execution node lease: %v", err)
	}
}

func openExecutionLeaseTestDB(t *testing.T) (*sql.DB, context.Context) {
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
		db.Close()
		t.Fatalf("ping PostgreSQL: %v", err)
	}
	return db, ctx
}

func executionLeaseProjectContext(ctx context.Context) context.Context {
	return identity.WithPrincipal(ctx, identity.Principal{
		Type: identity.PrincipalServiceAccount, SubjectID: "execution-worker",
		TenantID: "tenant-exec", ProjectID: "project-exec", AuthnMethod: "integration-test", Issuer: "test",
	})
}

func createExecutionLeaseFixture(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	_, err := db.ExecContext(ctx, `
		CREATE TABLE execution_node_runtime (
			tenant_id TEXT NOT NULL,
			project_id TEXT NOT NULL,
			execution_id TEXT NOT NULL,
			plan_revision BIGINT NOT NULL,
			node_id TEXT NOT NULL,
			state TEXT NOT NULL,
			effect_class TEXT NOT NULL,
			retry_safe BOOLEAN NOT NULL,
			idempotency_key TEXT,
			attempt_number INTEGER NOT NULL DEFAULT 0,
			lease_owner TEXT,
			lease_token TEXT,
			lease_fence BIGINT NOT NULL DEFAULT 0,
			claimed_at TIMESTAMPTZ,
			heartbeat_at TIMESTAMPTZ,
			lease_expires_at TIMESTAMPTZ,
			effect_started_at TIMESTAMPTZ,
			effect_committed_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (tenant_id, project_id, execution_id, plan_revision, node_id)
		);
	`)
	if err != nil {
		t.Fatalf("create execution lease fixture: %v", err)
	}
}

func cleanupExecutionLeaseFixture(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	_, _ = db.ExecContext(ctx, `DROP TABLE IF EXISTS execution_node_runtime CASCADE`)
}
