package execution

import (
	"context"
	"time"
)

// ExecutionProvider is the stable boundary between aicloud planning/routing
// and an execution backend. Provider completion is not a Verified Outcome.
type ExecutionProvider interface {
	Submit(context.Context, ExecutionRequest) (ExecutionRef, error)
	Get(context.Context, ExecutionRef) (ExecutionStatus, error)
	Cancel(context.Context, ExecutionRef) error
	Watch(context.Context, ExecutionRef) (<-chan ExecutionEvent, error)
}

type ExecutionRequest struct {
	ID           string
	WorkloadType string
	Capability   []string
	Goal         string
	ContextRefs  []string
	WorkspaceRef string
	Timeout      time.Duration
	Isolation    string
	Permissions  []string
	TraceID      string
}

type ExecutionRef struct {
	Provider string
	ID       string
}

type ExecutionPhase string

const (
	ExecutionQueued       ExecutionPhase = "QUEUED"
	ExecutionAssigned     ExecutionPhase = "ASSIGNED"
	ExecutionStarted      ExecutionPhase = "STARTED"
	ExecutionWaitingInput ExecutionPhase = "WAITING_INPUT"
	ExecutionCompleted    ExecutionPhase = "COMPLETED"
	ExecutionFailed       ExecutionPhase = "FAILED"
	ExecutionCancelled    ExecutionPhase = "CANCELLED"
)

type ExecutionStatus struct {
	Ref       ExecutionRef
	Phase     ExecutionPhase
	Message   string
	Artifacts []ArtifactRef
	UpdatedAt time.Time
}

type ArtifactRef struct {
	URI      string
	Checksum string
	Kind     string
}

type ExecutionEvent struct {
	Ref       ExecutionRef
	Phase     ExecutionPhase
	Message   string
	Artifacts []ArtifactRef
	At        time.Time
}

// OutcomeStatus intentionally lives above ExecutionStatus. An execution
// backend may complete successfully while verification rejects its result.
type OutcomeStatus string

const (
	OutcomePending  OutcomeStatus = "PENDING"
	OutcomeVerified OutcomeStatus = "VERIFIED"
	OutcomeRejected OutcomeStatus = "REJECTED"
)
