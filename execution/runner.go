package execution

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type EffectDisposition string

const (
	// Empty/NOT_APPLICABLE is expected for PURE and READ_ONLY work.
	EffectDispositionNotApplicable EffectDisposition = ""
	// NOT_APPLIED means the target can positively prove no external mutation was committed.
	EffectDispositionNotApplied EffectDisposition = "NOT_APPLIED"
	// COMMITTED means the target can positively prove the intended mutation was committed.
	EffectDispositionCommitted EffectDisposition = "COMMITTED"
	// UNKNOWN means the target started mutation work but cannot prove whether it committed.
	EffectDispositionUnknown EffectDisposition = "UNKNOWN"
)

type InvocationResult struct {
	Usage             Usage
	ArtifactRefs      []ArtifactID
	EffectDisposition EffectDisposition
}

type TargetInvoker interface {
	Invoke(ctx context.Context, execution Execution, node ExecutionNode, target ExecutionTarget) (InvocationResult, error)
}

type InvocationError struct {
	Class ErrorClass
	Err   error
}

func (e *InvocationError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return string(e.Class)
	}
	return string(e.Class) + ": " + e.Err.Error()
}

func (e *InvocationError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type AttemptRunner struct {
	invoker TargetInvoker
	budget  *BudgetLedger
	now     func() time.Time
}

func NewAttemptRunner(invoker TargetInvoker, budget *BudgetLedger) *AttemptRunner {
	return &AttemptRunner{
		invoker: invoker,
		budget:  budget,
		now:     time.Now,
	}
}

func (r *AttemptRunner) Run(ctx context.Context, execution Execution, ready ReadyNode, attemptID AttemptID, attemptNumber int) (ExecutionAttempt, InvocationResult, error) {
	if r.invoker == nil {
		return ExecutionAttempt{}, InvocationResult{}, errors.New("target invoker is required")
	}
	if r.budget == nil {
		return ExecutionAttempt{}, InvocationResult{}, errors.New("budget ledger is required")
	}
	if ready.Reservation == nil {
		return ExecutionAttempt{}, InvocationResult{}, errors.New("budget reservation is required")
	}
	if attemptID == "" {
		return ExecutionAttempt{}, InvocationResult{}, errors.New("attempt id is required")
	}
	if attemptNumber <= 0 {
		return ExecutionAttempt{}, InvocationResult{}, errors.New("attempt number must be positive")
	}

	started := r.now()
	attempt := ExecutionAttempt{
		ID:             attemptID,
		ExecutionRef:   execution.ID,
		NodeRef:        ready.Node.ID,
		TargetRef:      ready.Target.ID,
		TargetRevision: ready.Target.Revision,
		TargetSnapshot: ready.Target.Snapshot.Digest,
		AttemptNumber:  attemptNumber,
		Status:         AttemptRunning,
		IdempotencyKey: ready.Node.Effect.IdempotencyKey,
		StartedAt:      started,
	}

	result, err := r.invoker.Invoke(ctx, execution, ready.Node, ready.Target)
	finished := r.now()
	attempt.FinishedAt = &finished
	attempt.Usage = result.Usage

	if settleErr := r.budget.Settle(ready.Reservation.ID, actualBudgetForInvocation(ready.Target, result.Usage)); settleErr != nil {
		attempt.Status = AttemptFailed
		attempt.ErrorClass = ErrorUnknown
		return attempt, result, fmt.Errorf("settle budget reservation: %w", settleErr)
	}

	if err == nil {
		attempt.Status = AttemptSucceeded
		return attempt, result, nil
	}

	attempt.Status = AttemptFailed
	attempt.ErrorClass = classifyInvocationError(err)
	return attempt, result, err
}

func actualBudgetForInvocation(target ExecutionTarget, usage Usage) BudgetEstimate {
	actual := BudgetEstimate{
		Cost:         usage.Cost,
		Duration:     usage.Duration,
		NodeAttempts: 1,
	}
	if target.Type == TargetTool {
		actual.ToolCalls = 1
	}
	return actual
}

func classifyInvocationError(err error) ErrorClass {
	if err == nil {
		return ""
	}
	var invocationErr *InvocationError
	if errors.As(err, &invocationErr) && invocationErr.Class != "" {
		return invocationErr.Class
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrorTimeout
	}
	if errors.Is(err, context.Canceled) {
		return ErrorTransient
	}
	return ErrorUnknown
}
