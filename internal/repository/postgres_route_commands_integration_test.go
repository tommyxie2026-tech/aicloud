//go:build integration

package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/identity"
)

func TestScopedPostgresRouteCommandAtomicCommitReplayConflictAndRollback(t *testing.T) {
	dsn := os.Getenv("AICLOUD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("AICLOUD_TEST_DATABASE_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open PostgreSQL: %v", err)
	}
	defer db.Close()
	cleanupTaskCommandFixture(t, ctx, db)
	_, _ = db.ExecContext(ctx, `DROP TABLE IF EXISTS route_decisions CASCADE`)
	defer func() {
		_, _ = db.ExecContext(context.Background(), `DROP TABLE IF EXISTS route_decisions CASCADE`)
		cleanupTaskCommandFixture(t, context.Background(), db)
	}()
	createTaskCommandFixture(t, ctx, db)
	if _, err := db.ExecContext(ctx, `CREATE TABLE route_decisions (
		id TEXT PRIMARY KEY,
		task_id TEXT NOT NULL,
		selected JSONB NOT NULL,
		candidates JSONB NOT NULL,
		reason TEXT NOT NULL,
		fallback_chain JSONB NOT NULL,
		evidence_version TEXT NOT NULL DEFAULT '',
		policy_version TEXT NOT NULL DEFAULT '',
		created_at TIMESTAMPTZ NOT NULL
	)`); err != nil {
		t.Fatalf("create route decision fixture: %v", err)
	}

	principal := identity.Principal{
		Type: identity.PrincipalUser, SubjectID: "user-a", TenantID: "tenant-a", ProjectID: "project-a",
		AuthnMethod: "integration-test", Issuer: "test",
	}
	projectCtx := identity.WithPrincipal(ctx, principal)
	repo := NewScopedPostgresTaskCommands(db)
	now := time.Now().UTC()

	planningTask := fixtureTask(now)
	planningTask.Status = domain.TaskPlanning
	planningTask.UpdatedAt = now.Add(time.Second)
	planning, err := repo.CommitTransition(projectCtx, TaskCommandCommit{
		Task: planningTask,
		Transition: domain.TaskTransition{
			From: domain.TaskCreated, To: domain.TaskPlanning,
			Actor: "user:user-a", Cause: "prepare route", At: planningTask.UpdatedAt,
		},
		Event: domain.TaskEvent{
			EventID: "event-plan", EventType: "TaskPlanningStarted",
			Actor: domain.TaskEventActor{PrincipalType: "user", SubjectID: "user-a"},
			Payload: json.RawMessage(`{"to":"PLANNING"}`), SchemaVersion: 1,
		},
		Idempotency: domain.IdempotencyRecord{
			TenantID: "tenant-a", ProjectID: "project-a", Operation: "route:planning", Key: "route-plan-1",
			RequestDigest: "sha256:route", Status: domain.IdempotencyCompleted,
			CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour),
		},
	})
	if err != nil {
		t.Fatalf("commit planning transition: %v", err)
	}

	routingTask := planning.Task
	routingTask.Status = domain.TaskRouting
	routingTask.UpdatedAt = now.Add(2 * time.Second)
	decision := domain.RouteDecision{
		ID: "route-1", TaskID: "task-1",
		Selected: domain.RouteCandidate{
			ModelID: "model-a", ModelVersion: "v1", RouteClass: domain.RouteEfficient, EstimatedCost: 0.01,
		},
		Candidates: []domain.RouteCandidate{{ModelID: "model-a", ModelVersion: "v1"}},
		Reason: "best eligible model", EvidenceVersion: "evidence-v1", PolicyVersion: "policy-v1",
		CreatedAt: routingTask.UpdatedAt,
	}
	command := RouteTaskCommandCommit{
		Task: routingTask,
		Transition: domain.TaskTransition{
			From: domain.TaskPlanning, To: domain.TaskRouting,
			Actor: "user:user-a", Cause: "request model route", At: routingTask.UpdatedAt,
		},
		Decision: decision,
		Event: domain.TaskEvent{
			EventID: "event-route", EventType: "TaskRoutingStarted",
			Actor: domain.TaskEventActor{PrincipalType: "user", SubjectID: "user-a"},
			Payload: json.RawMessage(`{"routeDecisionId":"route-1"}`), SchemaVersion: 1,
		},
		Idempotency: domain.IdempotencyRecord{
			TenantID: "tenant-a", ProjectID: "project-a", Operation: "POST /api/v1/tasks/task-1/route", Key: "route-command-1",
			RequestDigest: "sha256:route", Status: domain.IdempotencyCompleted,
			CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour),
		},
	}

	result, err := repo.CommitRouteTransition(projectCtx, command)
	if err != nil {
		t.Fatalf("CommitRouteTransition returned error: %v", err)
	}
	if result.Replayed || result.Task.Version != 3 || result.Task.RouteDecisionID != "route-1" || result.Event.Sequence != 2 {
		t.Fatalf("unexpected route commit result: %#v", result)
	}
	var routeCount int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM route_decisions WHERE id='route-1' AND task_id='task-1'`).Scan(&routeCount); err != nil {
		t.Fatalf("count route decisions: %v", err)
	}
	if routeCount != 1 {
		t.Fatalf("route decision count=%d want=1", routeCount)
	}
	var status domain.TaskStatus
	var version int64
	var routeID string
	if err := db.QueryRowContext(ctx, `SELECT status, version, route_decision_id FROM tasks WHERE id='task-1'`).Scan(&status, &version, &routeID); err != nil {
		t.Fatalf("read routed task: %v", err)
	}
	if status != domain.TaskRouting || version != 3 || routeID != "route-1" {
		t.Fatalf("task status=%s version=%d route=%q", status, version, routeID)
	}

	replay, err := repo.CommitRouteTransition(projectCtx, command)
	if err != nil {
		t.Fatalf("route replay: %v", err)
	}
	if !replay.Replayed || replay.Decision.ID != "route-1" || replay.Task.ID != "task-1" {
		t.Fatalf("unexpected route replay: %#v", replay)
	}

	conflict := command
	conflict.Idempotency.RequestDigest = "sha256:changed"
	if _, err := repo.CommitRouteTransition(projectCtx, conflict); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("changed route request error=%v want ErrIdempotencyConflict", err)
	}

	if _, err := db.ExecContext(ctx, `
		DELETE FROM idempotency_records WHERE operation='POST /api/v1/tasks/task-1/route';
		DELETE FROM task_events WHERE event_id='event-route';
		DELETE FROM route_decisions WHERE id='route-1';
		UPDATE tasks SET status='PLANNING', version=2, route_decision_id='', estimated_cost=0 WHERE id='task-1';
	`); err != nil {
		t.Fatalf("reset route fixture: %v", err)
	}
	rollback := command
	rollback.Decision.ID = "route-rollback"
	rollback.Event.EventID = "event-route-invalid"
	rollback.Event.Payload = json.RawMessage(`{`)
	rollback.Idempotency.Key = "route-command-rollback"
	if _, err := repo.CommitRouteTransition(projectCtx, rollback); !errors.Is(err, domain.ErrInvalidTaskEvent) {
		t.Fatalf("invalid route event error=%v want ErrInvalidTaskEvent", err)
	}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM route_decisions WHERE id='route-rollback'`).Scan(&routeCount); err != nil {
		t.Fatalf("count rolled-back route: %v", err)
	}
	if routeCount != 0 {
		t.Fatalf("rolled-back route decision count=%d", routeCount)
	}
	if err := db.QueryRowContext(ctx, `SELECT status, version, route_decision_id FROM tasks WHERE id='task-1'`).Scan(&status, &version, &routeID); err != nil {
		t.Fatalf("read task after rollback: %v", err)
	}
	if status != domain.TaskPlanning || version != 2 || routeID != "" {
		t.Fatalf("route rollback changed task status=%s version=%d route=%q", status, version, routeID)
	}
}
