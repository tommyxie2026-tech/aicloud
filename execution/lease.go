package execution

import "time"

// NodeLease is the durable authorization held by one worker to execute one
// node attempt. AttemptRef is created atomically with the claim so every valid
// fence has a durable history record. The token/fence prevents stale workers
// from mutating newer control-plane state. External side effects are fenced
// only when the target can enforce the fence; otherwise EffectClass and
// target-side idempotency are still required. A lease does not provide
// exactly-once side-effect semantics.
type NodeLease struct {
	ExecutionRef  ExecutionID
	PlanRevision  int64
	NodeRef       NodeID
	AttemptRef    AttemptID
	OwnerWorkerID string
	Token         string
	Fence         int64
	AttemptNumber int
	ClaimedAt     time.Time
	HeartbeatAt   time.Time
	ExpiresAt     time.Time
}

// ActiveAt is an advisory local-clock check for observability/UI purposes.
// Authorization decisions must be made by the persistent lease repository,
// which uses the PostgreSQL clock as the authoritative time source.
func (l NodeLease) ActiveAt(now time.Time) bool {
	return l.Token != "" && l.Fence > 0 && !now.Before(l.ClaimedAt) && now.Before(l.ExpiresAt)
}

// NodeRuntimeRecord is the minimal durable node state required by the R1
// distributed execution kernel. The immutable plan remains the source of truth
// for node definition; this record stores only execution-time state.
type NodeRuntimeRecord struct {
	ExecutionRef      ExecutionID
	PlanRevision      int64
	NodeRef           NodeID
	State             NodeState
	EffectClass       EffectClass
	RetrySafe         bool
	IdempotencyKey    string
	AttemptNumber     int
	LeaseFence        int64
	EffectStartedAt   *time.Time
	EffectCommittedAt *time.Time
	UpdatedAt         time.Time
}

func (r NodeRuntimeRecord) Terminal() bool {
	switch r.State {
	case NodeSucceeded, NodeFailed, NodeSkipped, NodeCancelled:
		return true
	default:
		return false
	}
}
