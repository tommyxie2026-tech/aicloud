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
	if winner.Fence != 1 || winner.AttemptNumber != 1 || winner.Token == "" || winner.AttemptRef == "" {
		t.Fatalf("unexpected first lease: %+v", winner)
	}
	firstAttempt, err := repo.GetAttempt(projectCtx, winner.AttemptRef)
	if err != nil {
		t.Fatalf("get first attempt: %v", err)
	}
	if firstAttempt.Attempt.Status != execution.AttemptPending || firstAttempt.LeaseFence != 1 {
		t.Fatalf("claim must atomically create pending attempt: %+v", firstAttempt)
	}

	expireExecutionNodeLease(t, ctx, db, "exec-1", 1, "node-1")
	reclaimed, err := repo.Claim(projectCtx, "exec-1", 1, "node-1", "worker-c", time.Minute)
	if err != nil {
		t.Fatalf("reclaim expired lease: %v", err)
	}
	if reclaimed.Fence != 2 || reclaimed.AttemptNumber != 2 || reclaimed.Token == winner.Token || reclaimed.AttemptRef == winner.AttemptRef {
		t.Fatalf("reclaim must issue new token/fence/attempt: old=%+v new=%+v", winner, reclaimed)
	}

	history, err := repo.ListAttempts(projectCtx, "exec-1", 1, "node-1")
	if err != nil {
		t.Fatalf("list attempts after reclaim: %v", err)
	}
	if len(history) != 2 || history[0].Attempt.Status != execution.AttemptAbandoned || history[1].Attempt.Status != execution.AttemptPending {
		t.Fatalf("expected ABANDONED then PENDING attempts, got %+v", history)
	}

	if err := repo.Complete(projectCtx, winner, execution.NodeSucceeded, successCompletion()); !errors.Is(err, ErrNodeLeaseLost) {
		t.Fatalf("stale worker completion error=%v want ErrNodeLeaseLost", err)
	}
	if _, err := repo.StartAttempt(projectCtx, reclaimed, testAttemptBinding()); err != nil {
		t.Fatalf("start reclaimed attempt: %v", err)
	}
	if err := repo.Complete(projectCtx, reclaimed, execution.NodeSucceeded, successCompletion()); err != nil {
		t.Fatalf("current lease completion: %v", err)
	}

	record, lease, err := repo.Get(projectCtx, "exec-1", 1, "node-1")
	if err != nil {
		t.Fatalf("get completed node: %v", err)
	}
	if record.State != execution.NodeSucceeded || lease != nil {
		t.Fatalf("completed runtime state=%+v lease=%+v", record, lease)
	}
	history, err = repo.ListAttempts(projectCtx, "exec-1", 1, "node-1")
	if err != nil {
		t.Fatal(err)
	}
	if history[1].Attempt.Status != execution.AttemptSucceeded {
		t.Fatalf("current attempt should be durable SUCCEEDED, got %+v", history[1])
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
	if _, err := repo.StartAttempt(projectCtx, lease, testAttemptBinding()); err != nil {
		t.Fatalf("start attempt: %v", err)
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
	attempt, err := repo.GetAttempt(projectCtx, lease.AttemptRef)
	if err != nil {
		t.Fatal(err)
	}
	if attempt.Attempt.Status != execution.AttemptRunning {
		t.Fatalf("recovery-required attempt must remain RUNNING for explicit recovery, got %s", attempt.Attempt.Status)
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
	started, err := repo.StartAttempt(projectCtx, lease, testAttemptBinding())
	if err != nil {
		t.Fatalf("start mutation attempt: %v", err)
	}
	if started.Attempt.Status != execution.AttemptRunning || started.Attempt.TargetSnapshot == "" {
		t.Fatalf("unexpected started attempt: %+v", started)
	}
	if err := repo.MarkEffectStarted(projectCtx, lease); err != nil {
		t.Fatalf("mark effect started: %v", err)
	}
	if err := repo.Complete(projectCtx, lease, execution.NodeSucceeded, successCompletion()); !errors.Is(err, ErrNodeEffectNotCommitted) {
		t.Fatalf("success before commit error=%v want ErrNodeEffectNotCommitted", err)
	}
	if err := repo.MarkEffectCommitted(projectCtx, lease); err != nil {
		t.Fatalf("mark effect committed: %v", err)
	}
	if err := repo.Complete(projectCtx, lease, execution.NodeSucceeded, successCompletion()); err != nil {
		t.Fatalf("complete committed mutation: %v", err)
	}
	attempt, err := repo.GetAttempt(projectCtx, lease.AttemptRef)
	if err != nil {
		t.Fatal(err)
	}
	if attempt.Attempt.Status != execution.AttemptSucceeded || attempt.Attempt.EffectStartedAt == nil || attempt.Attempt.EffectCommittedAt == nil {
		t.Fatalf("attempt must retain durable effect lineage: %+v", attempt)
	}
}

func TestPostgresExecutionNodeLeaseReleaseBeforeEffectAbandonsAttempt(t *testing.T) {
	db, ctx := openExecutionLeaseTestDB(t)
	defer db.Close()
	cleanupExecutionLeaseFixture(t, context.Background(), db)
	defer cleanupExecutionLeaseFixture(t, context.Background(), db)
	createExecutionLeaseFixture(t, ctx, db)

	projectCtx := executionLeaseProjectContext(ctx)
	repo := NewPostgresExecutionNodeLeases(db)
	if err := repo.Register(projectCtx, execution.NodeRuntimeRecord{
		ExecutionRef: "exec-release", PlanRevision: 1, NodeRef: "read",
		State: execution.NodeReady, EffectClass: execution.EffectReadOnly, RetrySafe: true,
	}); err != nil {
		t.Fatal(err)
	}
	lease, err := repo.Claim(projectCtx, "exec-release", 1, "read", "worker-a", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.ReleaseBeforeEffect(projectCtx, lease); err != nil {
		t.Fatalf("release before effect: %v", err)
	}
	attempt, err := repo.GetAttempt(projectCtx, lease.AttemptRef)
	if err != nil {
		t.Fatal(err)
	}
	if attempt.Attempt.Status != execution.AttemptAbandoned || attempt.Attempt.FinishedAt == nil {
		t.Fatalf("released attempt must become ABANDONED: %+v", attempt)
	}
	runtime, activeLease, err := repo.Get(projectCtx, "exec-release", 1, "read")
	if err != nil {
		t.Fatal(err)
	}
	if runtime.State != execution.NodeReady || activeLease != nil {
		t.Fatalf("released node must return READY without lease: runtime=%+v lease=%+v", runtime, activeLease)
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

func testAttemptBinding() execution.AttemptBinding {
	return execution.AttemptBinding{
		TargetRef: "target-1", TargetRevision: 1, TargetSnapshot: "sha256:test-target",
		PolicyDecisionRef: "policy-1", BudgetReservationRef: "budget-1",
	}
}

func successCompletion() execution.AttemptCompletion {
	return execution.AttemptCompletion{
		Status: execution.AttemptSucceeded,
		Usage:  execution.Usage{InputTokens: 10, OutputTokens: 5, Cost: 0.25, Duration: time.Second},
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

		CREATE TABLE execution_attempts (
			tenant_id TEXT NOT NULL,
			project_id TEXT NOT NULL,
			attempt_id TEXT NOT NULL,
			execution_id TEXT NOT NULL,
			plan_revision BIGINT NOT NULL,
			node_id TEXT NOT NULL,
			attempt_number INTEGER NOT NULL,
			lease_fence BIGINT NOT NULL,
			lease_owner TEXT NOT NULL,
			status TEXT NOT NULL,
			target_id TEXT,
			target_revision BIGINT,
			target_snapshot_digest TEXT,
			policy_decision_id TEXT,
			budget_reservation_id TEXT,
			idempotency_key TEXT,
			error_class TEXT,
			input_tokens BIGINT NOT NULL DEFAULT 0,
			output_tokens BIGINT NOT NULL DEFAULT 0,
			cost NUMERIC(20,8) NOT NULL DEFAULT 0,
			duration_ms BIGINT NOT NULL DEFAULT 0,
			claimed_at TIMESTAMPTZ NOT NULL,
			started_at TIMESTAMPTZ,
			effect_started_at TIMESTAMPTZ,
			effect_committed_at TIMESTAMPTZ,
			finished_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			PRIMARY KEY (tenant_id, project_id, attempt_id),
			UNIQUE (tenant_id, project_id, execution_id, plan_revision, node_id, attempt_number),
			UNIQUE (tenant_id, project_id, execution_id, plan_revision, node_id, lease_fence)
		);
	`)
	if err != nil {
		t.Fatalf("create execution lease fixture: %v", err)
	}
}

func cleanupExecutionLeaseFixture(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	_, _ = db.ExecContext(ctx, `DROP TABLE IF EXISTS execution_attempts CASCADE; DROP TABLE IF EXISTS execution_node_runtime CASCADE`)
}
