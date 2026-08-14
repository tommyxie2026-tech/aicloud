package controlplane

import (
	"encoding/json"
	"testing"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

func TestWorkflowStartDeliveryKeyIsTenantTaskScoped(t *testing.T) {
	key := workflowStartDeliveryKey(" tenant-a ", " task-a ")
	if key != "workflow-start:tenant-a:task-a" {
		t.Fatalf("delivery key=%q", key)
	}
	if key == "task:create:client-key" {
		t.Fatal("workflow delivery identity must not reuse the public API idempotency key")
	}
}

func TestTaskCreatedPayloadCarriesDurableWorkflowCorrelation(t *testing.T) {
	payload, err := taskCreatedPayload(domain.Task{
		ID:      "task-a",
		TraceID: "trace-a",
		AgentID: "agent-a",
		Status:  domain.TaskCreated,
	})
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		TaskID  string            `json:"taskId"`
		TraceID string            `json:"traceId"`
		AgentID string            `json:"agentId"`
		Status  domain.TaskStatus `json:"status"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.TaskID != "task-a" || decoded.TraceID != "trace-a" || decoded.AgentID != "agent-a" || decoded.Status != domain.TaskCreated {
		t.Fatalf("unexpected TaskCreated payload: %+v", decoded)
	}
}
