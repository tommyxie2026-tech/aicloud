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

type ModelExecutionBeginCommit struct {
	Task        domain.Task
	Transition  *domain.TaskTransition
	Event       *domain.TaskEvent
	Idempotency domain.IdempotencyRecord
}

type ModelExecutionBeginResult struct {
	Task        domain.Task
	Event       *domain.TaskEvent
	Idempotency domain.IdempotencyRecord
	Replayed    bool
}

type ModelExecutionFinalizeCommit struct {
	Task        domain.Task
	Transitions []domain.TaskTransition
	Events      []domain.TaskEvent
	Idempotency domain.IdempotencyRecord
}

type ModelExecutionFinalizeResult struct {
	Task        domain.Task
	Events      []domain.TaskEvent
	Idempotency domain.IdempotencyRecord
}

// BeginModelExecution reserves the public command idempotency key and, on the
// first physical attempt, atomically moves ROUTING -> EXECUTING with the
// canonical TaskExecutionStarted event. The public idempotency record remains
// in_progress while the provider call runs outside the SQL transaction.
//
// A retry after a retryable provider failure reuses the same logical command
// while the Task is already EXECUTING and therefore does not append a duplicate
// TaskExecutionStarted event.
func (r *ScopedPostgresTaskCommands) BeginModelExecution(ctx context.Context, command ModelExecutionBeginCommit) (ModelExecutionBeginResult, error) {
	if r == nil || r.db == nil {
		return ModelExecutionBeginResult{}, fmt.Errorf("database is required")
	}
	if err := command.Idempotency.Validate(); err != nil {
		return ModelExecutionBeginResult{}, err
	}
	if command.Idempotency.Status != domain.IdempotencyInProgress {
		return ModelExecutionBeginResult{}, fmt.Errorf("model execution begin requires in_progress idempotency status")
	}
	principal, err := identity.RequireProject(ctx)
	if err != nil {
		return ModelExecutionBeginResult{}, err
	}
	if principal.Type == identity.PrincipalSystem && !principal.HasCapability(identity.CapabilityTaskSystemAccess) {
		return ModelExecutionBeginResult{}, fmt.Errorf("%w: %s", identity.ErrCapabilityRequired, identity.CapabilityTaskSystemAccess)
	}
	if command.Idempotency.TenantID != principal.TenantID || command.Idempotency.ProjectID != principal.ProjectID {
		return ModelExecutionBeginResult{}, fmt.Errorf("idempotency scope must match authenticated principal")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ModelExecutionBeginResult{}, fmt.Errorf("begin model execution command transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `SELECT
		set_config('aicloud.tenant_id', $1, true),
		set_config('aicloud.project_id', $2, true)`, principal.TenantID, principal.ProjectID); err != nil {
		return ModelExecutionBeginResult{}, fmt.Errorf("set model execution transaction scope: %w", err)
	}

	replayed, existing, err := reserveIdempotency(ctx, tx, command.Idempotency)
	if err != nil {
		return ModelExecutionBeginResult{}, err
	}
	if replayed {
		if err := tx.Commit(); err != nil {
			return ModelExecutionBeginResult{}, fmt.Errorf("commit model execution replay transaction: %w", err)
		}
		return ModelExecutionBeginResult{Idempotency: existing, Replayed: true}, nil
	}

	current, err := scanScopedTask(tx.QueryRowContext(ctx, `SELECT `+scopedTaskColumns+` FROM tasks WHERE id=$1 FOR UPDATE`, command.Task.ID))
	if errors.Is(err, sql.ErrNoRows) {
		return ModelExecutionBeginResult{}, ErrNotFound
	}
	if err != nil {
		return ModelExecutionBeginResult{}, fmt.Errorf("lock task for model execution: %w", err)
	}
	if !principal.OwnsProject(current.TenantID, current.ProjectID) {
		return ModelExecutionBeginResult{}, ErrNotFound
	}
	if current.TenantID != command.Task.TenantID || current.ProjectID != command.Task.ProjectID || current.CreatedBy != command.Task.CreatedBy {
		return ModelExecutionBeginResult{}, fmt.Errorf("task tenant, project and creator identity are immutable")
	}
	if command.Task.Version != current.Version {
		return ModelExecutionBeginResult{}, ErrVersionConflict
	}

	var persistedEvent *domain.TaskEvent
	if current.Status == domain.TaskRouting {
		if command.Transition == nil || command.Event == nil {
			return ModelExecutionBeginResult{}, fmt.Errorf("first model execution attempt requires transition and event evidence")
		}
		if command.Transition.From != domain.TaskRouting || command.Transition.To != domain.TaskExecuting || command.Task.Status != domain.TaskExecuting {
			return ModelExecutionBeginResult{}, fmt.Errorf("model execution begin must transition ROUTING to EXECUTING")
		}
		if strings.TrimSpace(command.Transition.Actor) == "" || strings.TrimSpace(command.Transition.Cause) == "" || command.Transition.At.IsZero() {
			return ModelExecutionBeginResult{}, fmt.Errorf("model execution transition evidence is incomplete")
		}
		if command.Event.EventType != "TaskExecutionStarted" {
			return ModelExecutionBeginResult{}, fmt.Errorf("model execution begin requires TaskExecutionStarted")
		}

		var nextSequence int64
		if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence), 0) + 1 FROM task_events WHERE task_id=$1`, current.ID).Scan(&nextSequence); err != nil {
			return ModelExecutionBeginResult{}, fmt.Errorf("allocate model execution event sequence: %w", err)
		}
		var nextVersion int64
		err = tx.QueryRowContext(ctx, `UPDATE tasks SET agent_id=$2, input=$3,
			status=$4, result=$5, cost=$6, estimated_cost=$7, actual_cost=$8,
			currency=$9, route_decision_id=$10, trace_id=$11, updated_at=$12,
			completed_at=$13, version=version+1
			WHERE id=$1 AND version=$14
			RETURNING version`,
			command.Task.ID, command.Task.AgentID, command.Task.Input, command.Task.Status,
			command.Task.Result, command.Task.Cost, command.Task.EstimatedCost,
			command.Task.ActualCost, command.Task.Currency, command.Task.RouteDecisionID,
			command.Task.TraceID, command.Task.UpdatedAt, command.Task.CompletedAt,
			command.Task.Version).Scan(&nextVersion)
		if errors.Is(err, sql.ErrNoRows) {
			return ModelExecutionBeginResult{}, ErrVersionConflict
		}
		if err != nil {
			return ModelExecutionBeginResult{}, fmt.Errorf("update task to executing: %w", err)
		}
		command.Task.Version = nextVersion

		now := time.Now().UTC()
		event := *command.Event
		event.TenantID = current.TenantID
		event.ProjectID = current.ProjectID
		event.TaskID = current.ID
		event.Sequence = nextSequence
		event.TraceID = command.Task.TraceID
		if event.OccurredAt.IsZero() {
			event.OccurredAt = command.Transition.At
		}
		if event.CreatedAt.IsZero() {
			event.CreatedAt = now
		}
		if event.SchemaVersion == 0 {
			event.SchemaVersion = 1
		}
		if err := event.Validate(); err != nil {
			return ModelExecutionBeginResult{}, err
		}
		if err := insertTaskEvent(ctx, tx, event); err != nil {
			return ModelExecutionBeginResult{}, err
		}
		persistedEvent = &event
	} else if current.Status == domain.TaskExecuting {
		if command.Transition != nil || command.Event != nil {
			return ModelExecutionBeginResult{}, fmt.Errorf("model execution retry must not append a duplicate execution-start transition")
		}
		command.Task = current
	} else {
		return ModelExecutionBeginResult{}, fmt.Errorf("%w: cannot begin model execution from %s", domain.ErrInvalidTaskTransition, current.Status)
	}

	if err := tx.Commit(); err != nil {
		return ModelExecutionBeginResult{}, fmt.Errorf("commit model execution begin transaction: %w", err)
	}
	command.Idempotency.Status = domain.IdempotencyInProgress
	return ModelExecutionBeginResult{
		Task: command.Task, Event: persistedEvent, Idempotency: command.Idempotency,
	}, nil
}

// FinalizeModelExecution atomically records the final Task projection, all
// remaining canonical lifecycle events and the public command result. Success
// uses EXECUTING -> VALIDATING -> COMPLETED; final failure uses
// EXECUTING -> FAILED.
func (r *ScopedPostgresTaskCommands) FinalizeModelExecution(ctx context.Context, command ModelExecutionFinalizeCommit) (ModelExecutionFinalizeResult, error) {
	if r == nil || r.db == nil {
		return ModelExecutionFinalizeResult{}, fmt.Errorf("database is required")
	}
	if err := command.Idempotency.Validate(); err != nil {
		return ModelExecutionFinalizeResult{}, err
	}
	if command.Idempotency.Status != domain.IdempotencyCompleted && command.Idempotency.Status != domain.IdempotencyFailedFinal {
		return ModelExecutionFinalizeResult{}, fmt.Errorf("model execution finalization requires completed or failed_final idempotency status")
	}
	principal, err := identity.RequireProject(ctx)
	if err != nil {
		return ModelExecutionFinalizeResult{}, err
	}
	if principal.Type == identity.PrincipalSystem && !principal.HasCapability(identity.CapabilityTaskSystemAccess) {
		return ModelExecutionFinalizeResult{}, fmt.Errorf("%w: %s", identity.ErrCapabilityRequired, identity.CapabilityTaskSystemAccess)
	}
	if command.Idempotency.TenantID != principal.TenantID || command.Idempotency.ProjectID != principal.ProjectID {
		return ModelExecutionFinalizeResult{}, fmt.Errorf("idempotency scope must match authenticated principal")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ModelExecutionFinalizeResult{}, fmt.Errorf("begin model execution finalization transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `SELECT
		set_config('aicloud.tenant_id', $1, true),
		set_config('aicloud.project_id', $2, true)`, principal.TenantID, principal.ProjectID); err != nil {
		return ModelExecutionFinalizeResult{}, fmt.Errorf("set model finalization transaction scope: %w", err)
	}

	reserved, err := loadIdempotencyForUpdate(ctx, tx, command.Idempotency)
	if err != nil {
		return ModelExecutionFinalizeResult{}, err
	}
	if reserved.RequestDigest != command.Idempotency.RequestDigest {
		return ModelExecutionFinalizeResult{}, ErrIdempotencyConflict
	}
	if reserved.Status != domain.IdempotencyInProgress {
		return ModelExecutionFinalizeResult{}, fmt.Errorf("model execution command is not in progress")
	}

	current, err := scanScopedTask(tx.QueryRowContext(ctx, `SELECT `+scopedTaskColumns+` FROM tasks WHERE id=$1 FOR UPDATE`, command.Task.ID))
	if errors.Is(err, sql.ErrNoRows) {
		return ModelExecutionFinalizeResult{}, ErrNotFound
	}
	if err != nil {
		return ModelExecutionFinalizeResult{}, fmt.Errorf("lock task for model finalization: %w", err)
	}
	if !principal.OwnsProject(current.TenantID, current.ProjectID) {
		return ModelExecutionFinalizeResult{}, ErrNotFound
	}
	if current.Status != domain.TaskExecuting {
		return ModelExecutionFinalizeResult{}, fmt.Errorf("%w: model execution can only finalize from EXECUTING", domain.ErrInvalidTaskTransition)
	}
	if current.TenantID != command.Task.TenantID || current.ProjectID != command.Task.ProjectID || current.CreatedBy != command.Task.CreatedBy {
		return ModelExecutionFinalizeResult{}, fmt.Errorf("task tenant, project and creator identity are immutable")
	}
	if command.Task.Version != current.Version {
		return ModelExecutionFinalizeResult{}, ErrVersionConflict
	}
	if len(command.Transitions) != len(command.Events) || len(command.Transitions) == 0 {
		return ModelExecutionFinalizeResult{}, fmt.Errorf("model finalization requires matching transition and event evidence")
	}

	expectedFrom := current.Status
	for i, transition := range command.Transitions {
		if transition.From != expectedFrom {
			return ModelExecutionFinalizeResult{}, fmt.Errorf("model finalization transition chain is discontinuous")
		}
		if strings.TrimSpace(transition.Actor) == "" || strings.TrimSpace(transition.Cause) == "" || transition.At.IsZero() {
			return ModelExecutionFinalizeResult{}, fmt.Errorf("model finalization transition evidence is incomplete")
		}
		canonical, ok := domain.CanonicalTaskStateEvent(transition.To)
		if !ok || command.Events[i].EventType != canonical {
			return ModelExecutionFinalizeResult{}, fmt.Errorf("canonical model finalization event mismatch for %s", transition.To)
		}
		expectedFrom = transition.To
	}
	if expectedFrom != command.Task.Status {
		return ModelExecutionFinalizeResult{}, fmt.Errorf("model finalization transition chain does not match final task status")
	}
	if command.Task.Status == domain.TaskCompleted {
		if len(command.Transitions) != 2 || command.Transitions[0].To != domain.TaskValidating || command.Transitions[1].To != domain.TaskCompleted || command.Idempotency.Status != domain.IdempotencyCompleted {
			return ModelExecutionFinalizeResult{}, fmt.Errorf("successful model execution must finalize EXECUTING -> VALIDATING -> COMPLETED")
		}
	} else if command.Task.Status == domain.TaskFailed {
		if len(command.Transitions) != 1 || command.Transitions[0].To != domain.TaskFailed || command.Idempotency.Status != domain.IdempotencyFailedFinal {
			return ModelExecutionFinalizeResult{}, fmt.Errorf("failed model execution must finalize EXECUTING -> FAILED")
		}
	} else {
		return ModelExecutionFinalizeResult{}, fmt.Errorf("unsupported model execution final status %s", command.Task.Status)
	}

	var nextSequence int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence), 0) + 1 FROM task_events WHERE task_id=$1`, current.ID).Scan(&nextSequence); err != nil {
		return ModelExecutionFinalizeResult{}, fmt.Errorf("allocate model finalization event sequence: %w", err)
	}
	versionIncrement := int64(len(command.Transitions))
	var nextVersion int64
	err = tx.QueryRowContext(ctx, `UPDATE tasks SET agent_id=$2, input=$3,
		status=$4, result=$5, cost=$6, estimated_cost=$7, actual_cost=$8,
		currency=$9, route_decision_id=$10, trace_id=$11, updated_at=$12,
		completed_at=$13, version=version+$14
		WHERE id=$1 AND version=$15
		RETURNING version`,
		command.Task.ID, command.Task.AgentID, command.Task.Input, command.Task.Status,
		command.Task.Result, command.Task.Cost, command.Task.EstimatedCost,
		command.Task.ActualCost, command.Task.Currency, command.Task.RouteDecisionID,
		command.Task.TraceID, command.Task.UpdatedAt, command.Task.CompletedAt,
		versionIncrement, command.Task.Version).Scan(&nextVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return ModelExecutionFinalizeResult{}, ErrVersionConflict
	}
	if err != nil {
		return ModelExecutionFinalizeResult{}, fmt.Errorf("update final model execution task projection: %w", err)
	}
	command.Task.Version = nextVersion

	now := time.Now().UTC()
	persistedEvents := make([]domain.TaskEvent, 0, len(command.Events))
	for i := range command.Events {
		event := command.Events[i]
		event.TenantID = current.TenantID
		event.ProjectID = current.ProjectID
		event.TaskID = current.ID
		event.Sequence = nextSequence + int64(i)
		event.TraceID = command.Task.TraceID
		if event.OccurredAt.IsZero() {
			event.OccurredAt = command.Transitions[i].At
		}
		if event.CreatedAt.IsZero() {
			event.CreatedAt = now
		}
		if event.SchemaVersion == 0 {
			event.SchemaVersion = 1
		}
		if err := event.Validate(); err != nil {
			return ModelExecutionFinalizeResult{}, err
		}
		if err := insertTaskEvent(ctx, tx, event); err != nil {
			return ModelExecutionFinalizeResult{}, err
		}
		persistedEvents = append(persistedEvents, event)
	}

	if command.Idempotency.ResourceID == "" {
		command.Idempotency.ResourceID = current.ID
	}
	if err := completeIdempotency(ctx, tx, command.Idempotency); err != nil {
		return ModelExecutionFinalizeResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return ModelExecutionFinalizeResult{}, fmt.Errorf("commit model execution finalization transaction: %w", err)
	}
	return ModelExecutionFinalizeResult{
		Task: command.Task, Events: persistedEvents, Idempotency: command.Idempotency,
	}, nil
}

// MarkModelExecutionRetryable releases an in-progress logical model command for
// an explicit retry while leaving the Task in EXECUTING. A later request with
// the same idempotency key may reacquire the command; it will not append a
// second TaskExecutionStarted event.
func (r *ScopedPostgresTaskCommands) MarkModelExecutionRetryable(ctx context.Context, record domain.IdempotencyRecord) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("database is required")
	}
	if err := record.Validate(); err != nil {
		return err
	}
	if record.Status != domain.IdempotencyFailedRetryable {
		return fmt.Errorf("retryable model execution result requires failed_retryable idempotency status")
	}
	principal, err := identity.RequireProject(ctx)
	if err != nil {
		return err
	}
	if record.TenantID != principal.TenantID || record.ProjectID != principal.ProjectID {
		return fmt.Errorf("idempotency scope must match authenticated principal")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin retryable model result transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `SELECT
		set_config('aicloud.tenant_id', $1, true),
		set_config('aicloud.project_id', $2, true)`, principal.TenantID, principal.ProjectID); err != nil {
		return fmt.Errorf("set retryable model result transaction scope: %w", err)
	}
	existing, err := loadIdempotencyForUpdate(ctx, tx, record)
	if err != nil {
		return err
	}
	if existing.RequestDigest != record.RequestDigest {
		return ErrIdempotencyConflict
	}
	if existing.Status != domain.IdempotencyInProgress {
		return fmt.Errorf("model execution command is not in progress")
	}
	if err := completeIdempotency(ctx, tx, record); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit retryable model result transaction: %w", err)
	}
	return nil
}

func insertTaskEvent(ctx context.Context, tx *sql.Tx, event domain.TaskEvent) error {
	if _, err := tx.ExecContext(ctx, `INSERT INTO task_events(
		event_id, tenant_id, project_id, task_id, sequence, event_type,
		actor_principal_type, actor_subject_id, payload, request_id, trace_id,
		schema_version, occurred_at, created_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,NULLIF($10,''),$11,$12,$13,$14)`,
		event.EventID, event.TenantID, event.ProjectID, event.TaskID, event.Sequence,
		event.EventType, event.Actor.PrincipalType, event.Actor.SubjectID,
		string(event.Payload), event.RequestID, event.TraceID, event.SchemaVersion,
		event.OccurredAt, event.CreatedAt); err != nil {
		return fmt.Errorf("append task event: %w", err)
	}
	return nil
}
