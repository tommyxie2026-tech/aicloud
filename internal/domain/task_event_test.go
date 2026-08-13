package domain

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestCanonicalTaskStateEvent(t *testing.T) {
	cases := map[TaskStatus]string{
		TaskCreated:         "TaskCreated",
		TaskPlanning:        "TaskPlanningStarted",
		TaskRouting:         "TaskRoutingStarted",
		TaskExecuting:       "TaskExecutionStarted",
		TaskWaitingApproval: "TaskApprovalRequested",
		TaskValidating:      "TaskValidationStarted",
		TaskCompleted:       "TaskCompleted",
		TaskFailed:          "TaskFailed",
		TaskCancelled:       "TaskCancelled",
		TaskExpired:         "TaskExpired",
	}
	for status, want := range cases {
		got, ok := CanonicalTaskStateEvent(status)
		if !ok || got != want {
			t.Fatalf("status %s event=%q ok=%t want=%q", status, got, ok, want)
		}
	}
	if _, ok := CanonicalTaskStateEvent(TaskStatus("UNKNOWN")); ok {
		t.Fatal("unknown state must not produce a canonical event")
	}
}

func TestTaskEventValidation(t *testing.T) {
	now := time.Now().UTC()
	event := TaskEvent{
		EventID: "event-1", TenantID: "tenant-a", ProjectID: "project-a", TaskID: "task-a",
		Sequence: 1, EventType: "TaskCreated",
		Actor:   TaskEventActor{PrincipalType: "user", SubjectID: "user-a"},
		Payload: json.RawMessage(`{"status":"CREATED"}`), TraceID: "trace-a",
		SchemaVersion: 1, OccurredAt: now, CreatedAt: now,
	}
	if err := event.Validate(); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}
	event.Sequence = 0
	if !errors.Is(event.Validate(), ErrInvalidTaskEvent) {
		t.Fatal("zero sequence must be rejected")
	}
}

func TestOutboxMessageValidation(t *testing.T) {
	now := time.Now().UTC()
	message := OutboxMessage{
		OutboxID: "outbox-1", TenantID: "tenant-a", ProjectID: "project-a", TaskID: "task-a",
		AggregateType: "Task", AggregateID: "task-a", EventType: "TaskCreated",
		Payload: json.RawMessage(`{"taskId":"task-a"}`), Destination: "temporal",
		IdempotencyKey: "delivery-task-a-1", Status: OutboxPending,
		AvailableAt: now, CreatedAt: now,
	}
	if err := message.Validate(); err != nil {
		t.Fatalf("valid outbox rejected: %v", err)
	}
	message.Attempts = -1
	if !errors.Is(message.Validate(), ErrInvalidOutboxMessage) {
		t.Fatal("negative attempts must be rejected")
	}
}

func TestIdempotencyRecordValidation(t *testing.T) {
	now := time.Now().UTC()
	record := IdempotencyRecord{
		TenantID: "tenant-a", ProjectID: "project-a", Operation: "POST /tasks",
		Key: "key-a", RequestDigest: "sha256:abc", Status: IdempotencyInProgress,
		CreatedAt: now, ExpiresAt: now.Add(time.Hour),
	}
	if err := record.Validate(); err != nil {
		t.Fatalf("valid idempotency record rejected: %v", err)
	}
	record.ExpiresAt = now
	if !errors.Is(record.Validate(), ErrInvalidIdempotencyRecord) {
		t.Fatal("non-increasing expiry must be rejected")
	}
}
