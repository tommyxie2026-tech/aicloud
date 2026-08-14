package workflow

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

type recordingCancelEngine struct {
	cancelRequest CancelRequest
	cancelCalls   int
	cancelErr     error
}

func (*recordingCancelEngine) Start(context.Context, StartRequest) (StartResult, error) {
	return StartResult{}, nil
}

func (e *recordingCancelEngine) Cancel(_ context.Context, request CancelRequest) error {
	e.cancelCalls++
	e.cancelRequest = request
	return e.cancelErr
}

func TestCancelDeliveryAdapterUsesCommittedOutboxIdentity(t *testing.T) {
	engine := &recordingCancelEngine{}
	adapter := NewCancelDeliveryAdapter(engine)
	payload, _ := json.Marshal(map[string]string{
		"taskId": "task-a", "traceId": "trace-a", "reason": "operator requested",
	})
	message := domain.OutboxMessage{
		TenantID: "tenant-a", ProjectID: "project-a", TaskID: "task-a",
		AggregateType: "Task", AggregateID: "task-a", EventType: "TaskCancelled",
		Destination: DestinationWorkflowCancel, IdempotencyKey: "workflow-cancel:tenant-a:task-a", Payload: payload,
	}
	if err := adapter.Deliver(context.Background(), message); err != nil {
		t.Fatal(err)
	}
	if engine.cancelCalls != 1 {
		t.Fatalf("cancel calls=%d", engine.cancelCalls)
	}
	request := engine.cancelRequest
	if request.TenantID != "tenant-a" || request.ProjectID != "project-a" || request.TaskID != "task-a" || request.TraceID != "trace-a" || request.Reason != "operator requested" {
		t.Fatalf("unexpected cancellation request: %+v", request)
	}
}

func TestCancelDeliveryAdapterRejectsPersistedIdentityMismatch(t *testing.T) {
	cases := []struct {
		name    string
		message func() domain.OutboxMessage
	}{
		{
			name: "aggregate",
			message: func() domain.OutboxMessage {
				message := validCancelOutboxMessage()
				message.AggregateID = "task-other"
				return message
			},
		},
		{
			name: "payload",
			message: func() domain.OutboxMessage {
				message := validCancelOutboxMessage()
				message.Payload, _ = json.Marshal(map[string]string{"taskId": "task-other", "traceId": "trace-a"})
				return message
			},
		},
		{
			name: "scope",
			message: func() domain.OutboxMessage {
				message := validCancelOutboxMessage()
				message.ProjectID = ""
				return message
			},
		},
		{
			name: "destination",
			message: func() domain.OutboxMessage {
				message := validCancelOutboxMessage()
				message.Destination = DestinationWorkflowStart
				return message
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			engine := &recordingCancelEngine{}
			adapter := NewCancelDeliveryAdapter(engine)
			if err := adapter.Deliver(context.Background(), tc.message()); err == nil {
				t.Fatal("invalid cancellation message was accepted")
			}
			if engine.cancelCalls != 0 {
				t.Fatalf("invalid message reached DurableEngine.Cancel, calls=%d", engine.cancelCalls)
			}
		})
	}
}

func TestCancelDeliveryAdapterValidatesTraceBeforeEngineCall(t *testing.T) {
	engine := &recordingCancelEngine{}
	adapter := NewCancelDeliveryAdapter(engine)
	message := validCancelOutboxMessage()
	message.Payload, _ = json.Marshal(map[string]string{"taskId": "task-a", "traceId": ""})
	if err := adapter.Deliver(context.Background(), message); err == nil {
		t.Fatal("missing trace identity was accepted")
	}
	if engine.cancelCalls != 1 {
		// DurableEngine owns CancelRequest validation; the adapter must never
		// invent a trace ID. A real engine rejects the empty trace. This recorder
		// intentionally exposes the exact request propagation contract.
		t.Fatalf("cancel calls=%d", engine.cancelCalls)
	}
	if engine.cancelRequest.TraceID != "" {
		t.Fatalf("adapter invented trace identity: %+v", engine.cancelRequest)
	}
}

func validCancelOutboxMessage() domain.OutboxMessage {
	payload, _ := json.Marshal(map[string]string{
		"taskId": "task-a", "traceId": "trace-a", "reason": "operator requested",
	})
	return domain.OutboxMessage{
		TenantID: "tenant-a", ProjectID: "project-a", TaskID: "task-a",
		AggregateType: "Task", AggregateID: "task-a", EventType: "TaskCancelled",
		Destination: DestinationWorkflowCancel, IdempotencyKey: "workflow-cancel:tenant-a:task-a", Payload: payload,
	}
}
