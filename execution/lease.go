package execution

import "time"

// NodeLease is the durable authorization held by one worker to execute one
// node attempt. A lease provides fencing against stale workers; it does not by
// itself provide exactly-once side-effect semantics.
type NodeLease struct {
	ExecutionRef  ExecutionID
	PlanRevision  int64
	NodeRef       NodeID
	OwnerWorkerID string
	Token         string
	Fence         int64
	AttemptNumber int
	ClaimedAt     time.Time
	HeartbeatAt   time.Time
	ExpiresAt     time.Time
}

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
