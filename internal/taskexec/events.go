package taskexec

// Execution events are persisted through the canonical domain.TaskEvent stream.
// Do not introduce a second execution-event store: task_events already provides
// sequencing, tenant/project scope, append-only history and outbox integration.
type EventType string

const (
	EventPlanGenerated      EventType = "ExecutionPlanGenerated"
	EventPolicyAllowed      EventType = "ExecutionPolicyAllowed"
	EventPolicyDenied       EventType = "ExecutionPolicyDenied"
	EventRunStarted         EventType = "ExecutionRunStarted"
	EventStepStarted        EventType = "ExecutionStepStarted"
	EventStepCompleted      EventType = "ExecutionStepCompleted"
	EventStepFailed         EventType = "ExecutionStepFailed"
	EventRunSuspended       EventType = "ExecutionRunSuspended"
	EventApprovalRequested  EventType = "ExecutionApprovalRequested"
	EventApprovalReceived   EventType = "ExecutionApprovalReceived"
	EventRunCompleted       EventType = "ExecutionRunCompleted"
	EventRunFailed          EventType = "ExecutionRunFailed"
	EventRunCancelled       EventType = "ExecutionRunCancelled"
)

func (e EventType) String() string { return string(e) }
