package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	execution "github.com/tommyxie2026-tech/aicloud/execution"
	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/identity"
)

type routeSelectionReplayPayload struct {
	Plan     execution.PlanRevisionRef       `json:"plan"`
	Node     execution.NodeID                `json:"node"`
	Decision domain.RouteDecision            `json:"decision"`
	Policy   execution.PolicyDecisionRecord  `json:"policy"`
	Target   execution.ExecutionTarget       `json:"target"`
}

var _ RouteSelectionStore = (*ScopedPostgresTaskCommands)(nil)

func (r *ScopedPostgresTaskCommands) ResolveRouteSelection(ctx context.Context, lookup IdempotencyLookup) (RouteSelectionResult, bool, error) {
	record, found, err := r.ResolveIdempotency(ctx, lookup)
	if err != nil || !found {
		return RouteSelectionResult{}, false, err
	}
	if record.Status != domain.IdempotencyCompleted || record.ResourceID == "" || len(record.ResponsePayload) == 0 {
		return RouteSelectionResult{}, true, fmt.Errorf("completed route selection replay is incomplete")
	}
	var payload routeSelectionReplayPayload
	if err := json.Unmarshal(record.ResponsePayload, &payload); err != nil {
		return RouteSelectionResult{}, true, fmt.Errorf("decode route selection replay: %w", err)
	}
	if payload.Decision.ID == "" || payload.Decision.TaskID != record.ResourceID {
		return RouteSelectionResult{}, true, fmt.Errorf("route selection replay decision lineage is invalid")
	}
	if payload.Target.ID == "" || payload.Target.Snapshot.Digest == "" {
		return RouteSelectionResult{}, true, fmt.Errorf("route selection replay is missing frozen target evidence")
	}
	return RouteSelectionResult{Plan: payload.Plan, Node: payload.Node, Decision: payload.Decision, Policy: payload.Policy, Target: payload.Target, Idempotency: record, Replayed: true}, true, nil
}

func (r *ScopedPostgresTaskCommands) CommitRouteSelection(ctx context.Context, command RouteSelectionCommit) (RouteSelectionResult, error) {
	if r == nil || r.db == nil {
		return RouteSelectionResult{}, fmt.Errorf("database is required")
	}
	if err := command.Idempotency.Validate(); err != nil {
		return RouteSelectionResult{}, err
	}
	if command.Idempotency.Status != domain.IdempotencyCompleted {
		return RouteSelectionResult{}, fmt.Errorf("successful route selection requires completed idempotency status")
	}
	if strings.TrimSpace(command.Decision.ID) == "" || strings.TrimSpace(command.Decision.TaskID) == "" {
		return RouteSelectionResult{}, fmt.Errorf("route decision id and task id are required")
	}
	if command.Target.ID == "" || command.Target.Snapshot.Digest == "" {
		return RouteSelectionResult{}, fmt.Errorf("frozen execution target identity and digest are required")
	}

	principal, err := identity.RequireProject(ctx)
	if err != nil {
		return RouteSelectionResult{}, err
	}
	if principal.Type == identity.PrincipalSystem && !principal.HasCapability(identity.CapabilityTaskSystemAccess) {
		return RouteSelectionResult{}, fmt.Errorf("%w: %s", identity.ErrCapabilityRequired, identity.CapabilityTaskSystemAccess)
	}
	if command.Idempotency.TenantID != principal.TenantID || command.Idempotency.ProjectID != principal.ProjectID {
		return RouteSelectionResult{}, fmt.Errorf("idempotency scope must match authenticated principal")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return RouteSelectionResult{}, fmt.Errorf("begin route selection transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `SELECT set_config('aicloud.tenant_id', $1, true), set_config('aicloud.project_id', $2, true)`, principal.TenantID, principal.ProjectID); err != nil {
		return RouteSelectionResult{}, fmt.Errorf("set route selection transaction scope: %w", err)
	}

	replayed, existing, err := reserveIdempotency(ctx, tx, command.Idempotency)
	if err != nil {
		return RouteSelectionResult{}, err
	}
	if replayed {
		result, err := replayRouteSelection(ctx, tx, existing)
		if err != nil {
			return RouteSelectionResult{}, err
		}
		if err := tx.Commit(); err != nil {
			return RouteSelectionResult{}, fmt.Errorf("commit route selection replay: %w", err)
		}
		result.Replayed = true
		return result, nil
	}

	current, err := scanScopedTask(tx.QueryRowContext(ctx, `SELECT `+scopedTaskColumns+` FROM tasks WHERE id=$1 FOR UPDATE`, command.Task.ID))
	if errors.Is(err, sql.ErrNoRows) {
		return RouteSelectionResult{}, ErrNotFound
	}
	if err != nil {
		return RouteSelectionResult{}, fmt.Errorf("lock task for route selection: %w", err)
	}
	if !principal.OwnsProject(current.TenantID, current.ProjectID) {
		return RouteSelectionResult{}, ErrNotFound
	}
	if current.Status != domain.TaskRouting || command.Task.Status != domain.TaskRouting {
		return RouteSelectionResult{}, fmt.Errorf("route selection requires TaskStatus=ROUTING")
	}
	if command.Task.Version != current.Version {
		return RouteSelectionResult{}, ErrVersionConflict
	}
	if command.Task.TenantID != current.TenantID || command.Task.ProjectID != current.ProjectID || command.Task.CreatedBy != current.CreatedBy {
		return RouteSelectionResult{}, fmt.Errorf("task tenant, project and creator identity are immutable")
	}
	if command.Decision.TaskID != current.ID {
		return RouteSelectionResult{}, fmt.Errorf("route decision task id must match command task")
	}
	if command.Decision.CreatedAt.IsZero() {
		return RouteSelectionResult{}, fmt.Errorf("route decision created time is required")
	}
	if err := insertRouteDecisionTx(ctx, tx, command.Decision); err != nil {
		return RouteSelectionResult{}, err
	}

	updatedAt := command.Task.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}
	var nextVersion int64
	err = tx.QueryRowContext(ctx, `UPDATE tasks SET route_decision_id=$2, estimated_cost=$3, updated_at=$4, version=version+1 WHERE id=$1 AND version=$5 AND status=$6 RETURNING version`, current.ID, command.Decision.ID, command.Decision.Selected.EstimatedCost, updatedAt, current.Version, domain.TaskRouting).Scan(&nextVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return RouteSelectionResult{}, ErrVersionConflict
	}
	if err != nil {
		return RouteSelectionResult{}, fmt.Errorf("update task route selection projection: %w", err)
	}
	command.Task = current
	command.Task.RouteDecisionID = command.Decision.ID
	command.Task.EstimatedCost = command.Decision.Selected.EstimatedCost
	command.Task.UpdatedAt = updatedAt
	command.Task.Version = nextVersion

	var nextSequence int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence),0)+1 FROM task_events WHERE task_id=$1`, current.ID).Scan(&nextSequence); err != nil {
		return RouteSelectionResult{}, fmt.Errorf("allocate route selection TaskEvent sequence: %w", err)
	}
	now := time.Now().UTC()
	command.Event.TenantID = current.TenantID
	command.Event.ProjectID = current.ProjectID
	command.Event.TaskID = current.ID
	command.Event.Sequence = nextSequence
	command.Event.TraceID = current.TraceID
	if command.Event.EventType == "" {
		command.Event.EventType = "TaskRouteSelected"
	}
	if command.Event.EventType != "TaskRouteSelected" {
		return RouteSelectionResult{}, fmt.Errorf("route selection must append TaskRouteSelected")
	}
	if command.Event.OccurredAt.IsZero() {
		command.Event.OccurredAt = command.Decision.CreatedAt
	}
	if command.Event.CreatedAt.IsZero() {
		command.Event.CreatedAt = now
	}
	if command.Event.SchemaVersion == 0 {
		command.Event.SchemaVersion = 1
	}
	if err := command.Event.Validate(); err != nil {
		return RouteSelectionResult{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO task_events(event_id, tenant_id, project_id, task_id, sequence, event_type, actor_principal_type, actor_subject_id, payload, request_id, trace_id, schema_version, occurred_at, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,NULLIF($10,''),$11,$12,$13,$14)`, command.Event.EventID, command.Event.TenantID, command.Event.ProjectID, command.Event.TaskID, command.Event.Sequence, command.Event.EventType, command.Event.Actor.PrincipalType, command.Event.Actor.SubjectID, string(command.Event.Payload), command.Event.RequestID, command.Event.TraceID, command.Event.SchemaVersion, command.Event.OccurredAt, command.Event.CreatedAt); err != nil {
		return RouteSelectionResult{}, fmt.Errorf("append route selection TaskEvent: %w", err)
	}

	responsePayload, err := json.Marshal(routeSelectionReplayPayload{Plan: command.Plan, Node: command.Node, Decision: command.Decision, Policy: command.Policy, Target: command.Target})
	if err != nil {
		return RouteSelectionResult{}, fmt.Errorf("encode route selection result: %w", err)
	}
	command.Idempotency.ResourceID = current.ID
	command.Idempotency.ResponsePayload = responsePayload
	if command.Idempotency.ResponseCode == 0 {
		command.Idempotency.ResponseCode = 200
	}
	if err := completeIdempotency(ctx, tx, command.Idempotency); err != nil {
		return RouteSelectionResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return RouteSelectionResult{}, fmt.Errorf("commit route selection transaction: %w", err)
	}
	return RouteSelectionResult{Task: command.Task, Plan: command.Plan, Node: command.Node, Decision: command.Decision, Policy: command.Policy, Target: command.Target, Event: command.Event, Idempotency: command.Idempotency}, nil
}

func replayRouteSelection(ctx context.Context, tx *sql.Tx, record domain.IdempotencyRecord) (RouteSelectionResult, error) {
	if record.Status != domain.IdempotencyCompleted || record.ResourceID == "" || len(record.ResponsePayload) == 0 {
		return RouteSelectionResult{}, fmt.Errorf("completed route selection replay is incomplete")
	}
	var payload routeSelectionReplayPayload
	if err := json.Unmarshal(record.ResponsePayload, &payload); err != nil {
		return RouteSelectionResult{}, fmt.Errorf("decode route selection replay: %w", err)
	}
	if payload.Target.ID == "" || payload.Target.Snapshot.Digest == "" {
		return RouteSelectionResult{}, fmt.Errorf("route selection replay is missing frozen target evidence")
	}
	task, err := scanScopedTask(tx.QueryRowContext(ctx, `SELECT `+scopedTaskColumns+` FROM tasks WHERE id=$1`, record.ResourceID))
	if errors.Is(err, sql.ErrNoRows) {
		return RouteSelectionResult{}, fmt.Errorf("route selection replay references missing task %q", record.ResourceID)
	}
	if err != nil {
		return RouteSelectionResult{}, fmt.Errorf("load task for route selection replay: %w", err)
	}
	return RouteSelectionResult{Task: task, Plan: payload.Plan, Node: payload.Node, Decision: payload.Decision, Policy: payload.Policy, Target: payload.Target, Idempotency: record}, nil
}
