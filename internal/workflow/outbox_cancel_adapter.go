package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

const DestinationWorkflowCancel = "workflow.cancel"

type CancelDeliveryAdapter struct {
	engine DurableEngine
}

func NewCancelDeliveryAdapter(engine DurableEngine) *CancelDeliveryAdapter {
	return &CancelDeliveryAdapter{engine: engine}
}

func (a *CancelDeliveryAdapter) Deliver(ctx context.Context, message domain.OutboxMessage) error {
	if a == nil || a.engine == nil {
		return fmt.Errorf("workflow cancel engine is not configured")
	}
	if message.Destination != DestinationWorkflowCancel {
		return fmt.Errorf("unsupported workflow cancellation destination %q", message.Destination)
	}
	if strings.TrimSpace(message.TenantID) == "" || strings.TrimSpace(message.ProjectID) == "" || strings.TrimSpace(message.TaskID) == "" {
		return fmt.Errorf("workflow cancellation Outbox message requires tenant, project and Task identity")
	}
	if message.AggregateType != "Task" || message.AggregateID != message.TaskID {
		return fmt.Errorf("workflow cancellation Outbox aggregate does not match Task identity")
	}

	var payload struct {
		TaskID  string `json:"taskId"`
		TraceID string `json:"traceId"`
		Reason  string `json:"reason,omitempty"`
	}
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		return fmt.Errorf("decode workflow cancellation payload: %w", err)
	}
	if payload.TaskID != message.TaskID {
		return fmt.Errorf("workflow cancellation payload Task identity mismatch")
	}

	if err := a.engine.Cancel(ctx, CancelRequest{
		TenantID:  message.TenantID,
		ProjectID: message.ProjectID,
		TaskID:    message.TaskID,
		TraceID:   payload.TraceID,
		Reason:    payload.Reason,
	}); err != nil {
		return fmt.Errorf("cancel durable Task workflow: %w", err)
	}
	return nil
}
