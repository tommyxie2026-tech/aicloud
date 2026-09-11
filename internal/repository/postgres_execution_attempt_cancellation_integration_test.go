//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	execution "github.com/tommyxie2026-tech/aicloud/execution"
)

func TestPostgresExecutionAttemptCanCancelBeforeStart(t *testing.T) {
	db, ctx := openExecutionLeaseTestDB(t)
	defer db.Close()
	cleanupExecutionLeaseFixture(t, context.Background(), db)
	defer cleanupExecutionLeaseFixture(t, context.Background(), db)
	createExecutionLeaseFixture(t, ctx, db)

	projectCtx := executionLeaseProjectContext(ctx)
	repo := NewPostgresExecutionNodeLeases(db)
	if err := repo.Register(projectCtx, execution.NodeRuntimeRecord{
		ExecutionRef: "exec-cancel", PlanRevision: 1, NodeRef: "node-cancel",
		State: execution.NodeReady, EffectClass: execution.EffectReadOnly, RetrySafe: true,
	}); err != nil {
		t.Fatalf("register node: %v", err)
	}

	lease, err := repo.Claim(projectCtx, "exec-cancel", 1, "node-cancel", "worker-a", time.Minute)
	if err != nil {
		t.Fatalf("claim node: %v", err)
	}
	before, err := repo.GetAttempt(projectCtx, lease.AttemptRef)
	if err != nil {
		t.Fatalf("get pending attempt: %v", err)
	}
	if before.Attempt.Status != execution.AttemptPending || !before.Attempt.StartedAt.IsZero() {
		t.Fatalf("expected unstarted PENDING attempt, got %+v", before)
	}

	if err := repo.Complete(projectCtx, lease, execution.NodeCancelled, execution.AttemptCompletion{
		Status: execution.AttemptCancelled,
	}); err != nil {
		t.Fatalf("cancel claimed attempt before StartAttempt: %v", err)
	}

	after, err := repo.GetAttempt(projectCtx, lease.AttemptRef)
	if err != nil {
		t.Fatalf("get cancelled attempt: %v", err)
	}
	if after.Attempt.Status != execution.AttemptCancelled || after.Attempt.FinishedAt == nil {
		t.Fatalf("attempt should be durable CANCELLED: %+v", after)
	}
	if !after.Attempt.StartedAt.IsZero() {
		t.Fatalf("pre-start cancellation must not invent started_at: %+v", after)
	}

	runtime, activeLease, err := repo.Get(projectCtx, "exec-cancel", 1, "node-cancel")
	if err != nil {
		t.Fatalf("get cancelled runtime: %v", err)
	}
	if runtime.State != execution.NodeCancelled || activeLease != nil {
		t.Fatalf("node and attempt cancellation must commit atomically: runtime=%+v lease=%+v", runtime, activeLease)
	}
}
