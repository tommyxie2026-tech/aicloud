package execution

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrPersistentWorkerConfig   = errors.New("persistent worker configuration is invalid")
	ErrPreparedNodeInvalid      = errors.New("prepared node is invalid")
	ErrEffectResolutionRequired = errors.New("mutation effect requires recovery resolution")
)

// AttemptExecutionCoordinator is the durable execution-authority boundary used
// by PersistentWorker. The PostgreSQL node lease repository implements this
// contract without creating an execution-package dependency on storage code.
type AttemptExecutionCoordinator interface {
	Claim(ctx context.Context, executionID ExecutionID, planRevision int64, nodeID NodeID, owner string, leaseDuration time.Duration) (NodeLease, error)
	StartAttempt(ctx context.Context, lease NodeLease, binding AttemptBinding) (AttemptRecord, error)
	MarkEffectStarted(ctx context.Context, lease NodeLease) error
	MarkEffectCommitted(ctx context.Context, lease NodeLease) error
	ReleaseBeforeEffect(ctx context.Context, lease NodeLease) error
	Complete(ctx context.Context, lease NodeLease, terminal NodeState, completion AttemptCompletion) error
}

// BudgetCoordinator is the request-scoped accounting boundary used by the
// persistent worker. Context is explicit because production implementations
// must preserve cancellation/deadline and tenant/project identity into storage
// transactions. Request identity must never be hidden in coordinator state.
type BudgetCoordinator interface {
	Reserve(ctx context.Context, id string, executionRef ExecutionID, nodeRef NodeID, estimate BudgetEstimate) (*BudgetReservation, error)
	Settle(ctx context.Context, id string, actual BudgetEstimate) error
	Release(ctx context.Context, id string) error
}

type PersistentWorker struct {
	workerID    string
	leaseTTL    time.Duration
	coordinator AttemptExecutionCoordinator
	budget      BudgetCoordinator
	invoker     TargetInvoker
}

type WorkerExecutionResult struct {
	Lease          NodeLease
	ReservationRef string
	Invocation     InvocationResult
}

func NewPersistentWorker(workerID string, leaseTTL time.Duration, coordinator AttemptExecutionCoordinator, budget BudgetCoordinator, invoker TargetInvoker) *PersistentWorker {
	return &PersistentWorker{
		workerID:    workerID,
		leaseTTL:    leaseTTL,
		coordinator: coordinator,
		budget:      budget,
		invoker:     invoker,
	}
}

// Execute performs the persistent single-node lifecycle:
//
// Prepare (already done) -> Claim -> Reserve -> StartAttempt -> Invoke
// -> Settle -> Effect/Attempt completion.
//
// Claim happens before Reserve so losing workers cannot consume budget for a
// node they never received authority to execute. Reservation IDs are derived
// from the durable AttemptRef, making the accounting lineage explicit.
func (w *PersistentWorker) Execute(ctx context.Context, execution Execution, candidate PreparedNode) (WorkerExecutionResult, error) {
	if err := w.validate(execution, candidate); err != nil {
		return WorkerExecutionResult{}, err
	}

	lease, err := w.coordinator.Claim(
		ctx,
		execution.ID,
		execution.PlanRef.Revision,
		candidate.Node.ID,
		w.workerID,
		w.leaseTTL,
	)
	if err != nil {
		return WorkerExecutionResult{}, err
	}
	result := WorkerExecutionResult{Lease: lease}

	reservationID := "attempt/" + string(lease.AttemptRef)
	reservation, err := w.budget.Reserve(ctx, reservationID, execution.ID, candidate.Node.ID, candidate.Estimate)
	if err != nil {
		cleanupErr := w.coordinator.ReleaseBeforeEffect(ctx, lease)
		return result, errors.Join(err, cleanupErr)
	}
	result.ReservationRef = reservation.ID

	binding := AttemptBinding{
		TargetRef:            candidate.Target.ID,
		TargetRevision:       candidate.Target.Revision,
		TargetSnapshot:       candidate.Target.Snapshot.Digest,
		PolicyDecisionRef:    candidate.Policy.ID,
		BudgetReservationRef: reservation.ID,
	}
	if _, err := w.coordinator.StartAttempt(ctx, lease, binding); err != nil {
		budgetErr := w.budget.Release(ctx, reservation.ID)
		leaseErr := w.coordinator.ReleaseBeforeEffect(ctx, lease)
		return result, errors.Join(err, budgetErr, leaseErr)
	}

	mutation := isMutationEffect(candidate.Node.Effect.Class)
	if mutation {
		if err := w.coordinator.MarkEffectStarted(ctx, lease); err != nil {
			// No external target call has happened yet. Account the started
			// technical attempt and try to close it as FAILED.
			settleErr := w.budget.Settle(ctx, reservation.ID, BudgetEstimate{NodeAttempts: 1})
			completeErr := w.coordinator.Complete(ctx, lease, NodeFailed, AttemptCompletion{
				Status:     AttemptFailed,
				ErrorClass: ErrorUnknown,
			})
			return result, errors.Join(err, settleErr, completeErr)
		}
	}

	invocation, invokeErr := w.invoker.Invoke(ctx, execution, candidate.Node, candidate.Target)
	result.Invocation = invocation

	if err := w.budget.Settle(ctx, reservation.ID, actualBudgetForInvocation(candidate.Target, invocation.Usage)); err != nil {
		// Invocation may already have produced a real-world effect. Do not
		// manufacture a terminal Attempt when accounting durability is unknown.
		return result, fmt.Errorf("settle attempt budget: %w", err)
	}

	if mutation {
		return w.finishMutation(ctx, lease, candidate, result, invokeErr)
	}
	return w.finishReadOnly(ctx, lease, result, invokeErr)
}

func (w *PersistentWorker) finishReadOnly(ctx context.Context, lease NodeLease, result WorkerExecutionResult, invokeErr error) (WorkerExecutionResult, error) {
	if invokeErr == nil {
		if err := w.coordinator.Complete(ctx, lease, NodeSucceeded, AttemptCompletion{
			Status: AttemptSucceeded,
			Usage:  result.Invocation.Usage,
		}); err != nil {
			return result, err
		}
		return result, nil
	}

	if err := w.coordinator.Complete(ctx, lease, NodeFailed, AttemptCompletion{
		Status:     AttemptFailed,
		ErrorClass: classifyInvocationError(invokeErr),
		Usage:      result.Invocation.Usage,
	}); err != nil {
		return result, errors.Join(invokeErr, err)
	}
	return result, invokeErr
}

func (w *PersistentWorker) finishMutation(ctx context.Context, lease NodeLease, candidate PreparedNode, result WorkerExecutionResult, invokeErr error) (WorkerExecutionResult, error) {
	switch result.Invocation.EffectDisposition {
	case EffectDispositionCommitted:
		if err := w.coordinator.MarkEffectCommitted(ctx, lease); err != nil {
			return result, errors.Join(ErrEffectResolutionRequired, err)
		}
		if invokeErr != nil {
			// The external effect is known committed but the invocation still
			// returned an error. Keep the Attempt open for explicit recovery /
			// verification rather than guessing SUCCESS or FAILED.
			return result, errors.Join(ErrEffectResolutionRequired, invokeErr)
		}
		if err := w.coordinator.Complete(ctx, lease, NodeSucceeded, AttemptCompletion{
			Status: AttemptSucceeded,
			Usage:  result.Invocation.Usage,
		}); err != nil {
			return result, err
		}
		return result, nil

	case EffectDispositionNotApplied:
		if invokeErr == nil {
			return result, fmt.Errorf("%w: target reported NOT_APPLIED without an invocation error", ErrEffectResolutionRequired)
		}
		if err := w.coordinator.Complete(ctx, lease, NodeFailed, AttemptCompletion{
			Status:     AttemptFailed,
			ErrorClass: classifyInvocationError(invokeErr),
			Usage:      result.Invocation.Usage,
		}); err != nil {
			return result, errors.Join(invokeErr, err)
		}
		return result, invokeErr

	case EffectDispositionUnknown, EffectDispositionNotApplicable:
		// Empty disposition is treated as UNKNOWN for mutations. Safety wins
		// over convenience: never infer that a mutation failed or succeeded.
		if invokeErr != nil {
			return result, errors.Join(ErrEffectResolutionRequired, invokeErr)
		}
		return result, ErrEffectResolutionRequired

	default:
		return result, fmt.Errorf("%w: unknown effect disposition %q for node %s", ErrEffectResolutionRequired, result.Invocation.EffectDisposition, candidate.Node.ID)
	}
}

func (w *PersistentWorker) validate(execution Execution, candidate PreparedNode) error {
	if w == nil || strings.TrimSpace(w.workerID) == "" || w.leaseTTL <= 0 || w.coordinator == nil || w.budget == nil || w.invoker == nil {
		return ErrPersistentWorkerConfig
	}
	if execution.ID == "" || execution.PlanRef.Revision < 1 || candidate.Node.ID == "" {
		return fmt.Errorf("%w: execution, plan revision and node are required", ErrPreparedNodeInvalid)
	}
	if candidate.Policy.ID == "" || candidate.Policy.Decision != PolicyAllow {
		return fmt.Errorf("%w: an ALLOW policy decision with id is required", ErrPreparedNodeInvalid)
	}
	if candidate.Policy.ExecutionRef != "" && candidate.Policy.ExecutionRef != execution.ID {
		return fmt.Errorf("%w: policy execution mismatch", ErrPreparedNodeInvalid)
	}
	if candidate.Policy.PlanRevision > 0 && candidate.Policy.PlanRevision != execution.PlanRef.Revision {
		return fmt.Errorf("%w: policy plan revision mismatch", ErrPreparedNodeInvalid)
	}
	if candidate.Policy.NodeRef != "" && candidate.Policy.NodeRef != candidate.Node.ID {
		return fmt.Errorf("%w: policy node mismatch", ErrPreparedNodeInvalid)
	}
	if candidate.Target.ID == "" || candidate.Target.Revision < 1 || strings.TrimSpace(candidate.Target.Snapshot.Digest) == "" {
		return fmt.Errorf("%w: target id, revision and immutable snapshot digest are required", ErrPreparedNodeInvalid)
	}
	return nil
}

func isMutationEffect(effect EffectClass) bool {
	return effect == EffectIdempotentMutation || effect == EffectNonIdempotentMutation
}
