package execution

import "time"

const AttemptAbandoned AttemptStatus = "ABANDONED"

// AttemptRecord is the durable execution history bound to one node lease fence.
// The embedded ExecutionAttempt keeps the public attempt contract while these
// fields preserve distributed-execution lineage required for audit/recovery.
type AttemptRecord struct {
	Attempt              ExecutionAttempt
	PlanRevision         int64
	LeaseFence           int64
	LeaseOwner           string
	PolicyDecisionRef    PolicyDecisionID
	BudgetReservationRef string
	ClaimedAt            time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// AttemptBinding freezes the concrete target and governance snapshots before a
// claimed PENDING attempt may enter RUNNING and invoke an external target.
type AttemptBinding struct {
	TargetRef            TargetID
	TargetRevision       int64
	TargetSnapshot       string
	PolicyDecisionRef    PolicyDecisionID
	BudgetReservationRef string
}

// AttemptCompletion is the durable result recorded when a node attempt reaches
// a terminal technical state. Evidence/Outcome verification remains separate.
type AttemptCompletion struct {
	Status     AttemptStatus
	ErrorClass ErrorClass
	Usage      Usage
}

func (c AttemptCompletion) Terminal() bool {
	switch c.Status {
	case AttemptSucceeded, AttemptFailed, AttemptCancelled, AttemptAbandoned:
		return true
	default:
		return false
	}
}
