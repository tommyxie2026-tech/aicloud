package workflow

import (
	"context"
	"testing"
)

type fakeTemporalBackend struct {
	startRequest temporalStartRequest
	startResult  temporalStartResult
	startErr     error
	cancelID     string
	cancelErr    error
}

func (f *fakeTemporalBackend) Start(_ context.Context, request temporalStartRequest) (temporalStartResult, error) {
	f.startRequest = request
	return f.startResult, f.startErr
}

func (f *fakeTemporalBackend) Cancel(_ context.Context, workflowID string) error {
	f.cancelID = workflowID
	return f.cancelErr
}

func TestTemporalEngineStartsDeterministicTaskWorkflow(t *testing.T) {
	backend := &fakeTemporalBackend{startResult: temporalStartResult{RunID: "run-a"}}
	engine, err := newTemporalEngineWithBackend(backend, "aicloud-task-v1")
	if err != nil {
		t.Fatal(err)
	}
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
	if result.WorkflowID != "task/task-a" || result.RunID != "run-a" || result.AlreadyStarted {
		t.Fatalf("unexpected result: %+v", result)
	}
	if backend.startRequest.WorkflowID != "task/task-a" || backend.startRequest.TaskQueue != "aicloud-task-v1" {
		t.Fatalf("unexpected Temporal start: %+v", backend.startRequest)
	}
	if backend.startRequest.Input.TenantID != "tenant-a" || backend.startRequest.Input.TraceID != "trace-a" {
		t.Fatalf("trusted scope was not propagated: %+v", backend.startRequest.Input)
	}
}

func TestTemporalEngineNormalizesAlreadyStartedResult(t *testing.T) {
	backend := &fakeTemporalBackend{startResult: temporalStartResult{RunID: "run-existing", AlreadyStarted: true}}
	engine, err := newTemporalEngineWithBackend(backend, "aicloud-task-v1")
	if err != nil {
		t.Fatal(err)
	}
	result, err := engine.Start(context.Background(), StartRequest{
		TenantID:     "tenant-a",
		ProjectID:    "project-a",
		TaskID:       "task-a",
		TraceID:      "trace-a",
		WorkflowType: TaskExecutionWorkflowType,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.AlreadyStarted || result.RunID != "run-existing" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestTemporalEngineCancelsByDeterministicWorkflowID(t *testing.T) {
	backend := &fakeTemporalBackend{}
	engine, err := newTemporalEngineWithBackend(backend, "aicloud-task-v1")
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.Cancel(context.Background(), CancelRequest{
		TenantID:  "tenant-a",
		ProjectID: "project-a",
		TaskID:    "task-a",
		TraceID:   "trace-a",
		Reason:    "business cancellation committed",
	}); err != nil {
		t.Fatal(err)
	}
	if backend.cancelID != "task/task-a" {
		t.Fatalf("cancel workflow ID=%q", backend.cancelID)
	}
}
