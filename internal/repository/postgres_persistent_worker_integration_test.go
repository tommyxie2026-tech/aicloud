//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	execution "github.com/tommyxie2026-tech/aicloud/execution"
)

type repositoryWorkerInvoker struct {
	calls int
}

func (i *repositoryWorkerInvoker) Invoke(_ context.Context, _ execution.Execution, _ execution.ExecutionNode, _ execution.ExecutionTarget) (execution.InvocationResult, error) {
	i.calls++
	return execution.InvocationResult{Usage: execution.Usage{InputTokens: 12, OutputTokens: 4, Cost: 0.5}}, nil
}

func TestPostgresPersistentWorkerReadOnlyLifecycle(t *testing.T) {
	db, ctx := openExecutionLeaseTestDB(t)
	defer db.Close()
	cleanupExecutionLeaseFixture(t, context.Background(), db)
	defer cleanupExecutionLeaseFixture(t, context.Background(), db)
	createExecutionLeaseFixture(t, ctx, db)

	projectCtx := executionLeaseProjectContext(ctx)
	repo := NewPostgresExecutionNodeLeases(db)
	if err := repo.Register(projectCtx, execution.NodeRuntimeRecord{
		ExecutionRef: "exec-worker", PlanRevision: 1, NodeRef: "inspect",
		State: execution.NodeReady, EffectClass: execution.EffectReadOnly, RetrySafe: true,
	}); err != nil {
		t.Fatalf("register runtime: %v", err)
	}

	exec := execution.Execution{
		ID:      "exec-worker",
		GoalRef: "goal-worker",
		PlanRef: execution.PlanRevisionRef{PlanID: "plan-worker", Revision: 1},
		Status:  execution.ExecutionStatus{Phase: execution.ExecutionRunning},
	}
	candidate := execution.PreparedNode{
		Node: execution.ExecutionNode{
			ID: "inspect", Type: execution.NodeModel,
			Effect: execution.EffectSpec{Class: execution.EffectReadOnly, RetrySafe: true},
		},
		Target: execution.ExecutionTarget{
			ID: "model-private", Revision: 7, Type: execution.TargetModel,
			Snapshot: execution.TargetSnapshot{Digest: "sha256:model-private-r7"},
		},
		Policy: execution.PolicyDecisionRecord{
			ID: "policy-worker-1", ExecutionRef: "exec-worker", PlanRevision: 1,
			NodeRef: "inspect", Decision: execution.PolicyAllow,
		},
		Estimate: execution.BudgetEstimate{Cost: 1, NodeAttempts: 1},
	}
	ledger := execution.NewBudgetLedger(execution.BudgetState{
		Limit: execution.BudgetLimit{MaxCost: 5, MaxNodeAttempts: 2},
	})
	invoker := &repositoryWorkerInvoker{}
	worker := execution.NewPersistentWorker("worker-postgres", time.Minute, repo, ledger, invoker)

	result, err := worker.Execute(projectCtx, exec, candidate)
	if err != nil {
		t.Fatalf("execute persistent worker: %v", err)
	}
	if invoker.calls != 1 {
		t.Fatalf("expected one target invocation, got %d", invoker.calls)
	}
	if result.Lease.AttemptRef == "" || result.ReservationRef != "attempt/"+string(result.Lease.AttemptRef) {
		t.Fatalf("durable attempt/budget lineage missing: %+v", result)
	}

	runtime, activeLease, err := repo.Get(projectCtx, "exec-worker", 1, "inspect")
	if err != nil {
		t.Fatalf("get runtime: %v", err)
	}
	if runtime.State != execution.NodeSucceeded || activeLease != nil {
		t.Fatalf("worker must atomically finish node and release lease: runtime=%+v lease=%+v", runtime, activeLease)
	}

	attempts, err := repo.ListAttempts(projectCtx, "exec-worker", 1, "inspect")
	if err != nil {
		t.Fatalf("list attempts: %v", err)
	}
	if len(attempts) != 1 {
		t.Fatalf("expected one durable attempt, got %d", len(attempts))
	}
	attempt := attempts[0]
	if attempt.Attempt.Status != execution.AttemptSucceeded || attempt.Attempt.TargetRef != "model-private" || attempt.Attempt.TargetRevision != 7 || attempt.Attempt.TargetSnapshot != "sha256:model-private-r7" {
		t.Fatalf("attempt target lineage incomplete: %+v", attempt)
	}
	if attempt.PolicyDecisionRef != "policy-worker-1" || attempt.BudgetReservationRef != result.ReservationRef {
		t.Fatalf("attempt governance lineage incomplete: %+v", attempt)
	}
	if attempt.Attempt.Usage.Cost != 0.5 || attempt.Attempt.Usage.InputTokens != 12 || attempt.Attempt.Usage.OutputTokens != 4 {
		t.Fatalf("attempt usage not persisted: %+v", attempt.Attempt.Usage)
	}

	budget := ledger.Snapshot()
	if budget.Reserved.Cost != 0 || budget.Consumed.Cost != 0.5 || budget.Consumed.NodeAttempts != 1 {
		t.Fatalf("unexpected budget state: %+v", budget)
	}
}
