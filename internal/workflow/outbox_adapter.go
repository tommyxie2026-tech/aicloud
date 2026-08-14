package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

const DestinationWorkflowStart = "workflow.start"

type StartDeliveryAdapter struct {
	engine DurableEngine
}

func NewStartDeliveryAdapter(engine DurableEngine) *StartDeliveryAdapter {
	return &StartDeliveryAdapter{engine: engine}
}

func (a *StartDeliveryAdapter) Deliver(ctx context.Context, message domain.OutboxMessage) error {
	if a == nil || a.engine == nil {
		return fmt.Errorf("workflow start engine is not configured")
	}
	if message.Destination != DestinationWorkflowStart {
		return fmt.Errorf("unsupported workflow destination %q", message.Destination)
	}
	if strings.TrimSpace(message.TenantID) == "" || strings.TrimSpace(message.ProjectID) == "" || strings.TrimSpace(message.TaskID) == "" {
		return fmt.Errorf("workflow start Outbox message requires tenant, project and task identity")
	}
	if message.AggregateType != "Task" || message.AggregateID != message.TaskID {
		return fmt.Errorf("workflow start Outbox aggregate does not match Task identity")
	}

	var payload struct {
		TaskID  string `json:"taskId"`
		TraceID string `json:"traceId"`
	}
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		return fmt.Errorf("decode workflow start payload: %w", err)
	}
	if payload.TaskID != message.TaskID {
		return fmt.Errorf("workflow start payload Task identity mismatch")
	}

	_, err := a.engine.Start(ctx, StartRequest{
		TenantID:     message.TenantID,
		ProjectID:    message.ProjectID,
		TaskID:       message.TaskID,
		TraceID:      payload.TraceID,
		WorkflowType: TaskExecutionWorkflowType,
	})
	if err != nil {
		return fmt.Errorf("start durable Task workflow: %w", err)
	}
	return nil
}
