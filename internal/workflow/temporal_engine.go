package workflow

import (
	"context"
	"errors"
	"fmt"
	"strings"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
)

type TemporalEngine struct {
	backend   temporalBackend
	taskQueue string
}

type temporalBackend interface {
	Start(context.Context, temporalStartRequest) (temporalStartResult, error)
	Cancel(context.Context, string) error
}

type temporalStartRequest struct {
	WorkflowID   string
	WorkflowType string
	TaskQueue    string
	Input        StartRequest
}

type temporalStartResult struct {
	RunID          string
	AlreadyStarted bool
}

type sdkTemporalBackend struct {
	client client.Client
}

func NewTemporalEngine(temporalClient client.Client, taskQueue string) (*TemporalEngine, error) {
	if temporalClient == nil {
		return nil, fmt.Errorf("Temporal client is required")
	}
	return newTemporalEngineWithBackend(sdkTemporalBackend{client: temporalClient}, taskQueue)
}

func newTemporalEngineWithBackend(backend temporalBackend, taskQueue string) (*TemporalEngine, error) {
	if backend == nil {
		return nil, fmt.Errorf("Temporal backend is required")
	}
	taskQueue = strings.TrimSpace(taskQueue)
	if taskQueue == "" {
		return nil, fmt.Errorf("Temporal task queue is required")
	}
	return &TemporalEngine{backend: backend, taskQueue: taskQueue}, nil
}

func (e *TemporalEngine) Start(ctx context.Context, request StartRequest) (StartResult, error) {
	if err := request.Validate(); err != nil {
		return StartResult{}, err
	}
	workflowID, err := WorkflowID(request.TaskID)
	if err != nil {
		return StartResult{}, err
	}
	result, err := e.backend.Start(ctx, temporalStartRequest{
		WorkflowID:   workflowID,
		WorkflowType: request.WorkflowType,
		TaskQueue:    e.taskQueue,
		Input:        request,
	})
	if err != nil {
		return StartResult{}, fmt.Errorf("start Temporal workflow %s: %w", workflowID, err)
	}
	return StartResult{
		WorkflowID:     workflowID,
		RunID:          result.RunID,
		AlreadyStarted: result.AlreadyStarted,
	}, nil
}

func (e *TemporalEngine) Cancel(ctx context.Context, request CancelRequest) error {
	if err := request.Validate(); err != nil {
		return err
	}
	workflowID, err := WorkflowID(request.TaskID)
	if err != nil {
		return err
	}
	if err := e.backend.Cancel(ctx, workflowID); err != nil {
		return fmt.Errorf("cancel Temporal workflow %s: %w", workflowID, err)
	}
	return nil
}

func (b sdkTemporalBackend) Start(ctx context.Context, request temporalStartRequest) (temporalStartResult, error) {
	run, err := b.client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:                                      request.WorkflowID,
		TaskQueue:                               request.TaskQueue,
		WorkflowIDReusePolicy:                   enumspb.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		WorkflowIDConflictPolicy:                enumspb.WORKFLOW_ID_CONFLICT_POLICY_FAIL,
		WorkflowExecutionErrorWhenAlreadyStarted: true,
	}, request.WorkflowType, request.Input)
	if err != nil {
		var alreadyStarted *serviceerror.WorkflowExecutionAlreadyStarted
		if errors.As(err, &alreadyStarted) {
			return temporalStartResult{RunID: alreadyStarted.RunId, AlreadyStarted: true}, nil
		}
		return temporalStartResult{}, err
	}
	if run == nil {
		return temporalStartResult{}, fmt.Errorf("Temporal returned a nil WorkflowRun")
	}
	return temporalStartResult{RunID: run.GetRunID()}, nil
}

func (b sdkTemporalBackend) Cancel(ctx context.Context, workflowID string) error {
	err := b.client.CancelWorkflow(ctx, workflowID, "")
	if err == nil {
		return nil
	}
	var notFound *serviceerror.NotFound
	if errors.As(err, &notFound) {
		return nil
	}
	return err
}
