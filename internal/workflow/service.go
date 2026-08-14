package workflow

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const TaskExecutionWorkflowType = "task-execution-v1"

var (
	ErrInvalidStartRequest  = errors.New("invalid workflow start request")
	ErrInvalidCancelRequest = errors.New("invalid workflow cancel request")
)

type StartRequest struct {
	TenantID     string `json:"tenantId"`
	ProjectID    string `json:"projectId"`
	TaskID       string `json:"taskId"`
	TraceID      string `json:"traceId"`
	WorkflowType string `json:"workflowType"`
}

func (r StartRequest) Validate() error {
	if strings.TrimSpace(r.TenantID) == "" {
		return fmt.Errorf("%w: tenant ID is required", ErrInvalidStartRequest)
	}
	if strings.TrimSpace(r.ProjectID) == "" {
		return fmt.Errorf("%w: project ID is required", ErrInvalidStartRequest)
	}
	if strings.TrimSpace(r.TaskID) == "" {
		return fmt.Errorf("%w: task ID is required", ErrInvalidStartRequest)
	}
	if strings.TrimSpace(r.TraceID) == "" {
		return fmt.Errorf("%w: trace ID is required", ErrInvalidStartRequest)
	}
	workflowType := strings.TrimSpace(r.WorkflowType)
	if workflowType == "" {
		return fmt.Errorf("%w: workflow type is required", ErrInvalidStartRequest)
	}
	if workflowType != TaskExecutionWorkflowType {
		return fmt.Errorf("%w: workflow type %q is unsupported", ErrInvalidStartRequest, workflowType)
	}
	return nil
}

type StartResult struct {
	WorkflowID     string `json:"workflowId"`
	RunID          string `json:"runId,omitempty"`
	AlreadyStarted bool   `json:"alreadyStarted"`
}

type CancelRequest struct {
	TenantID  string `json:"tenantId"`
	ProjectID string `json:"projectId"`
	TaskID    string `json:"taskId"`
	TraceID   string `json:"traceId"`
	Reason    string `json:"reason,omitempty"`
}

func (r CancelRequest) Validate() error {
	if strings.TrimSpace(r.TenantID) == "" {
		return fmt.Errorf("%w: tenant ID is required", ErrInvalidCancelRequest)
	}
	if strings.TrimSpace(r.ProjectID) == "" {
		return fmt.Errorf("%w: project ID is required", ErrInvalidCancelRequest)
	}
	if strings.TrimSpace(r.TaskID) == "" {
		return fmt.Errorf("%w: task ID is required", ErrInvalidCancelRequest)
	}
	if strings.TrimSpace(r.TraceID) == "" {
		return fmt.Errorf("%w: trace ID is required", ErrInvalidCancelRequest)
	}
	return nil
}

func WorkflowID(taskID string) (string, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return "", fmt.Errorf("%w: task ID is required", ErrInvalidStartRequest)
	}
	return "task/" + taskID, nil
}

// Engine is the pre-S3 compatibility seam used only by the legacy non-durable
// CreateTask path. New durable execution code must use DurableEngine.
//
// Deprecated: remove after S3C moves all Task starts behind the Outbox path.
type Engine interface {
	Start(context.Context, string) error
}

// DurableEngine is the S3 execution boundary. It is intentionally Temporal
// neutral and carries trusted business identity explicitly.
type DurableEngine interface {
	Start(context.Context, StartRequest) (StartResult, error)
	Cancel(context.Context, CancelRequest) error
}

type NoopEngine struct{}

func (NoopEngine) Start(context.Context, string) error { return nil }

type NoopDurableEngine struct{}

func (NoopDurableEngine) Start(_ context.Context, request StartRequest) (StartResult, error) {
	if err := request.Validate(); err != nil {
		return StartResult{}, err
	}
	workflowID, err := WorkflowID(request.TaskID)
	if err != nil {
		return StartResult{}, err
	}
	return StartResult{WorkflowID: workflowID}, nil
}

func (NoopDurableEngine) Cancel(_ context.Context, request CancelRequest) error {
	return request.Validate()
}
