package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/identity"
)

type RouteTaskCommandCommit struct {
	Task        domain.Task
	Transition  domain.TaskTransition
	Decision    domain.RouteDecision
	Event       domain.TaskEvent
	Outbox      []domain.OutboxMessage
	Idempotency domain.IdempotencyRecord
}

type RouteTaskCommandResult struct {
	Task        domain.Task
	Decision    domain.RouteDecision
	Event       domain.TaskEvent
	Outbox      []domain.OutboxMessage
	Idempotency domain.IdempotencyRecord
	Replayed    bool
}

// CommitRouteTransition closes the R5 routing dual-write boundary. The
// RouteDecision is persisted in the same SQL transaction as the Task
// PLANNING->ROUTING projection update, canonical TaskEvent, optional Outbox and
// command Idempotency result.
func (r *ScopedPostgresTaskCommands) CommitRouteTransition(ctx context.Context, command RouteTaskCommandCommit) (RouteTaskCommandResult, error) {
	if r == nil || r.db == nil {
		return RouteTaskCommandResult{}, fmt.Errorf("database is required")
	}
	if err := command.Idempotency.Validate(); err != nil {
		return RouteTaskCommandResult{}, err
	}
	if command.Idempotency.Status != domain.IdempotencyCompleted {
		return RouteTaskCommandResult{}, fmt.Errorf("successful route command requires completed idempotency status")
	}

	principal, err := identity.RequireProject(ctx)
	if err != nil {
		return RouteTaskCommandResult{}, err
	}
	if principal.Type == identity.PrincipalSystem && !principal.HasCapability(identity.CapabilityTaskSystemAccess) {
		return RouteTaskCommandResult{}, fmt.Errorf("%w: %s", identity.ErrCapabilityRequired, identity.CapabilityTaskSystemAccess)
	}
	if command.Idempotency.TenantID != principal.TenantID || command.Idempotency.ProjectID != principal.ProjectID {
		return RouteTaskCommandResult{}, fmt.Errorf("idempotency scope must match authenticated principal")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return RouteTaskCommandResult{}, fmt.Errorf("begin route command transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `SELECT
		set_config('aicloud.tenant_id', $1, true),
		set_config('aicloud.project_id', $2, true)`, principal.TenantID, principal.ProjectID); err != nil {
		return RouteTaskCommandResult{}, fmt.Errorf("set route command transaction scope: %w", err)
	}

	replayed, existing, err := reserveIdempotency(ctx, tx, command.Idempotency)
	if err != nil {
		return RouteTaskCommandResult{}, err
	}
	if replayed {
		result, err := replayRouteCommand(ctx, tx, existing)
		if err != nil {
			return RouteTaskCommandResult{}, err
		}
		if err := tx.Commit(); err != nil {
			return RouteTaskCommandResult{}, fmt.Errorf("commit route replay transaction: %w", err)
		}
		result.Replayed = true
		return result, nil
	}

	current, err := scanScopedTask(tx.QueryRowContext(ctx, `SELECT `+scopedTaskColumns+` FROM tasks WHERE id=$1 FOR UPDATE`, command.Task.ID))
	if errors.Is(err, sql.ErrNoRows) {
		return RouteTaskCommandResult{}, ErrNotFound
	}
	if err != nil {
		return RouteTaskCommandResult{}, fmt.Errorf("lock scoped task for route command: %w", err)
	}
	if !principal.OwnsProject(current.TenantID, current.ProjectID) {
		return RouteTaskCommandResult{}, ErrNotFound
	}
	if current.TenantID != command.Task.TenantID || current.ProjectID != command.Task.ProjectID || current.CreatedBy != command.Task.CreatedBy {
		return RouteTaskCommandResult{}, fmt.Errorf("task tenant, project and creator identity are immutable")
	}
	if command.Task.Version != current.Version {
		return RouteTaskCommandResult{}, ErrVersionConflict
	}
	if current.Status != domain.TaskPlanning || command.Transition.From != domain.TaskPlanning || command.Transition.To != domain.TaskRouting || command.Task.Status != domain.TaskRouting {
		return RouteTaskCommandResult{}, fmt.Errorf("route command requires PLANNING -> ROUTING transition")
	}
	if strings.TrimSpace(command.Transition.Actor) == "" || strings.TrimSpace(command.Transition.Cause) == "" || command.Transition.At.IsZero() {
		return RouteTaskCommandResult{}, fmt.Errorf("task transition evidence is incomplete")
	}
	if strings.TrimSpace(command.Decision.ID) == "" {
		return RouteTaskCommandResult{}, fmt.Errorf("route decision id is required")
	}
	if command.Decision.TaskID != "" && command.Decision.TaskID != current.ID {
		return RouteTaskCommandResult{}, fmt.Errorf("route decision task id must match command task")
	}
	command.Decision.TaskID = current.ID
	if command.Decision.CreatedAt.IsZero() {
		command.Decision.CreatedAt = command.Transition.At
	}
	command.Task.RouteDecisionID = command.Decision.ID
	command.Task.EstimatedCost = command.Decision.Selected.EstimatedCost

	var nextSequence int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence), 0) + 1 FROM task_events WHERE task_id=$1`, current.ID).Scan(&nextSequence); err != nil {
		return RouteTaskCommandResult{}, fmt.Errorf("allocate route TaskEvent sequence: %w", err)
	}

	if err := insertRouteDecisionTx(ctx, tx, command.Decision); err != nil {
		return RouteTaskCommandResult{}, err
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
		return RouteTaskCommandResult{}, ErrVersionConflict
	}
	if err != nil {
		return RouteTaskCommandResult{}, fmt.Errorf("update task projection in route transaction: %w", err)
	}
	command.Task.Version = nextVersion

	now := time.Now().UTC()
	command.Event.TenantID = current.TenantID
	command.Event.ProjectID = current.ProjectID
	command.Event.TaskID = current.ID
	command.Event.Sequence = nextSequence
	command.Event.TraceID = command.Task.TraceID
	if command.Event.EventType == "" {
		command.Event.EventType = "TaskRoutingStarted"
	}
	if command.Event.EventType != "TaskRoutingStarted" {
		return RouteTaskCommandResult{}, fmt.Errorf("route command must append TaskRoutingStarted")
	}
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
		return RouteTaskCommandResult{}, err
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
		return RouteTaskCommandResult{}, fmt.Errorf("append route TaskEvent: %w", err)
	}

	for i := range command.Outbox {
		message := &command.Outbox[i]
		prepareTaskOutboxMessage(message, command.Task, now)
		if message.TenantID != current.TenantID || message.ProjectID != current.ProjectID {
			return RouteTaskCommandResult{}, fmt.Errorf("outbox scope must match task scope")
		}
		if message.TaskID != "" && message.TaskID != current.ID {
			return RouteTaskCommandResult{}, fmt.Errorf("outbox task id must match route command task")
		}
		if err := message.Validate(); err != nil {
			return RouteTaskCommandResult{}, err
		}
		if err := insertOutboxMessage(ctx, tx, *message); err != nil {
			return RouteTaskCommandResult{}, err
		}
	}

	responsePayload, err := json.Marshal(command.Decision)
	if err != nil {
		return RouteTaskCommandResult{}, fmt.Errorf("encode durable route result: %w", err)
	}
	command.Idempotency.ResourceID = current.ID
	command.Idempotency.ResponsePayload = responsePayload
	if command.Idempotency.ResponseCode == 0 {
		command.Idempotency.ResponseCode = 201
	}
	if err := completeIdempotency(ctx, tx, command.Idempotency); err != nil {
		return RouteTaskCommandResult{}, err
	}

	if err := tx.Commit(); err != nil {
		return RouteTaskCommandResult{}, fmt.Errorf("commit route command transaction: %w", err)
	}
	return RouteTaskCommandResult{
		Task: command.Task, Decision: command.Decision, Event: command.Event,
		Outbox: command.Outbox, Idempotency: command.Idempotency,
	}, nil
}

func insertRouteDecisionTx(ctx context.Context, tx *sql.Tx, decision domain.RouteDecision) error {
	selected, err := json.Marshal(decision.Selected)
	if err != nil {
		return fmt.Errorf("encode selected route candidate: %w", err)
	}
	candidates, err := json.Marshal(decision.Candidates)
	if err != nil {
		return fmt.Errorf("encode route candidates: %w", err)
	}
	fallback, err := json.Marshal(decision.FallbackChain)
	if err != nil {
		return fmt.Errorf("encode route fallback chain: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO route_decisions (
		id, task_id, selected, candidates, reason, fallback_chain,
		evidence_version, policy_version, created_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		decision.ID, decision.TaskID, selected, candidates, decision.Reason, fallback,
		decision.EvidenceVersion, decision.PolicyVersion, decision.CreatedAt); err != nil {
		return fmt.Errorf("persist route decision in task transaction: %w", err)
	}
	return nil
}

func replayRouteCommand(ctx context.Context, tx *sql.Tx, record domain.IdempotencyRecord) (RouteTaskCommandResult, error) {
	if record.Status != domain.IdempotencyCompleted || record.ResourceID == "" {
		return RouteTaskCommandResult{}, fmt.Errorf("completed route replay is missing task resource identity")
	}
	if len(record.ResponsePayload) == 0 {
		return RouteTaskCommandResult{}, fmt.Errorf("completed route replay is missing durable response payload")
	}
	var decision domain.RouteDecision
	if err := json.Unmarshal(record.ResponsePayload, &decision); err != nil {
		return RouteTaskCommandResult{}, fmt.Errorf("decode durable route replay result: %w", err)
	}
	task, err := scanScopedTask(tx.QueryRowContext(ctx, `SELECT `+scopedTaskColumns+` FROM tasks WHERE id=$1`, record.ResourceID))
	if errors.Is(err, sql.ErrNoRows) {
		return RouteTaskCommandResult{}, fmt.Errorf("route replay references missing task %q", record.ResourceID)
	}
	if err != nil {
		return RouteTaskCommandResult{}, fmt.Errorf("load task for route replay: %w", err)
	}
	return RouteTaskCommandResult{Task: task, Decision: decision, Idempotency: record}, nil
}
