package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/identity"
)

type TaskCreateCommit struct {
	Task        domain.Task
	Event       domain.TaskEvent
	Outbox      []domain.OutboxMessage
	Idempotency domain.IdempotencyRecord
}

// CreateTask atomically creates the Task projection, appends TaskCreated,
// records required Outbox delivery intent and completes the command
// idempotency record. Exact completed duplicates replay the existing Task.
func (r *ScopedPostgresTaskCommands) CreateTask(ctx context.Context, command TaskCreateCommit) (TaskCommandCommitResult, error) {
	if r == nil || r.db == nil {
		return TaskCommandCommitResult{}, fmt.Errorf("database is required")
	}
	if err := command.Idempotency.Validate(); err != nil {
		return TaskCommandCommitResult{}, err
	}
	if command.Idempotency.Status != domain.IdempotencyCompleted {
		return TaskCommandCommitResult{}, fmt.Errorf("successful task creation requires completed idempotency status")
	}

	principal, err := identity.RequireProject(ctx)
	if err != nil {
		return TaskCommandCommitResult{}, err
	}
	if principal.Type == identity.PrincipalSystem && !principal.HasCapability(identity.CapabilityTaskSystemAccess) {
		return TaskCommandCommitResult{}, fmt.Errorf("%w: %s", identity.ErrCapabilityRequired, identity.CapabilityTaskSystemAccess)
	}
	if command.Idempotency.TenantID != principal.TenantID || command.Idempotency.ProjectID != principal.ProjectID {
		return TaskCommandCommitResult{}, fmt.Errorf("idempotency scope must match authenticated principal")
	}

	if command.Task.TenantID != "" && command.Task.TenantID != principal.TenantID {
		return TaskCommandCommitResult{}, fmt.Errorf("task tenant identity does not match principal")
	}
	if command.Task.ProjectID != "" && command.Task.ProjectID != principal.ProjectID {
		return TaskCommandCommitResult{}, fmt.Errorf("task project identity does not match principal")
	}
	if command.Task.CreatedBy != "" && command.Task.CreatedBy != principal.SubjectID {
		return TaskCommandCommitResult{}, fmt.Errorf("task creator identity does not match principal")
	}
	command.Task.TenantID = principal.TenantID
	command.Task.ProjectID = principal.ProjectID
	command.Task.CreatedBy = principal.SubjectID
	if err := validateTaskCreation(command.Task); err != nil {
		return TaskCommandCommitResult{}, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return TaskCommandCommitResult{}, fmt.Errorf("begin task creation transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `SELECT
		set_config('aicloud.tenant_id', $1, true),
		set_config('aicloud.project_id', $2, true)`, principal.TenantID, principal.ProjectID); err != nil {
		return TaskCommandCommitResult{}, fmt.Errorf("set task creation transaction scope: %w", err)
	}

	replayed, existing, err := reserveIdempotency(ctx, tx, command.Idempotency)
	if err != nil {
		return TaskCommandCommitResult{}, err
	}
	if replayed {
		if existing.Status != domain.IdempotencyCompleted || existing.ResourceID == "" {
			return TaskCommandCommitResult{}, fmt.Errorf("completed task creation replay is missing resource identity")
		}
		task, err := scanScopedTask(tx.QueryRowContext(ctx, `SELECT `+scopedTaskColumns+` FROM tasks WHERE id=$1`, existing.ResourceID))
		if errors.Is(err, sql.ErrNoRows) {
			return TaskCommandCommitResult{}, fmt.Errorf("idempotency replay references missing task %q", existing.ResourceID)
		}
		if err != nil {
			return TaskCommandCommitResult{}, fmt.Errorf("load task for idempotency replay: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return TaskCommandCommitResult{}, fmt.Errorf("commit task creation replay transaction: %w", err)
		}
		return TaskCommandCommitResult{Task: task, Idempotency: existing, Replayed: true}, nil
	}

	_, err = tx.ExecContext(ctx, `INSERT INTO tasks (
		id, tenant_id, project_id, created_by, agent_id, input, status, version,
		result, cost, estimated_cost, actual_cost, currency, route_decision_id,
		trace_id, created_at, updated_at, completed_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`,
		command.Task.ID, command.Task.TenantID, command.Task.ProjectID, command.Task.CreatedBy,
		command.Task.AgentID, command.Task.Input, command.Task.Status, command.Task.Version,
		command.Task.Result, command.Task.Cost, command.Task.EstimatedCost, command.Task.ActualCost,
		command.Task.Currency, command.Task.RouteDecisionID, command.Task.TraceID,
		command.Task.CreatedAt, command.Task.UpdatedAt, command.Task.CompletedAt)
	if err != nil {
		return TaskCommandCommitResult{}, fmt.Errorf("create task projection in command transaction: %w", err)
	}

	now := time.Now().UTC()
	command.Event.TenantID = command.Task.TenantID
	command.Event.ProjectID = command.Task.ProjectID
	command.Event.TaskID = command.Task.ID
	command.Event.Sequence = 1
	command.Event.TraceID = command.Task.TraceID
	if command.Event.EventType == "" {
		command.Event.EventType = "TaskCreated"
	}
	if command.Event.EventType != "TaskCreated" {
		return TaskCommandCommitResult{}, fmt.Errorf("task creation must append TaskCreated")
	}
	if command.Event.OccurredAt.IsZero() {
		command.Event.OccurredAt = command.Task.CreatedAt
	}
	if command.Event.CreatedAt.IsZero() {
		command.Event.CreatedAt = now
	}
	if command.Event.SchemaVersion == 0 {
		command.Event.SchemaVersion = 1
	}
	if err := command.Event.Validate(); err != nil {
		return TaskCommandCommitResult{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO task_events(
		event_id, tenant_id, project_id, task_id, sequence, event_type,
		actor_principal_type, actor_subject_id, payload, request_id, trace_id,
		schema_version, occurred_at, created_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,NULLIF($10,''),$11,$12,$13,$14)`,
		command.Event.EventID, command.Event.TenantID, command.Event.ProjectID,
		command.Event.TaskID, command.Event.Sequence, command.Event.EventType,
		command.Event.Actor.PrincipalType, command.Event.Actor.SubjectID,
		string(command.Event.Payload), command.Event.RequestID, command.Event.TraceID,
		command.Event.SchemaVersion, command.Event.OccurredAt, command.Event.CreatedAt); err != nil {
		return TaskCommandCommitResult{}, fmt.Errorf("append TaskCreated event: %w", err)
	}

	for i := range command.Outbox {
		message := &command.Outbox[i]
		prepareTaskOutboxMessage(message, command.Task, now)
		if message.TenantID != command.Task.TenantID || message.ProjectID != command.Task.ProjectID {
			return TaskCommandCommitResult{}, fmt.Errorf("outbox scope must match task scope")
		}
		if message.TaskID != "" && message.TaskID != command.Task.ID {
			return TaskCommandCommitResult{}, fmt.Errorf("outbox task id must match command task")
		}
		if err := message.Validate(); err != nil {
			return TaskCommandCommitResult{}, err
		}
		if err := insertOutboxMessage(ctx, tx, *message); err != nil {
			return TaskCommandCommitResult{}, err
		}
	}

	command.Idempotency.ResourceID = command.Task.ID
	if err := completeIdempotency(ctx, tx, command.Idempotency); err != nil {
		return TaskCommandCommitResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return TaskCommandCommitResult{}, fmt.Errorf("commit task creation transaction: %w", err)
	}
	return TaskCommandCommitResult{
		Task: command.Task, Event: command.Event, Outbox: command.Outbox,
		Idempotency: command.Idempotency,
	}, nil
}

func validateTaskCreation(task domain.Task) error {
	if strings.TrimSpace(task.ID) == "" || strings.TrimSpace(task.Input) == "" || strings.TrimSpace(task.TraceID) == "" {
		return fmt.Errorf("task id, input and trace id are required")
	}
	if task.TenantID == "" || task.ProjectID == "" || task.CreatedBy == "" {
		return fmt.Errorf("task tenant, project and creator identity are required")
	}
	if task.Status != domain.TaskCreated || task.Version != 1 {
		return fmt.Errorf("new task must start at CREATED with version 1")
	}
	if task.CreatedAt.IsZero() || task.UpdatedAt.IsZero() {
		return fmt.Errorf("task creation timestamps are required")
	}
	if task.CompletedAt != nil {
		return fmt.Errorf("new task cannot have completed_at")
	}
	return nil
}

func prepareTaskOutboxMessage(message *domain.OutboxMessage, task domain.Task, now time.Time) {
	if message.TenantID == "" {
		message.TenantID = task.TenantID
	}
	if message.ProjectID == "" {
		message.ProjectID = task.ProjectID
	}
	if message.TaskID == "" && message.AggregateType == "Task" && message.AggregateID == task.ID {
		message.TaskID = task.ID
	}
	if message.Status == "" {
		message.Status = domain.OutboxPending
	}
	if message.CreatedAt.IsZero() {
		message.CreatedAt = now
	}
	if message.AvailableAt.IsZero() {
		message.AvailableAt = now
	}
}

func insertOutboxMessage(ctx context.Context, tx *sql.Tx, message domain.OutboxMessage) error {
	if _, err := tx.ExecContext(ctx, `INSERT INTO outbox_messages(
		outbox_id, tenant_id, project_id, task_id, aggregate_type, aggregate_id,
		event_type, payload, destination, idempotency_key, status, attempts,
		available_at, created_at, delivered_at
	) VALUES ($1,$2,$3,NULLIF($4,''),$5,$6,$7,$8::jsonb,$9,$10,$11,$12,$13,$14,$15)`,
		message.OutboxID, message.TenantID, message.ProjectID, message.TaskID,
		message.AggregateType, message.AggregateID, message.EventType, string(message.Payload),
		message.Destination, message.IdempotencyKey, message.Status, message.Attempts,
		message.AvailableAt, message.CreatedAt, message.DeliveredAt); err != nil {
		return fmt.Errorf("append outbox message: %w", err)
	}
	return nil
}
