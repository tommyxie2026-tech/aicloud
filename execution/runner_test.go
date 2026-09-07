package execution

import (
    "context"
    "errors"
    "testing"
)

type fakeInvoker struct {
    result InvocationResult
    err    error
}

func (f fakeInvoker) Invoke(_ context.Context, _ Execution, _ ExecutionNode, _ ExecutionTarget) (InvocationResult, error) {
    return f.result, f.err
}

func TestAttemptRunnerSuccessSettlesBudget(t *testing.T) {
    ledger := NewBudgetLedger(BudgetState{Limit: BudgetLimit{MaxCost: 10, MaxNodeAttempts: 2}})
    reservation, err := ledger.Reserve("r1", "exec-1", "node-1", BudgetEstimate{Cost: 4, NodeAttempts: 1})
    if err != nil {
        t.Fatal(err)
    }

    ready := ReadyNode{
        Node:        ExecutionNode{ID: "node-1", Type: NodeModel},
        Target:      ExecutionTarget{ID: "target-1", Revision: 2, Type: TargetModel, Snapshot: TargetSnapshot{Digest: "sha256:abc"}},
        Reservation: reservation,
    }
    runner := NewAttemptRunner(fakeInvoker{result: InvocationResult{Usage: Usage{Cost: 3}}}, ledger)

    attempt, _, err := runner.Run(context.Background(), testExecution(), ready, "attempt-1", 1)
    if err != nil {
        t.Fatal(err)
    }
    if attempt.Status != AttemptSucceeded {
        t.Fatalf("expected SUCCEEDED, got %s", attempt.Status)
    }
    if attempt.TargetRevision != 2 || attempt.TargetSnapshot != "sha256:abc" {
        t.Fatalf("target snapshot not preserved: %+v", attempt)
    }

    snap := ledger.Snapshot()
    if snap.Reserved.Cost != 0 || snap.Consumed.Cost != 3 || snap.Consumed.NodeAttempts != 1 {
        t.Fatalf("unexpected ledger state: %+v", snap)
    }
}

func TestAttemptRunnerClassifiesInvocationFailure(t *testing.T) {
    ledger := NewBudgetLedger(BudgetState{Limit: BudgetLimit{MaxCost: 10, MaxNodeAttempts: 2}})
    reservation, err := ledger.Reserve("r1", "exec-1", "node-1", BudgetEstimate{Cost: 2, NodeAttempts: 1})
    if err != nil {
        t.Fatal(err)
    }

    ready := ReadyNode{
        Node:        ExecutionNode{ID: "node-1", Type: NodeModel},
        Target:      ExecutionTarget{ID: "target-1", Revision: 1, Type: TargetModel},
        Reservation: reservation,
    }
    expected := errors.New("target unavailable")
    runner := NewAttemptRunner(fakeInvoker{
        result: InvocationResult{Usage: Usage{Cost: 1}},
        err:    &InvocationError{Class: ErrorTargetUnavailable, Err: expected},
    }, ledger)

    attempt, _, err := runner.Run(context.Background(), testExecution(), ready, "attempt-1", 1)
    if !errors.Is(err, expected) {
        t.Fatalf("expected wrapped target error, got %v", err)
    }
    if attempt.Status != AttemptFailed || attempt.ErrorClass != ErrorTargetUnavailable {
        t.Fatalf("unexpected attempt result: %+v", attempt)
    }

    snap := ledger.Snapshot()
    if snap.Consumed.Cost != 1 || snap.Reserved.Cost != 0 {
        t.Fatalf("failed attempt usage must still be accounted: %+v", snap)
    }
}

func TestAttemptRunnerRequiresReservation(t *testing.T) {
    ledger := NewBudgetLedger(BudgetState{Limit: BudgetLimit{MaxCost: 10}})
    runner := NewAttemptRunner(fakeInvoker{}, ledger)
    _, _, err := runner.Run(context.Background(), testExecution(), ReadyNode{Node: ExecutionNode{ID: "n"}}, "attempt-1", 1)
    if err == nil {
        t.Fatal("expected missing reservation error")
    }
}
