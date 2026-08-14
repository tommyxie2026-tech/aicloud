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
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
	tracepkg "github.com/tommyxie2026-tech/aicloud/internal/trace"
)

const taskCancelOperationV1 = "controlplane.task.cancel.v1"

type CancelTaskResult struct {
	Task     domain.Task
	Replayed bool
}

// CancelTaskIdempotent is the internal durable cancellation command. It does
// not expose an HTTP contract. The business cancellation is committed before
// any workflow cancellation can be delivered.
func (s *Service) CancelTaskIdempotent(
	ctx context.Context,
	taskID string,
	expectedVersion int64,
	reason string,
	metadata CommandMetadata,
) (CancelTaskResult, error) {
	if err := metadata.Validate(); err != nil {
		return CancelTaskResult{}, err
	}
	if strings.TrimSpace(taskID) == "" {
		return CancelTaskResult{}, fmt.Errorf("Task ID is required")
	}
	if expectedVersion <= 0 {
		return CancelTaskResult{}, fmt.Errorf("expected Task version must be greater than zero")
	}
	commands := s.taskCommandStore()
	if commands == nil {
		return CancelTaskResult{}, fmt.Errorf("durable Task command store is required for cancellation")
	}
	principal, err := identity.RequireProject(ctx)
	if err != nil {
		return CancelTaskResult{}, err
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "Task cancellation requested"
	}

	requestDigest, err := taskCancelDigest(principal, taskID, expectedVersion, reason)
	if err != nil {
		return CancelTaskResult{}, err
	}
	existing, found, err := commands.ResolveIdempotency(ctx, repository.IdempotencyLookup{
		TenantID:      principal.TenantID,
		ProjectID:     principal.ProjectID,
		Operation:     taskCancelOperationV1,
		Key:           strings.TrimSpace(metadata.IdempotencyKey),
		RequestDigest: requestDigest,
	})
	if err != nil {
		return CancelTaskResult{}, err
	}
	if found {
		task, decodeErr := decodeCancelledTask(existing.ResponsePayload, principal, taskID)
		if decodeErr != nil {
			return CancelTaskResult{}, decodeErr
		}
		return CancelTaskResult{Task: task, Replayed: true}, nil
	}

	current, err := s.tasks.Get(ctx, taskID)
	if err != nil {
		return CancelTaskResult{}, err
	}
	if current.TenantID != principal.TenantID || current.ProjectID != principal.ProjectID {
		return CancelTaskResult{}, repository.ErrNotFound
	}
	if current.IsTerminal() {
		if current.Status == domain.TaskCancelled {
			return CancelTaskResult{Task: current}, nil
		}
		return CancelTaskResult{}, domain.ErrTaskTerminal
	}
	if current.Version != expectedVersion {
		return CancelTaskResult{}, repository.ErrVersionConflict
	}

	now := time.Now().UTC()
	next := current
	transition, err := next.Transition(domain.TaskTransitionCommand{
		To:    domain.TaskCancelled,
		Actor: principal.SubjectID,
		Cause: reason,
		At:    now,
	})
	if err != nil {
		return CancelTaskResult{}, err
	}

	eventPayload, err := json.Marshal(map[string]any{
		"fromStatus": transition.From,
		"toStatus":   transition.To,
		"cause":      transition.Cause,
	})
	if err != nil {
		return CancelTaskResult{}, fmt.Errorf("encode Task cancellation event: %w", err)
	}
	cancelPayload, err := json.Marshal(map[string]string{
		"taskId":  current.ID,
		"traceId": current.TraceID,
		"reason":  reason,
	})
	if err != nil {
		return CancelTaskResult{}, fmt.Errorf("encode workflow cancellation intent: %w", err)
	}

	responseTask := next
	responseTask.Version = expectedVersion + 1
	responsePayload, err := json.Marshal(responseTask)
	if err != nil {
		return CancelTaskResult{}, fmt.Errorf("encode Task cancellation response: %w", err)
	}

	commit := repository.TaskCommandCommit{
		Task:            next,
		ExpectedVersion: expectedVersion,
		Event: domain.TaskEvent{
			EventID:   tracepkg.NewID("task-event"),
			EventType: "TaskCancelled",
			Actor: domain.TaskEventActor{
				PrincipalType: string(principal.Type),
				SubjectID:     principal.SubjectID,
			},
			Payload:       eventPayload,
			RequestID:     strings.TrimSpace(metadata.RequestID),
			SchemaVersion: 1,
			OccurredAt:    now,
			CreatedAt:     now,
		},
		Outbox: []domain.OutboxMessage{{
			OutboxID:       tracepkg.NewID("outbox"),
			AggregateType:  "Task",
			AggregateID:    current.ID,
			EventType:      "TaskCancelled",
			Payload:        cancelPayload,
			Destination:    "workflow.cancel",
			IdempotencyKey: workflowCancelDeliveryKey(principal.TenantID, current.ID),
			Status:         domain.OutboxPending,
			AvailableAt:    now,
			CreatedAt:      now,
		}},
		Idempotency: domain.IdempotencyRecord{
			TenantID:        principal.TenantID,
			ProjectID:       principal.ProjectID,
			Operation:       taskCancelOperationV1,
			Key:             strings.TrimSpace(metadata.IdempotencyKey),
			RequestDigest:   requestDigest,
			Status:          domain.IdempotencyCompleted,
			ResourceID:      current.ID,
			ResponseCode:    200,
			ResponseDigest:  taskCancelSHA256(responsePayload),
			ResponsePayload: responsePayload,
			CreatedAt:       now,
			ExpiresAt:       now.Add(30 * 24 * time.Hour),
		},
	}

	result, err := commands.CommitTransition(ctx, commit)
	if err != nil {
		if errors.Is(err, repository.ErrVersionConflict) || errors.Is(err, repository.ErrIdempotencyConflict) {
			return CancelTaskResult{}, err
		}
		return CancelTaskResult{}, err
	}
	if result.Replayed {
		task, decodeErr := decodeCancelledTask(result.Idempotency.ResponsePayload, principal, taskID)
		if decodeErr != nil {
			return CancelTaskResult{}, decodeErr
		}
		return CancelTaskResult{Task: task, Replayed: true}, nil
	}
	return CancelTaskResult{Task: result.Task}, nil
}

func workflowCancelDeliveryKey(tenantID, taskID string) string {
	return "workflow-cancel:" + strings.TrimSpace(tenantID) + ":" + strings.TrimSpace(taskID)
}

func taskCancelDigest(principal identity.Principal, taskID string, expectedVersion int64, reason string) (string, error) {
	payload, err := json.Marshal(struct {
		TenantID        string `json:"tenantId"`
		ProjectID       string `json:"projectId"`
		TaskID          string `json:"taskId"`
		ExpectedVersion int64  `json:"expectedVersion"`
		Reason          string `json:"reason"`
	}{
		TenantID: principal.TenantID, ProjectID: principal.ProjectID, TaskID: taskID,
		ExpectedVersion: expectedVersion, Reason: reason,
	})
	if err != nil {
		return "", err
	}
	return taskCancelSHA256(payload), nil
}

func taskCancelSHA256(payload []byte) string {
	sum := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func decodeCancelledTask(payload json.RawMessage, principal identity.Principal, taskID string) (domain.Task, error) {
	if len(payload) == 0 {
		return domain.Task{}, fmt.Errorf("Task cancellation idempotency replay is missing response payload")
	}
	var task domain.Task
	if err := json.Unmarshal(payload, &task); err != nil {
		return domain.Task{}, fmt.Errorf("decode Task cancellation replay: %w", err)
	}
	if task.ID != taskID || task.TenantID != principal.TenantID || task.ProjectID != principal.ProjectID || task.Status != domain.TaskCancelled {
		return domain.Task{}, fmt.Errorf("Task cancellation replay correlation is invalid")
	}
	return task, nil
}
