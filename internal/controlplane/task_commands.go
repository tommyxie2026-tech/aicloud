package controlplane

import (
	"context"
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

var ErrCommandMetadataRequired = errors.New("command idempotency metadata is required")

// CommandMetadata is transport-independent metadata for a public business
// command. RequestDigest must describe canonical business intent rather than
// raw transport bytes.
type CommandMetadata struct {
	IdempotencyKey string
	RequestDigest  string
	RequestID      string
}

func (m CommandMetadata) Validate() error {
	if strings.TrimSpace(m.IdempotencyKey) == "" || strings.TrimSpace(m.RequestDigest) == "" {
		return ErrCommandMetadataRequired
	}
	return nil
}

type TaskCommandResult struct {
	Task     domain.Task
	Replayed bool
}

// CreateTaskIdempotent is the R6 public Task creation path. When the production
// PostgreSQL Task repository exposes a TaskCommandStore, Task + TaskCreated +
// workflow Outbox intent + Idempotency result are committed atomically. The
// legacy path is retained only for development repositories that do not expose
// the durable command kernel.
func (s *Service) CreateTaskIdempotent(ctx context.Context, input, agentID string, metadata CommandMetadata) (TaskCommandResult, error) {
	if err := metadata.Validate(); err != nil {
		return TaskCommandResult{}, err
	}
	commands := s.taskCommandStore()
	if commands == nil {
		task, err := s.CreateTask(ctx, input, agentID)
		return TaskCommandResult{Task: task}, err
	}

	principal, err := identity.RequireProject(ctx)
	if err != nil {
		return TaskCommandResult{}, err
	}
	now := time.Now().UTC()
	task, err := domain.NewTask(domain.NewTaskParams{
		ID:       fmt.Sprintf("task-%d", now.UnixNano()),
		AgentID:  agentID,
		Input:    input,
		TraceID:  fmt.Sprintf("trace-%d", now.UnixNano()),
		Currency: "USD",
		Now:      now,
	})
	if err != nil {
		return TaskCommandResult{}, err
	}

	payload, err := taskCreatedPayload(task)
	if err != nil {
		return TaskCommandResult{}, err
	}

	commit := repository.TaskCreateCommit{
		Task: task,
		Event: domain.TaskEvent{
			EventID:   tracepkg.NewID("task-event"),
			EventType: "TaskCreated",
			Actor: domain.TaskEventActor{
				PrincipalType: string(principal.Type),
				SubjectID:     principal.SubjectID,
			},
			Payload:       payload,
			RequestID:     strings.TrimSpace(metadata.RequestID),
			SchemaVersion: 1,
			OccurredAt:    now,
			CreatedAt:     now,
		},
		Outbox: []domain.OutboxMessage{{
			OutboxID:       tracepkg.NewID("outbox"),
			AggregateType:  "Task",
			AggregateID:    task.ID,
			EventType:      "TaskCreated",
			Payload:        payload,
			Destination:    "workflow.start",
			IdempotencyKey: workflowStartDeliveryKey(principal.TenantID, task.ID),
			Status:         domain.OutboxPending,
			AvailableAt:    now,
			CreatedAt:      now,
		}},
		Idempotency: domain.IdempotencyRecord{
			TenantID:      principal.TenantID,
			ProjectID:     principal.ProjectID,
			Operation:     "POST /api/v1/tasks",
			Key:           strings.TrimSpace(metadata.IdempotencyKey),
			RequestDigest: strings.TrimSpace(metadata.RequestDigest),
			Status:        domain.IdempotencyCompleted,
			ResponseCode:  202,
			CreatedAt:     now,
			ExpiresAt:     now.Add(24 * time.Hour),
		},
	}

	result, err := commands.CreateTask(ctx, commit)
	if err != nil {
		return TaskCommandResult{}, err
	}
	if !result.Replayed {
		s.appendTrace(ctx, tracepkg.Event{
			ID: tracepkg.NewID("trace-event"), TraceID: result.Task.TraceID, TaskID: result.Task.ID,
			SpanID: tracepkg.NewID("span"), Name: "task.created", Kind: "TASK",
			Status: tracepkg.StatusOK, Attributes: map[string]string{
				"agent.id": agentID, "task.status": string(result.Task.Status),
				"task.version": "1", "command.idempotent": "true",
			}, StartedAt: now, EndedAt: timePointer(now),
		})
	}
	return TaskCommandResult{Task: result.Task, Replayed: result.Replayed}, nil
}

func taskCreatedPayload(task domain.Task) (json.RawMessage, error) {
	payload, err := json.Marshal(map[string]any{
		"taskId":  task.ID,
		"traceId": task.TraceID,
		"status":  task.Status,
		"agentId": task.AgentID,
	})
	if err != nil {
		return nil, fmt.Errorf("encode TaskCreated payload: %w", err)
	}
	return payload, nil
}

func workflowStartDeliveryKey(tenantID, taskID string) string {
	return "workflow-start:" + strings.TrimSpace(tenantID) + ":" + strings.TrimSpace(taskID)
}

func (s *Service) taskCommandStore() repository.TaskCommandStore {
	if s == nil || s.tasks == nil {
		return nil
	}
	provider, ok := s.tasks.(repository.TaskCommandStoreProvider)
	if !ok {
		return nil
	}
	return provider.TaskCommands()
}
