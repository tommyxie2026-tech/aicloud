package execution

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeAttemptCoordinator struct {
	lease          NodeLease
	claimErr       error
	startErr       error
	markStartedErr error
	markCommitErr  error
	releaseErr     error
	completeErr    error

	calls          []string
	lastBinding    AttemptBinding
	lastNodeState  NodeState
	lastCompletion AttemptCompletion
}

func (f *fakeAttemptCoordinator) Claim(_ context.Context, executionID ExecutionID, planRevision int64, nodeID NodeID, owner string, _ time.Duration) (NodeLease, error) {
	f.calls = append(f.calls, "claim")
	if f.claimErr != nil {
		return NodeLease{}, f.claimErr
	}
	lease := f.lease
	if lease.ExecutionRef == "" {
		lease = NodeLease{
			ExecutionRef: executionID, PlanRevision: planRevision, NodeRef: nodeID,
			AttemptRef: "att-1", OwnerWorkerID: owner, Token: "lease-1",
			Fence: 1, AttemptNumber: 1,
		}
	}
	return lease, nil
}

func (f *fakeAttemptCoordinator) StartAttempt(_ context.Context, lease NodeLease, binding AttemptBinding) (AttemptRecord, error) {
	f.calls = append(f.calls, "start")
	f.lastBinding = binding
	if f.startErr != nil {
		return AttemptRecord{}, f.startErr
	}
	return AttemptRecord{Attempt: ExecutionAttempt{ID: lease.AttemptRef, Status: AttemptRunning}}, nil
}

func (f *fakeAttemptCoordinator) MarkEffectStarted(_ context.Context, _ NodeLease) error {
	f.calls = append(f.calls, "effect-started")
	return f.markStartedErr
}

func (f *fakeAttemptCoordinator) MarkEffectCommitted(_ context.Context, _ NodeLease) error {
	f.calls = append(f.calls, "effect-committed")
	return f.markCommitErr
}

func (f *fakeAttemptCoordinator) ReleaseBeforeEffect(_ context.Context, _ NodeLease) error {
	f.calls = append(f.calls, "release")
	return f.releaseErr
}

func (f *fakeAttemptCoordinator) Complete(_ context.Context, _ NodeLease, state NodeState, completion AttemptCompletion) error {
	f.calls = append(f.calls, "complete")
	f.lastNodeState = state
	f.lastCompletion = completion
	return f.completeErr
}

type countingInvoker struct {
	calls  int
	result InvocationResult
	err    error
}

func (i *countingInvoker) Invoke(_ context.Context, _ Execution, _ ExecutionNode, _ ExecutionTarget) (InvocationResult, error) {
	i.calls++
	return i.result, i.err
}

func preparedReadOnlyNode() PreparedNode {
	return PreparedNode{
		Node: ExecutionNode{ID: "node-1", Type: NodeModel, Effect: EffectSpec{Class: EffectReadOnly}},
		Target: ExecutionTarget{
			ID: "target-1", Revision: 2, Type: TargetModel,
			Snapshot: TargetSnapshot{Digest: "sha256:target-1"},
		},
		Policy: PolicyDecisionRecord{
			ID: "policy-1", ExecutionRef: "exec-1", PlanRevision: 1,
			NodeRef: "node-1", Decision: PolicyAllow,
		},
		Estimate: BudgetEstimate{Cost: 2, NodeAttempts: 1},
	}
}

func TestPersistentWorkerClaimsBeforeBudgetAndCompletesReadOnly(t *testing.T) {
	coordinator := &fakeAttemptCoordinator{}
	ledger := NewBudgetLedger(BudgetState{Limit: BudgetLimit{MaxCost: 10, MaxNodeAttempts: 2}})
	invoker := &countingInvoker{result: InvocationResult{Usage: Usage{Cost: 1}}}
	worker := NewPersistentWorker("worker-a", time.Minute, coordinator, NewInMemoryBudgetCoordinator(ledger), invoker)

	result, err := worker.Execute(context.Background(), testExecution(), preparedReadOnlyNode())
	if err != nil {
		t.Fatal(err)
	}
	if result.Lease.AttemptRef != "att-1" || result.ReservationRef != "attempt/att-1" {
		t.Fatalf("attempt lineage not preserved: %+v", result)
	}
	if invoker.calls != 1 {
		t.Fatalf("expected one target invocation, got %d", invoker.calls)
	}
	if coordinator.lastBinding.TargetSnapshot != "sha256:target-1" || coordinator.lastBinding.PolicyDecisionRef != "policy-1" {
		t.Fatalf("StartAttempt binding incomplete: %+v", coordinator.lastBinding)
	}
	if coordinator.lastNodeState != NodeSucceeded || coordinator.lastCompletion.Status != AttemptSucceeded {
		t.Fatalf("unexpected terminal state: node=%s attempt=%s", coordinator.lastNodeState, coordinator.lastCompletion.Status)
	}
	snapshot := ledger.Snapshot()
	if snapshot.Reserved.Cost != 0 || snapshot.Consumed.Cost != 1 || snapshot.Consumed.NodeAttempts != 1 {
		t.Fatalf("unexpected budget accounting: %+v", snapshot)
	}
	wantCalls := []string{"claim", "start", "complete"}
	if !sameStrings(coordinator.calls, wantCalls) {
		t.Fatalf("unexpected coordinator sequence got=%v want=%v", coordinator.calls, wantCalls)
	}
}

func TestPersistentWorkerClaimFailureCannotReserveOrInvoke(t *testing.T) {
	claimErr := errors.New("lease held")
	coordinator := &fakeAttemptCoordinator{claimErr: claimErr}
	ledger := NewBudgetLedger(BudgetState{Limit: BudgetLimit{MaxCost: 10, MaxNodeAttempts: 2}})
	invoker := &countingInvoker{}
	worker := NewPersistentWorker("worker-a", time.Minute, coordinator, NewInMemoryBudgetCoordinator(ledger), invoker)

	_, err := worker.Execute(context.Background(), testExecution(), preparedReadOnlyNode())
	if !errors.Is(err, claimErr) {
		t.Fatalf("expected claim error, got %v", err)
	}
	if invoker.calls != 0 {
		t.Fatalf("losing worker must not invoke target, got %d calls", invoker.calls)
	}
	snapshot := ledger.Snapshot()
	if snapshot.Reserved.Cost != 0 || snapshot.Consumed.Cost != 0 {
		t.Fatalf("losing worker must not touch budget: %+v", snapshot)
	}
	if !sameStrings(coordinator.calls, []string{"claim"}) {
		t.Fatalf("unexpected calls: %v", coordinator.calls)
	}
}

func TestPersistentWorkerBudgetFailureReleasesClaimBeforeStart(t *testing.T) {
	coordinator := &fakeAttemptCoordinator{}
	ledger := NewBudgetLedger(BudgetState{Limit: BudgetLimit{MaxCost: 1, MaxNodeAttempts: 2}})
	invoker := &countingInvoker{}
	worker := NewPersistentWorker("worker-a", time.Minute, coordinator, NewInMemoryBudgetCoordinator(ledger), invoker)

	_, err := worker.Execute(context.Background(), testExecution(), preparedReadOnlyNode())
	if !errors.Is(err, ErrBudgetExceeded) {
		t.Fatalf("expected budget exceeded, got %v", err)
	}
	if invoker.calls != 0 {
		t.Fatalf("budget failure must happen before target invocation")
	}
	if !sameStrings(coordinator.calls, []string{"claim", "release"}) {
		t.Fatalf("claim must be released after budget rejection: %v", coordinator.calls)
	}
}

func TestPersistentWorkerCommittedMutationCompletesSuccess(t *testing.T) {
	coordinator := &fakeAttemptCoordinator{}
	ledger := NewBudgetLedger(BudgetState{Limit: BudgetLimit{MaxCost: 10, MaxNodeAttempts: 2}})
	invoker := &countingInvoker{result: InvocationResult{
		Usage:             Usage{Cost: 2},
		EffectDisposition: EffectDispositionCommitted,
	}}
	worker := NewPersistentWorker("worker-a", time.Minute, coordinator, NewInMemoryBudgetCoordinator(ledger), invoker)
	candidate := preparedReadOnlyNode()
	candidate.Node.Type = NodeTool
	candidate.Node.Effect = EffectSpec{Class: EffectIdempotentMutation, IdempotencyKey: "effect-1", RetrySafe: true}
	candidate.Target.Type = TargetTool

	_, err := worker.Execute(context.Background(), testExecution(), candidate)
	if err != nil {
		t.Fatal(err)
	}
	wantCalls := []string{"claim", "start", "effect-started", "effect-committed", "complete"}
	if !sameStrings(coordinator.calls, wantCalls) {
		t.Fatalf("unexpected mutation sequence got=%v want=%v", coordinator.calls, wantCalls)
	}
	if coordinator.lastNodeState != NodeSucceeded || coordinator.lastCompletion.Status != AttemptSucceeded {
		t.Fatalf("unexpected mutation completion: node=%s attempt=%s", coordinator.lastNodeState, coordinator.lastCompletion.Status)
	}
}

func TestPersistentWorkerUnknownMutationRequiresRecovery(t *testing.T) {
	coordinator := &fakeAttemptCoordinator{}
	ledger := NewBudgetLedger(BudgetState{Limit: BudgetLimit{MaxCost: 10, MaxNodeAttempts: 2}})
	invoker := &countingInvoker{result: InvocationResult{
		Usage:             Usage{Cost: 2},
		EffectDisposition: EffectDispositionUnknown,
	}}
	worker := NewPersistentWorker("worker-a", time.Minute, coordinator, NewInMemoryBudgetCoordinator(ledger), invoker)
	candidate := preparedReadOnlyNode()
	candidate.Node.Type = NodeTool
	candidate.Node.Effect = EffectSpec{Class: EffectIdempotentMutation, IdempotencyKey: "effect-1", RetrySafe: true}
	candidate.Target.Type = TargetTool

	_, err := worker.Execute(context.Background(), testExecution(), candidate)
	if !errors.Is(err, ErrEffectResolutionRequired) {
		t.Fatalf("expected recovery requirement, got %v", err)
	}
	wantCalls := []string{"claim", "start", "effect-started"}
	if !sameStrings(coordinator.calls, wantCalls) {
		t.Fatalf("unknown effect must remain open for recovery, got calls=%v", coordinator.calls)
	}
	snapshot := ledger.Snapshot()
	if snapshot.Reserved.Cost != 0 || snapshot.Consumed.Cost != 2 {
		t.Fatalf("real invocation usage must still be settled: %+v", snapshot)
	}
}

func TestPersistentWorkerMutationNotAppliedCanFailExplicitly(t *testing.T) {
	coordinator := &fakeAttemptCoordinator{}
	ledger := NewBudgetLedger(BudgetState{Limit: BudgetLimit{MaxCost: 10, MaxNodeAttempts: 2}})
	targetErr := &InvocationError{Class: ErrorTargetUnavailable, Err: errors.New("rejected before apply")}
	invoker := &countingInvoker{
		result: InvocationResult{EffectDisposition: EffectDispositionNotApplied},
		err:    targetErr,
	}
	worker := NewPersistentWorker("worker-a", time.Minute, coordinator, NewInMemoryBudgetCoordinator(ledger), invoker)
	candidate := preparedReadOnlyNode()
	candidate.Node.Type = NodeTool
	candidate.Node.Effect = EffectSpec{Class: EffectIdempotentMutation, IdempotencyKey: "effect-1", RetrySafe: true}
	candidate.Target.Type = TargetTool

	_, err := worker.Execute(context.Background(), testExecution(), candidate)
	if !errors.Is(err, targetErr.Err) {
		t.Fatalf("expected invocation error, got %v", err)
	}
	if coordinator.lastNodeState != NodeFailed || coordinator.lastCompletion.Status != AttemptFailed || coordinator.lastCompletion.ErrorClass != ErrorTargetUnavailable {
		t.Fatalf("NOT_APPLIED mutation may fail explicitly: node=%s completion=%+v", coordinator.lastNodeState, coordinator.lastCompletion)
	}
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
