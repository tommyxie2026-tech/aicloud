package workflow

import (
	"context"
	"errors"
	"testing"
)

func TestWorkflowIDIsDeterministicPerTask(t *testing.T) {
	first, err := WorkflowID("task-123")
	if err != nil {
		t.Fatal(err)
	}
	second, err := WorkflowID("task-123")
	if err != nil {
		t.Fatal(err)
	}
	if first != "task/task-123" || second != first {
		t.Fatalf("workflow IDs first=%q second=%q", first, second)
	}
}

func TestWorkflowIDRejectsMissingTask(t *testing.T) {
	if _, err := WorkflowID(" "); !errors.Is(err, ErrInvalidStartRequest) {
		t.Fatalf("error=%v", err)
	}
}

func TestNoopDurableEngineValidatesTrustedScope(t *testing.T) {
	engine := NoopDurableEngine{}
	request := StartRequest{
		TenantID:     "tenant-a",
		ProjectID:    "project-a",
		TaskID:       "task-a",
		TraceID:      "trace-a",
		WorkflowType: TaskExecutionWorkflowType,
	}
	result, err := engine.Start(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.WorkflowID != "task/task-a" {
		t.Fatalf("workflow ID=%q", result.WorkflowID)
	}

	request.TenantID = ""
	if _, err := engine.Start(context.Background(), request); !errors.Is(err, ErrInvalidStartRequest) {
		t.Fatalf("error=%v", err)
	}
}
