//go:build integration

package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	execution "github.com/tommyxie2026-tech/aicloud/execution"
)

func TestPostgresExecutionNodeRequeueFailedReadOnly(t *testing.T) {
	db, ctx := openExecutionLeaseTestDB(t)
	defer db.Close()
	cleanupExecutionLeaseFixture(t, context.Background(), db)
	defer cleanupExecutionLeaseFixture(t, context.Background(), db)
	createExecutionLeaseFixture(t, ctx, db)

	projectCtx := executionLeaseProjectContext(ctx)
	repo := NewPostgresExecutionNodeLeases(db)
	if err := repo.Register(projectCtx, execution.NodeRuntimeRecord{
		ExecutionRef: "exec-retry", PlanRevision: 1, NodeRef: "read",
		State: execution.NodeReady, EffectClass: execution.EffectReadOnly, RetrySafe: true,
	}); err != nil {
		t.Fatal(err)
	}
	lease, err := repo.Claim(projectCtx, "exec-retry", 1, "read", "worker-a", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.StartAttempt(projectCtx, lease, testAttemptBinding()); err != nil {
		t.Fatal(err)
	}
	if err := repo.Complete(projectCtx, lease, execution.NodeFailed, execution.AttemptCompletion{
		Status: execution.AttemptFailed, ErrorClass: execution.ErrorTransient,
	}); err != nil {
		t.Fatal(err)
	}
	failed, active, err := repo.Get(projectCtx, "exec-retry", 1, "read")
	if err != nil {
		t.Fatal(err)
	}
	if failed.State != execution.NodeFailed || active != nil {
		t.Fatalf("failed node state=%s lease=%+v", failed.State, active)
	}
	if err := repo.RequeueFailed(projectCtx, "exec-retry", 1, "read"); err != nil {
		t.Fatalf("requeue read-only failure: %v", err)
	}
	ready, active, err := repo.Get(projectCtx, "exec-retry", 1, "read")
	if err != nil {
		t.Fatal(err)
	}
	if ready.State != execution.NodeReady || active != nil || ready.AttemptNumber != 1 {
		t.Fatalf("requeued runtime=%+v lease=%+v", ready, active)
	}
	second, err := repo.Claim(projectCtx, "exec-retry", 1, "read", "worker-b", time.Minute)
	if err != nil {
		t.Fatalf("claim requeued node: %v", err)
	}
	if second.AttemptNumber != 2 || second.Fence != 2 {
		t.Fatalf("retry must preserve monotonic attempt/fence: %+v", second)
	}
}

func TestPostgresExecutionNodeRequeueRefusesMutation(t *testing.T) {
	db, ctx := openExecutionLeaseTestDB(t)
	defer db.Close()
	cleanupExecutionLeaseFixture(t, context.Background(), db)
	defer cleanupExecutionLeaseFixture(t, context.Background(), db)
	createExecutionLeaseFixture(t, ctx, db)

	projectCtx := executionLeaseProjectContext(ctx)
	repo := NewPostgresExecutionNodeLeases(db)
	if err := repo.Register(projectCtx, execution.NodeRuntimeRecord{
		ExecutionRef: "exec-mutation-retry", PlanRevision: 1, NodeRef: "write",
		State: execution.NodeReady, EffectClass: execution.EffectIdempotentMutation,
		RetrySafe: true, IdempotencyKey: "exec-mutation-retry/write",
	}); err != nil {
		t.Fatal(err)
	}
	lease, err := repo.Claim(projectCtx, "exec-mutation-retry", 1, "write", "worker-a", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.StartAttempt(projectCtx, lease, testAttemptBinding()); err != nil {
		t.Fatal(err)
	}
	if err := repo.MarkEffectStarted(projectCtx, lease); err != nil {
		t.Fatal(err)
	}
	if err := repo.Complete(projectCtx, lease, execution.NodeFailed, execution.AttemptCompletion{
		Status: execution.AttemptFailed, ErrorClass: execution.ErrorTransient,
	}); err != nil {
		t.Fatal(err)
	}
	if err := repo.RequeueFailed(projectCtx, "exec-mutation-retry", 1, "write"); !errors.Is(err, ErrNodeRecoveryRequired) {
		t.Fatalf("mutation requeue error=%v want ErrNodeRecoveryRequired", err)
	}
	record, _, err := repo.Get(projectCtx, "exec-mutation-retry", 1, "write")
	if err != nil {
		t.Fatal(err)
	}
	if record.State != execution.NodeFailed {
		t.Fatalf("mutation must remain FAILED for explicit recovery, got %s", record.State)
	}
}
