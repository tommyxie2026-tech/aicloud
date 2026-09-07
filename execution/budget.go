package execution

import (
	"errors"
	"fmt"
	"sync"
)

var (
	ErrBudgetExceeded     = errors.New("budget exceeded")
	ErrReservationMissing = errors.New("budget reservation not found")
	ErrReservationClosed  = errors.New("budget reservation already settled or released")
)

type BudgetEstimate struct {
	Cost          float64
	NodeAttempts  int
	FrontierCalls int
	ToolCalls     int
}

type BudgetReservation struct {
	ID           string
	ExecutionRef ExecutionID
	NodeRef      NodeID
	Estimate     BudgetEstimate
	Closed       bool
}

type BudgetLedger struct {
	mu           sync.Mutex
	state        BudgetState
	reservations map[string]*BudgetReservation
}

func NewBudgetLedger(state BudgetState) *BudgetLedger {
	return &BudgetLedger{
		state:        state,
		reservations: map[string]*BudgetReservation{},
	}
}

func (l *BudgetLedger) Snapshot() BudgetState {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.state
}

func (l *BudgetLedger) Reserve(id string, executionRef ExecutionID, nodeRef NodeID, estimate BudgetEstimate) (*BudgetReservation, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if id == "" {
		return nil, errors.New("reservation id is required")
	}
	if _, exists := l.reservations[id]; exists {
		return nil, fmt.Errorf("reservation %q already exists", id)
	}
	if estimate.Cost < 0 || estimate.NodeAttempts < 0 || estimate.FrontierCalls < 0 || estimate.ToolCalls < 0 {
		return nil, errors.New("budget estimate cannot contain negative values")
	}

	next := AccountingCounters{
		Cost:          l.state.Reserved.Cost + l.state.Consumed.Cost + estimate.Cost,
		NodeAttempts:  l.state.Reserved.NodeAttempts + l.state.Consumed.NodeAttempts + estimate.NodeAttempts,
		FrontierCalls: l.state.Reserved.FrontierCalls + l.state.Consumed.FrontierCalls + estimate.FrontierCalls,
		ToolCalls:     l.state.Reserved.ToolCalls + l.state.Consumed.ToolCalls + estimate.ToolCalls,
	}
	if err := validateCountersAgainstLimit(next, l.state.Limit); err != nil {
		return nil, err
	}

	reservation := &BudgetReservation{
		ID:           id,
		ExecutionRef: executionRef,
		NodeRef:      nodeRef,
		Estimate:     estimate,
	}
	l.reservations[id] = reservation
	applyEstimate(&l.state.Reserved, estimate, 1)
	return cloneReservation(reservation), nil
}

func (l *BudgetLedger) Settle(id string, actual BudgetEstimate) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	reservation, ok := l.reservations[id]
	if !ok {
		return ErrReservationMissing
	}
	if reservation.Closed {
		return ErrReservationClosed
	}
	if actual.Cost < 0 || actual.NodeAttempts < 0 || actual.FrontierCalls < 0 || actual.ToolCalls < 0 {
		return errors.New("actual usage cannot contain negative values")
	}

	// Release the estimate first, then atomically account actual usage.
	applyEstimate(&l.state.Reserved, reservation.Estimate, -1)
	applyEstimate(&l.state.Consumed, actual, 1)
	reservation.Closed = true

	// Actual usage may exceed the estimate. We still account truthfully; the caller
	// can observe exhaustion and prevent future attempts.
	return nil
}

func (l *BudgetLedger) Release(id string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	reservation, ok := l.reservations[id]
	if !ok {
		return ErrReservationMissing
	}
	if reservation.Closed {
		return ErrReservationClosed
	}
	applyEstimate(&l.state.Reserved, reservation.Estimate, -1)
	reservation.Closed = true
	return nil
}

func (l *BudgetLedger) Exhausted() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	combined := AccountingCounters{
		Cost:          l.state.Reserved.Cost + l.state.Consumed.Cost,
		NodeAttempts:  l.state.Reserved.NodeAttempts + l.state.Consumed.NodeAttempts,
		FrontierCalls: l.state.Reserved.FrontierCalls + l.state.Consumed.FrontierCalls,
		ToolCalls:     l.state.Reserved.ToolCalls + l.state.Consumed.ToolCalls,
	}
	return validateCountersAgainstLimit(combined, l.state.Limit) != nil
}

func validateCountersAgainstLimit(c AccountingCounters, limit BudgetLimit) error {
	if limit.MaxCost > 0 && c.Cost > limit.MaxCost {
		return fmt.Errorf("%w: cost %.4f > %.4f", ErrBudgetExceeded, c.Cost, limit.MaxCost)
	}
	if limit.MaxNodeAttempts > 0 && c.NodeAttempts > limit.MaxNodeAttempts {
		return fmt.Errorf("%w: node attempts %d > %d", ErrBudgetExceeded, c.NodeAttempts, limit.MaxNodeAttempts)
	}
	if limit.MaxFrontierCalls > 0 && c.FrontierCalls > limit.MaxFrontierCalls {
		return fmt.Errorf("%w: frontier calls %d > %d", ErrBudgetExceeded, c.FrontierCalls, limit.MaxFrontierCalls)
	}
	if limit.MaxToolCalls > 0 && c.ToolCalls > limit.MaxToolCalls {
		return fmt.Errorf("%w: tool calls %d > %d", ErrBudgetExceeded, c.ToolCalls, limit.MaxToolCalls)
	}
	return nil
}

func applyEstimate(dst *AccountingCounters, v BudgetEstimate, sign int) {
	dst.Cost += float64(sign) * v.Cost
	dst.NodeAttempts += sign * v.NodeAttempts
	dst.FrontierCalls += sign * v.FrontierCalls
	dst.ToolCalls += sign * v.ToolCalls
}

func cloneReservation(in *BudgetReservation) *BudgetReservation {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}
