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

var (
	ErrIdempotencyConflict   = errors.New("idempotency key conflicts with a different request")
	ErrIdempotencyInProgress = errors.New("idempotent command is already in progress")
)

type TaskCommandCommit struct {
	Task        domain.Task
	Transition  domain.TaskTransition
	Event       domain.TaskEvent
	Outbox      []domain.OutboxMessage
	Idempotency domain.IdempotencyRecord
}

type TaskCommandCommitResult struct {
	Task        domain.Task
	Event       domain.TaskEvent
	Outbox      []domain.OutboxMessage
	Idempotency domain.IdempotencyRecord
	Replayed    bool
}

// ScopedPostgresTaskCommands is the R6 business-transaction boundary. One
// CommitTransition call owns exactly one SQL transaction containing the Task
// projection update, canonical TaskEvent append, required Outbox messages and
// command Idempotency result.
type ScopedPostgresTaskCommands struct {
	db *sql.DB
}

func NewScopedPostgresTaskCommands(db *sql.DB) *ScopedPostgresTaskCommands {
	return &ScopedPostgresTaskCommands{db: db}
}

func (r *ScopedPostgresTaskCommands) CommitTransition(ctx context.Context, command TaskCommandCommit) (TaskCommandCommitResult, error) {
	if r == nil || r.db == nil {
		return TaskCommandCommitResult{}, fmt.Errorf("database is required")
	}
	if err := command.Idempotency.Validate(); err != nil {
		return TaskCommandCommitResult{}, err
	}
	if command.Idempotency.Status == domain.IdempotencyInProgress {
		return TaskCommandCommitResult{}, fmt.Errorf("final idempotency status must not be in_progress")
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

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return TaskCommandCommitResult{}, fmt.Errorf("begin task command transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `SELECT
		set_config('aicloud.tenant_id', $1, true),
		set_config('aicloud.project_id', $2, true)`, principal.TenantID, principal.ProjectID); err != nil {
		return TaskCommandCommitResult{}, fmt.Errorf("set task command transaction scope: %w", err)
	}

	replayed, existing, err := reserveIdempotency(ctx, tx, command.Idempotency)
	if err != nil {
		return TaskCommandCommitResult{}, err
	}
	if replayed {
		if err := tx.Commit(); err != nil {
			return TaskCommandCommitResult{}, fmt.Errorf("commit idempotency replay transaction: %w", err)
		}
		return TaskCommandCommitResult{Idempotency: existing, Replayed: true}, nil
	}

	current, err := scanScopedTask(tx.QueryRowContext(ctx, `SELECT `+scopedTaskColumns+` FROM tasks WHERE id=$1 FOR UPDATE`, command.Task.ID))
	if errors.Is(err, sql.ErrNoRows) {
		return TaskCommandCommitResult{}, ErrNotFound
	}
	if err != nil {
		return TaskCommandCommitResult{}, fmt.Errorf("lock scoped task for command: %w", err)
	}
	if !principal.OwnsProject(current.TenantID, current.ProjectID) {
		return TaskCommandCommitResult{}, ErrNotFound
	}
	if current.TenantID != command.Task.TenantID || current.ProjectID != command.Task.ProjectID || current.CreatedBy != command.Task.CreatedBy {
		return TaskCommandCommitResult{}, fmt.Errorf("task tenant, project and creator identity are immutable")
	}
	if command.Task.Version != current.Version {
		return TaskCommandCommitResult{}, ErrVersionConflict
	}
	if command.Transition.From != current.Status || command.Transition.To != command.Task.Status {
		return TaskCommandCommitResult{}, fmt.Errorf("task transition evidence does not match persisted state")
	}
	if strings.TrimSpace(command.Transition.Actor) == "" || strings.TrimSpace(command.Transition.Cause) == "" || command.Transition.At.IsZero() {
		return TaskCommandCommitResult{}, fmt.Errorf("task transition evidence is incomplete")
	}
	canonicalEventType, ok := domain.CanonicalTaskStateEvent(command.Task.Status)
	if !ok || command.Event.EventType != canonicalEventType {
		return TaskCommandCommitResult{}, fmt.Errorf("canonical event mismatch for task state %s", command.Task.Status)
	}

	var nextSequence int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence), 0) + 1 FROM task_events WHERE task_id=$1`, current.ID).Scan(&nextSequence); err != nil {
		return TaskCommandCommitResult{}, fmt.Errorf("allocate task event sequence: %w", err)
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
		return TaskCommandCommitResult{}, ErrVersionConflict
	}
	if err != nil {
		return TaskCommandCommitResult{}, fmt.Errorf("update task projection in command transaction: %w", err)
	}
	command.Task.Version = nextVersion

	now := time.Now().UTC()
	command.Event.TenantID = current.TenantID
	command.Event.ProjectID = current.ProjectID
	command.Event.TaskID = current.ID
	command.Event.Sequence = nextSequence
	command.Event.TraceID = command.Task.TraceID
	if command.Event.OccurredAt.IsZero() {
		command.Event.OccurredAt = command.Transition.At
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
		return TaskCommandCommitResult{}, fmt.Errorf("append task event: %w", err)
	}

	for i := range command.Outbox {
		message := &command.Outbox[i]
		if message.TenantID == "" {
			message.TenantID = current.TenantID
		}
		if message.ProjectID == "" {
			message.ProjectID = current.ProjectID
		}
		if message.TaskID == "" && message.AggregateType == "Task" && message.AggregateID == current.ID {
			message.TaskID = current.ID
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
		if message.TenantID != current.TenantID || message.ProjectID != current.ProjectID {
			return TaskCommandCommitResult{}, fmt.Errorf("outbox scope must match task scope")
		}
		if message.TaskID != "" && message.TaskID != current.ID {
			return TaskCommandCommitResult{}, fmt.Errorf("outbox task id must match command task")
		}
		if err := message.Validate(); err != nil {
			return TaskCommandCommitResult{}, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO outbox_messages(
			outbox_id, tenant_id, project_id, task_id, aggregate_type, aggregate_id,
			event_type, payload, destination, idempotency_key, status, attempts,
			available_at, created_at, delivered_at
		) VALUES ($1,$2,$3,NULLIF($4,''),$5,$6,$7,$8::jsonb,$9,$10,$11,$12,$13,$14,$15)`,
			message.OutboxID, message.TenantID, message.ProjectID, message.TaskID,
			message.AggregateType, message.AggregateID, message.EventType,
			string(message.Payload), message.Destination, message.IdempotencyKey,
			message.Status, message.Attempts, message.AvailableAt, message.CreatedAt,
			message.DeliveredAt); err != nil {
			return TaskCommandCommitResult{}, fmt.Errorf("append outbox message: %w", err)
		}
	}

	if command.Idempotency.ResourceID == "" {
		command.Idempotency.ResourceID = current.ID
	}
	if err := completeIdempotency(ctx, tx, command.Idempotency); err != nil {
		return TaskCommandCommitResult{}, err
	}

	if err := tx.Commit(); err != nil {
		return TaskCommandCommitResult{}, fmt.Errorf("commit task command transaction: %w", err)
	}
	return TaskCommandCommitResult{
		Task: command.Task, Event: command.Event, Outbox: command.Outbox,
		Idempotency: command.Idempotency,
	}, nil
}

func reserveIdempotency(ctx context.Context, tx *sql.Tx, desired domain.IdempotencyRecord) (bool, domain.IdempotencyRecord, error) {
	var inserted string
	err := tx.QueryRowContext(ctx, `INSERT INTO idempotency_records(
		tenant_id, project_id, operation, idempotency_key, request_digest, status,
		created_at, expires_at
	) VALUES ($1,$2,$3,$4,$5,'in_progress',$6,$7)
	ON CONFLICT (tenant_id, project_id, operation, idempotency_key) DO NOTHING
	RETURNING status`, desired.TenantID, desired.ProjectID, desired.Operation,
		desired.Key, desired.RequestDigest, desired.CreatedAt, desired.ExpiresAt).Scan(&inserted)
	if err == nil {
		return false, domain.IdempotencyRecord{}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return false, domain.IdempotencyRecord{}, fmt.Errorf("reserve idempotency record: %w", err)
	}

	existing, err := loadIdempotencyForUpdate(ctx, tx, desired)
	if err != nil {
		return false, domain.IdempotencyRecord{}, err
	}
	if existing.RequestDigest != desired.RequestDigest {
		return false, domain.IdempotencyRecord{}, ErrIdempotencyConflict
	}
	switch existing.Status {
	case domain.IdempotencyCompleted, domain.IdempotencyFailedFinal:
		return true, existing, nil
	case domain.IdempotencyInProgress:
		return false, domain.IdempotencyRecord{}, ErrIdempotencyInProgress
	case domain.IdempotencyFailedRetryable:
		if _, err := tx.ExecContext(ctx, `UPDATE idempotency_records
			SET status='in_progress', resource_id=NULL, response_code=NULL,
				response_digest=NULL, response_payload=NULL
			WHERE tenant_id=$1 AND project_id=$2 AND operation=$3 AND idempotency_key=$4`,
			desired.TenantID, desired.ProjectID, desired.Operation, desired.Key); err != nil {
			return false, domain.IdempotencyRecord{}, fmt.Errorf("reacquire retryable idempotency record: %w", err)
		}
		return false, domain.IdempotencyRecord{}, nil
	default:
		return false, domain.IdempotencyRecord{}, fmt.Errorf("unsupported idempotency status %q", existing.Status)
	}
}

func loadIdempotencyForUpdate(ctx context.Context, tx *sql.Tx, key domain.IdempotencyRecord) (domain.IdempotencyRecord, error) {
	var record domain.IdempotencyRecord
	var responsePayload []byte
	err := tx.QueryRowContext(ctx, `SELECT tenant_id, project_id, operation,
		idempotency_key, request_digest, status, COALESCE(resource_id,''),
		COALESCE(response_code,0), COALESCE(response_digest,''),
		COALESCE(response_payload,'null'::jsonb)::text, created_at, expires_at
		FROM idempotency_records
		WHERE tenant_id=$1 AND project_id=$2 AND operation=$3 AND idempotency_key=$4
		FOR UPDATE`, key.TenantID, key.ProjectID, key.Operation, key.Key).Scan(
		&record.TenantID, &record.ProjectID, &record.Operation, &record.Key,
		&record.RequestDigest, &record.Status, &record.ResourceID,
		&record.ResponseCode, &record.ResponseDigest, &responsePayload,
		&record.CreatedAt, &record.ExpiresAt,
	)
	if err != nil {
		return domain.IdempotencyRecord{}, fmt.Errorf("load idempotency record: %w", err)
	}
	if string(responsePayload) != "null" {
		record.ResponsePayload = append(record.ResponsePayload[:0], responsePayload...)
	}
	return record, nil
}

func completeIdempotency(ctx context.Context, tx *sql.Tx, record domain.IdempotencyRecord) error {
	var payload any
	if len(record.ResponsePayload) > 0 {
		payload = string(record.ResponsePayload)
	}
	result, err := tx.ExecContext(ctx, `UPDATE idempotency_records SET
		status=$5, resource_id=NULLIF($6,''), response_code=NULLIF($7,0),
		response_digest=NULLIF($8,''), response_payload=$9::jsonb
		WHERE tenant_id=$1 AND project_id=$2 AND operation=$3 AND idempotency_key=$4
			AND request_digest=$10`,
		record.TenantID, record.ProjectID, record.Operation, record.Key,
		record.Status, record.ResourceID, record.ResponseCode, record.ResponseDigest,
		payload, record.RequestDigest)
	if err != nil {
		return fmt.Errorf("complete idempotency record: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read idempotency completion rows: %w", err)
	}
	if affected != 1 {
		return fmt.Errorf("complete idempotency record affected %d rows", affected)
	}
	return nil
}
