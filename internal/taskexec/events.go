package taskexec

import "time"

type EventType string

const (
	EventTaskCreated EventType = "task.created"
	EventPlanGenerated EventType = "plan.generated"
	EventPolicyAllowed EventType = "policy.allowed"
	EventPolicyDenied EventType = "policy.denied"
	EventRunStarted EventType = "run.started"
	EventStepStarted EventType = "step.started"
	EventStepCompleted EventType = "step.completed"
	EventStepFailed EventType = "step.failed"
	EventRunSuspended EventType = "run.suspended"
	EventApprovalRequested EventType = "human.approval.requested"
	EventApprovalReceived EventType = "human.approval.received"
	EventRunCompleted EventType = "run.completed"
	EventRunFailed EventType = "run.failed"
	EventRunCancelled EventType = "run.cancelled"
)

type Event struct {
	ID string
	Type EventType
	TaskID, PlanID, RunID, TraceID, Tenant, Actor string
	OccurredAt time.Time
	Payload []byte
}

type EventStore interface {
	Append(event Event) error
	ListByRun(runID string) ([]Event, error)
}
