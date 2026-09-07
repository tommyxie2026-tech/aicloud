package execution

import (
    "errors"
    "sync"
    "testing"
)

func TestBudgetReservationPreventsConcurrentOversubscription(t *testing.T) {
    ledger := NewBudgetLedger(BudgetState{Limit: BudgetLimit{MaxCost: 10, MaxNodeAttempts: 2}})

    var wg sync.WaitGroup
    errs := make(chan error, 2)
    for i := 0; i < 2; i++ {
        wg.Add(1)
        go func(id string) {
            defer wg.Done()
            _, err := ledger.Reserve(id, "exec-1", "node-1", BudgetEstimate{Cost: 6, NodeAttempts: 1})
            errs <- err
        }(string(rune('a' + i)))
    }
    wg.Wait()
    close(errs)

    successes := 0
    failures := 0
    for err := range errs {
        if err == nil {
            successes++
            continue
        }
        if errors.Is(err, ErrBudgetExceeded) {
            failures++
            continue
        }
        t.Fatalf("unexpected error: %v", err)
    }
    if successes != 1 || failures != 1 {
        t.Fatalf("expected one reservation success and one budget rejection, got success=%d failure=%d", successes, failures)
    }
}

func TestBudgetSettlementAccountsActualAndReleasesReservation(t *testing.T) {
    ledger := NewBudgetLedger(BudgetState{Limit: BudgetLimit{MaxCost: 20, MaxNodeAttempts: 3}})
    if _, err := ledger.Reserve("r1", "exec-1", "node-1", BudgetEstimate{Cost: 8, NodeAttempts: 1}); err != nil {
        t.Fatal(err)
    }
    if err := ledger.Settle("r1", BudgetEstimate{Cost: 5, NodeAttempts: 1}); err != nil {
        t.Fatal(err)
    }

    snap := ledger.Snapshot()
    if snap.Reserved.Cost != 0 || snap.Consumed.Cost != 5 {
        t.Fatalf("unexpected accounting: reserved=%v consumed=%v", snap.Reserved.Cost, snap.Consumed.Cost)
    }
    if err := ledger.Settle("r1", BudgetEstimate{Cost: 5}); !errors.Is(err, ErrReservationClosed) {
        t.Fatalf("expected ErrReservationClosed, got %v", err)
    }
}

func TestActualUsageCanExceedEstimateButBlocksFutureReserve(t *testing.T) {
    ledger := NewBudgetLedger(BudgetState{Limit: BudgetLimit{MaxCost: 10, MaxNodeAttempts: 3}})
    if _, err := ledger.Reserve("r1", "exec-1", "node-1", BudgetEstimate{Cost: 4, NodeAttempts: 1}); err != nil {
        t.Fatal(err)
    }
    if err := ledger.Settle("r1", BudgetEstimate{Cost: 12, NodeAttempts: 1}); err != nil {
        t.Fatal(err)
    }
    if !ledger.Exhausted() {
        t.Fatal("expected ledger to be exhausted after actual usage exceeded limit")
    }
    if _, err := ledger.Reserve("r2", "exec-1", "node-2", BudgetEstimate{Cost: 1, NodeAttempts: 1}); !errors.Is(err, ErrBudgetExceeded) {
        t.Fatalf("expected future reservation to fail, got %v", err)
    }
}
