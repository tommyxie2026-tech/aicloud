package controlplane

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/identity"
	"github.com/tommyxie2026-tech/aicloud/internal/modelruntime"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
	tracepkg "github.com/tommyxie2026-tech/aicloud/internal/trace"
	"github.com/tommyxie2026-tech/aicloud/model/provider"
)

type ModelCommandResult struct {
	Result   modelruntime.Result
	Replayed bool
}

type storedModelCommandResponse struct {
	Result modelruntime.Result      `json:"result"`
	Error  *storedModelCommandError `json:"error,omitempty"`
}

type storedModelCommandError struct {
	Code      provider.ProviderErrorCode `json:"code,omitempty"`
	Message   string                     `json:"message"`
	Retryable bool                       `json:"retryable"`
}

// ExecuteModelIdempotent is the R6 public logical model-operation boundary.
// The provider transport call deliberately executes outside PostgreSQL. The
// first transaction reserves command identity and commits ROUTING->EXECUTING;
// a second transaction commits the final Task lifecycle and command result.
// Ambiguous crashes therefore fail closed as in_progress rather than blindly
// repeating a provider call whose physical side effect cannot be proven absent.
func (s *Service) ExecuteModelIdempotent(ctx context.Context, taskID string, request provider.ProviderRequest, metadata CommandMetadata) (ModelCommandResult, error) {
	if err := metadata.Validate(); err != nil {
		return ModelCommandResult{}, err
	}
	if s.modelRuntime == nil {
		return ModelCommandResult{}, fmt.Errorf("model runtime is not configured")
	}
	commands := s.taskCommandStore()
	if commands == nil {
		result, err := s.ExecuteModel(ctx, taskID, request)
		return ModelCommandResult{Result: result}, err
	}
	principal, err := identity.RequireProject(ctx)
	if err != nil {
		return ModelCommandResult{}, err
	}
	operation := modelCommandOperation(taskID)
	if replay, found, err := resolveModelReplay(ctx, commands, principal, operation, metadata); err != nil {
		return ModelCommandResult{}, err
	} else if found {
		return replay.result, replay.err
	}

	task, err := s.tasks.Get(ctx, taskID)
	if err != nil {
		return ModelCommandResult{}, err
	}
	if task.RouteDecisionID == "" {
		return ModelCommandResult{}, fmt.Errorf("task does not have a route decision")
	}
	decision, err := s.routes.Get(ctx, task.RouteDecisionID)
	if err != nil {
		return ModelCommandResult{}, err
	}

	request.RequestID = logicalModelOperationID(task.ID, metadata.IdempotencyKey)
	now := time.Now().UTC()
	begin := repository.ModelExecutionBeginCommit{
		Task: task,
		Idempotency: domain.IdempotencyRecord{
			TenantID:      principal.TenantID,
			ProjectID:     principal.ProjectID,
			Operation:     operation,
			Key:           strings.TrimSpace(metadata.IdempotencyKey),
			RequestDigest: strings.TrimSpace(metadata.RequestDigest),
			Status:        domain.IdempotencyInProgress,
			CreatedAt:     now,
			ExpiresAt:     now.Add(24 * time.Hour),
		},
	}
	var executionTransition *domain.TaskTransition
	if task.Status == domain.TaskRouting {
		transition, transitionErr := begin.Task.Transition(domain.TaskTransitionCommand{
			To: domain.TaskExecuting, Actor: transitionActor(ctx), Cause: "execute selected model deployment", At: now,
		})
		if transitionErr != nil {
			return ModelCommandResult{}, transitionErr
		}
		payload, payloadErr := json.Marshal(map[string]any{
			"operationId":     request.RequestID,
			"routeDecisionId": decision.ID,
			"modelId":         decision.Selected.ModelID,
			"modelVersion":    decision.Selected.ModelVersion,
		})
		if payloadErr != nil {
			return ModelCommandResult{}, fmt.Errorf("encode model execution start event: %w", payloadErr)
		}
		begin.Transition = &transition
		begin.Event = &domain.TaskEvent{
			EventID: tracepkg.NewID("task-event"), EventType: "TaskExecutionStarted",
			Actor: principalTaskEventActor(principal), Payload: payload,
			RequestID: strings.TrimSpace(metadata.RequestID), SchemaVersion: 1,
			OccurredAt: now, CreatedAt: now,
		}
		executionTransition = &transition
	} else if task.Status != domain.TaskExecuting {
		return ModelCommandResult{}, fmt.Errorf("%w: cannot execute model from %s", domain.ErrInvalidTaskTransition, task.Status)
	}

	begun, err := commands.BeginModelExecution(ctx, begin)
	if err != nil {
		return ModelCommandResult{}, err
	}
	if begun.Replayed {
		replayed, decodeErr := decodeStoredModelResponse(begun.Idempotency)
		if decodeErr != nil {
			return ModelCommandResult{}, decodeErr
		}
		return ModelCommandResult{Result: replayed.result.Result, Replayed: true}, replayed.err
	}
	task = begun.Task
	if executionTransition != nil {
		s.appendTaskTransitionTrace(ctx, task, *executionTransition, "true")
	}

	result, executeErr := s.modelRuntime.Execute(ctx, task.ID, task.TraceID, decision, request)
	if executeErr != nil && modelExecutionRetryable(result, executeErr) {
		stored, encodeErr := encodeStoredModelResponse(result, executeErr)
		if encodeErr != nil {
			return ModelCommandResult{Result: result}, encodeErr
		}
		retryable := begin.Idempotency
		retryable.Status = domain.IdempotencyFailedRetryable
		retryable.ResourceID = task.ID
		retryable.ResponseCode = 503
		retryable.ResponsePayload = stored
		if err := commands.MarkModelExecutionRetryable(ctx, retryable); err != nil {
			return ModelCommandResult{Result: result}, err
		}
		return ModelCommandResult{Result: result}, executeErr
	}

	if executeErr != nil {
		task.Result = executeErr.Error()
		failedAt := time.Now().UTC()
		failedTransition, transitionErr := task.Transition(domain.TaskTransitionCommand{
			To: domain.TaskFailed, Actor: transitionActor(ctx), Cause: "model execution failed", At: failedAt,
		})
		if transitionErr != nil {
			return ModelCommandResult{Result: result}, transitionErr
		}
		payload, payloadErr := json.Marshal(map[string]any{
			"operationId": request.RequestID,
			"error":       executeErr.Error(),
		})
		if payloadErr != nil {
			return ModelCommandResult{Result: result}, payloadErr
		}
		stored, encodeErr := encodeStoredModelResponse(result, executeErr)
		if encodeErr != nil {
			return ModelCommandResult{Result: result}, encodeErr
		}
		finalRecord := begin.Idempotency
		finalRecord.Status = domain.IdempotencyFailedFinal
		finalRecord.ResourceID = task.ID
		finalRecord.ResponseCode = 500
		finalRecord.ResponsePayload = stored
		finalized, finalizeErr := commands.FinalizeModelExecution(ctx, repository.ModelExecutionFinalizeCommit{
			Task: task, Transitions: []domain.TaskTransition{failedTransition},
			Events: []domain.TaskEvent{{
				EventID: tracepkg.NewID("task-event"), EventType: "TaskFailed",
				Actor: principalTaskEventActor(principal), Payload: payload,
				RequestID: strings.TrimSpace(metadata.RequestID), SchemaVersion: 1,
				OccurredAt: failedAt, CreatedAt: failedAt,
			}},
			Idempotency: finalRecord,
		})
		if finalizeErr != nil {
			return ModelCommandResult{Result: result}, finalizeErr
		}
		s.appendTaskTransitionTrace(ctx, finalized.Task, failedTransition, "true")
		return ModelCommandResult{Result: result}, executeErr
	}

	task.Result = encodeProviderResult(result.Response)
	if s.costLedger != nil {
		if total, currency, aggregateErr := s.costLedger.AggregateTask(ctx, task.ID); aggregateErr == nil {
			task.ActualCost = total
			task.Cost = total
			if currency != "" {
				task.Currency = currency
			}
		}
	}
	validatingAt := time.Now().UTC()
	validating, err := task.Transition(domain.TaskTransitionCommand{
		To: domain.TaskValidating, Actor: transitionActor(ctx), Cause: "validate model execution result", At: validatingAt,
	})
	if err != nil {
		return ModelCommandResult{Result: result}, err
	}
	completedAt := time.Now().UTC()
	completed, err := task.Transition(domain.TaskTransitionCommand{
		To: domain.TaskCompleted, Actor: transitionActor(ctx), Cause: "model execution result accepted", At: completedAt,
	})
	if err != nil {
		return ModelCommandResult{Result: result}, err
	}
	validationPayload, _ := json.Marshal(map[string]any{"operationId": request.RequestID})
	completedPayload, _ := json.Marshal(map[string]any{
		"operationId":  request.RequestID,
		"modelId":      result.Candidate.ModelID,
		"modelVersion": result.Candidate.ModelVersion,
		"fallback":     result.Fallback,
	})
	stored, err := encodeStoredModelResponse(result, nil)
	if err != nil {
		return ModelCommandResult{Result: result}, err
	}
	finalRecord := begin.Idempotency
	finalRecord.Status = domain.IdempotencyCompleted
	finalRecord.ResourceID = task.ID
	finalRecord.ResponseCode = 200
	finalRecord.ResponsePayload = stored
	finalized, err := commands.FinalizeModelExecution(ctx, repository.ModelExecutionFinalizeCommit{
		Task: task, Transitions: []domain.TaskTransition{validating, completed},
		Events: []domain.TaskEvent{
			{
				EventID: tracepkg.NewID("task-event"), EventType: "TaskValidationStarted",
				Actor: principalTaskEventActor(principal), Payload: validationPayload,
				RequestID: strings.TrimSpace(metadata.RequestID), SchemaVersion: 1,
				OccurredAt: validatingAt, CreatedAt: validatingAt,
			},
			{
				EventID: tracepkg.NewID("task-event"), EventType: "TaskCompleted",
				Actor: principalTaskEventActor(principal), Payload: completedPayload,
				RequestID: strings.TrimSpace(metadata.RequestID), SchemaVersion: 1,
				OccurredAt: completedAt, CreatedAt: completedAt,
			},
		},
		Idempotency: finalRecord,
	})
	if err != nil {
		return ModelCommandResult{Result: result}, err
	}
	s.appendTaskTransitionTrace(ctx, finalized.Task, validating, "true")
	s.appendTaskTransitionTrace(ctx, finalized.Task, completed, "true")
	return ModelCommandResult{Result: result}, nil
}

type resolvedModelReplay struct {
	result ModelCommandResult
	err    error
}

func resolveModelReplay(ctx context.Context, commands repository.TaskCommandStore, principal identity.Principal, operation string, metadata CommandMetadata) (resolvedModelReplay, bool, error) {
	record, found, err := commands.ResolveIdempotency(ctx, repository.IdempotencyLookup{
		TenantID: principal.TenantID, ProjectID: principal.ProjectID,
		Operation: operation, Key: strings.TrimSpace(metadata.IdempotencyKey),
		RequestDigest: strings.TrimSpace(metadata.RequestDigest),
	})
	if err != nil || !found {
		return resolvedModelReplay{}, false, err
	}
	decoded, err := decodeStoredModelResponse(record)
	if err != nil {
		return resolvedModelReplay{}, false, err
	}
	decoded.result.Replayed = true
	return decoded, true, nil
}

func decodeStoredModelResponse(record domain.IdempotencyRecord) (resolvedModelReplay, error) {
	if len(record.ResponsePayload) == 0 {
		return resolvedModelReplay{}, fmt.Errorf("model command replay does not contain a response")
	}
	var stored storedModelCommandResponse
	if err := json.Unmarshal(record.ResponsePayload, &stored); err != nil {
		return resolvedModelReplay{}, fmt.Errorf("decode model command replay: %w", err)
	}
	resolved := resolvedModelReplay{result: ModelCommandResult{Result: stored.Result, Replayed: true}}
	if stored.Error != nil {
		if stored.Error.Code != "" {
			resolved.err = &provider.ProviderError{
				Code: stored.Error.Code, Message: stored.Error.Message, Retryable: stored.Error.Retryable,
			}
		} else {
			resolved.err = errors.New(stored.Error.Message)
		}
	}
	return resolved, nil
}

func encodeStoredModelResponse(result modelruntime.Result, err error) (json.RawMessage, error) {
	stored := storedModelCommandResponse{Result: result}
	if err != nil {
		item := &storedModelCommandError{Message: err.Error()}
		var providerErr *provider.ProviderError
		if errors.As(err, &providerErr) {
			item.Code = providerErr.Code
			item.Message = providerErr.Message
			item.Retryable = providerErr.Retryable
		}
		stored.Error = item
	}
	body, marshalErr := json.Marshal(stored)
	if marshalErr != nil {
		return nil, fmt.Errorf("encode model command response: %w", marshalErr)
	}
	return body, nil
}

func modelExecutionRetryable(result modelruntime.Result, err error) bool {
	if err == nil {
		return false
	}
	if len(result.Attempts) > 0 && result.Attempts[len(result.Attempts)-1].Retryable {
		return true
	}
	var providerErr *provider.ProviderError
	if errors.As(err, &providerErr) {
		return providerErr.Retryable || providerErr.Code == provider.ErrProviderUnavailable ||
			providerErr.Code == provider.ErrProviderTimeout || providerErr.Code == provider.ErrRateLimited
	}
	return errors.Is(err, context.DeadlineExceeded)
}

func modelCommandOperation(taskID string) string {
	return "POST /api/v1/tasks/" + strings.TrimSpace(taskID) + "/model"
}

func logicalModelOperationID(taskID, idempotencyKey string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(taskID) + "|" + strings.TrimSpace(idempotencyKey)))
	return "model-op-" + hex.EncodeToString(sum[:12])
}

func (s *Service) appendTaskTransitionTrace(ctx context.Context, task domain.Task, transition domain.TaskTransition, idempotent string) {
	now := transition.At
	s.appendTrace(ctx, tracepkg.Event{
		ID: tracepkg.NewID("trace-event"), TraceID: task.TraceID, TaskID: task.ID,
		SpanID: tracepkg.NewID("span"), Name: "task.transition", Kind: "TASK",
		Status: tracepkg.StatusOK, Attributes: map[string]string{
			"task.from": string(transition.From), "task.to": string(transition.To),
			"task.actor": transition.Actor, "task.cause": transition.Cause,
			"task.version":       fmt.Sprintf("%d", task.Version),
			"command.idempotent": idempotent,
		}, StartedAt: now, EndedAt: timePointer(now),
	})
}
