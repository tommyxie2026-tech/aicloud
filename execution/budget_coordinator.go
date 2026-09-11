package execution

import (
	"context"
	"errors"
)

// InMemoryBudgetCoordinator adapts the process-local BudgetLedger to the
// request-scoped BudgetCoordinator used by PersistentWorker. It is suitable for
// tests/bootstrap only; distributed workers should use a shared durable
// implementation.
type InMemoryBudgetCoordinator struct {
	ledger *BudgetLedger
}

func NewInMemoryBudgetCoordinator(ledger *BudgetLedger) *InMemoryBudgetCoordinator {
	return &InMemoryBudgetCoordinator{ledger: ledger}
}

func (c *InMemoryBudgetCoordinator) Reserve(ctx context.Context, id string, executionRef ExecutionID, nodeRef NodeID, estimate BudgetEstimate) (*BudgetReservation, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	if c == nil || c.ledger == nil {
		return nil, errors.New("budget ledger is required")
	}
	return c.ledger.Reserve(id, executionRef, nodeRef, estimate)
}

func (c *InMemoryBudgetCoordinator) Settle(ctx context.Context, id string, actual BudgetEstimate) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if c == nil || c.ledger == nil {
		return errors.New("budget ledger is required")
	}
	return c.ledger.Settle(id, actual)
}

func (c *InMemoryBudgetCoordinator) Release(ctx context.Context, id string) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if c == nil || c.ledger == nil {
		return errors.New("budget ledger is required")
	}
	return c.ledger.Release(id)
}

func contextError(ctx context.Context) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	return ctx.Err()
}

var _ BudgetCoordinator = (*InMemoryBudgetCoordinator)(nil)
