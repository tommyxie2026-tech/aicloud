package workflow

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

type recordingDurableEngine struct {
	startRequest StartRequest
	starts       int
}

func (e *recordingDurableEngine) Start(_ context.Context, request StartRequest) (StartResult, error) {
	e.startRequest = request
	e.starts++
	workflowID, _ := WorkflowID(request.TaskID)
	return StartResult{WorkflowID: workflowID}, nil
}

func (*recordingDurableEngine) Cancel(context.Context, CancelRequest) error { return nil }

func TestStartDeliveryAdapterUsesTaskIdentityNotOutboxIdempotencyKey(t *testing.T) {
	engine := &recordingDurableEngine{}
	adapter := NewStartDeliveryAdapter(engine)
	payload, _ := json.Marshal(map[string]string{"taskId": "task-a", "traceId": "trace-a"})
	message := domain.OutboxMessage{
		TenantID:       "tenant-a",
		ProjectID:      "project-a",
		TaskID:         "task-a",
		AggregateType:  "Task",
		AggregateID:    "task-a",
		Payload:        payload,
		Destination:    DestinationWorkflowStart,
		IdempotencyKey: "task:create:client-supplied-key",
	}
	if err := adapter.Deliver(context.Background(), message); err != nil {
		t.Fatal(err)
	}
	if engine.starts != 1 {
		t.Fatalf("starts=%d", engine.starts)
	}
	if engine.startRequest.TaskID != "task-a" || engine.startRequest.TraceID != "trace-a" {
		t.Fatalf("unexpected request: %+v", engine.startRequest)
	}
	if engine.startRequest.WorkflowType != TaskExecutionWorkflowType {
		t.Fatalf("workflow type=%q", engine.startRequest.WorkflowType)
	}
}

func TestStartDeliveryAdapterRejectsIdentityMismatch(t *testing.T) {
	engine := &recordingDurableEngine{}
	adapter := NewStartDeliveryAdapter(engine)
	payload, _ := json.Marshal(map[string]string{"taskId": "task-other", "traceId": "trace-a"})
	message := domain.OutboxMessage{
		TenantID:      "tenant-a",
		ProjectID:     "project-a",
		TaskID:        "task-a",
		AggregateType: "Task",
		AggregateID:   "task-a",
		Payload:       payload,
		Destination:   DestinationWorkflowStart,
	}
	if err := adapter.Deliver(context.Background(), message); err == nil {
		t.Fatal("identity mismatch was accepted")
	}
	if engine.starts != 0 {
		t.Fatalf("starts=%d", engine.starts)
	}
}
