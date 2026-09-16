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

	execution "github.com/tommyxie2026-tech/aicloud/execution"
	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/identity"
)

func TestScopedPostgresRouteSelectionAtomicCommitReplayAndRollback(t *testing.T) {
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
			Actor: "user:user-a", Cause: "planning", At: planningTask.UpdatedAt,
		},
		Event: domain.TaskEvent{
			EventID: "event-select-plan", EventType: "TaskPlanningStarted",
			Actor:   domain.TaskEventActor{PrincipalType: "user", SubjectID: "user-a"},
			Payload: json.RawMessage(`{"to":"PLANNING"}`), SchemaVersion: 1,
		},
		Idempotency: domain.IdempotencyRecord{
			TenantID: "tenant-a", ProjectID: "project-a", Operation: "select:planning", Key: "select-plan-1",
			RequestDigest: "sha256:select-plan", Status: domain.IdempotencyCompleted,
			CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour),
		},
	})
	if err != nil {
		t.Fatalf("commit planning transition: %v", err)
	}

	routingTask := planning.Task
	routingTask.Status = domain.TaskRouting
	routingTask.UpdatedAt = now.Add(2 * time.Second)
	routing, err := repo.CommitTransition(projectCtx, TaskCommandCommit{
		Task: routingTask,
		Transition: domain.TaskTransition{
			From: domain.TaskPlanning, To: domain.TaskRouting,
			Actor: "system:temporal", Cause: "planning complete", At: routingTask.UpdatedAt,
		},
		Event: domain.TaskEvent{
			EventID: "event-select-routing", EventType: "TaskRoutingStarted",
			Actor:   domain.TaskEventActor{PrincipalType: "system", SubjectID: "temporal"},
			Payload: json.RawMessage(`{"to":"ROUTING"}`), SchemaVersion: 1,
		},
		Idempotency: domain.IdempotencyRecord{
			TenantID: "tenant-a", ProjectID: "project-a", Operation: "select:routing", Key: "select-routing-1",
			RequestDigest: "sha256:select-routing", Status: domain.IdempotencyCompleted,
			CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour),
		},
	})
	if err != nil {
		t.Fatalf("commit routing transition: %v", err)
	}

	decision := domain.RouteDecision{
		ID: "route-selected-1", TaskID: "task-1",
		Selected: domain.RouteCandidate{
			ModelID: "model-a", ModelVersion: "v1", DeploymentID: "dep-a",
			RouteClass: domain.RouteEfficient, EstimatedCost: 0.02,
		},
		Candidates: []domain.RouteCandidate{{ModelID: "model-a", ModelVersion: "v1", DeploymentID: "dep-a"}},
		Reason:     "best policy-approved deployment", EvidenceVersion: "evidence-v2", PolicyVersion: "policy-v2",
		CreatedAt: now.Add(3 * time.Second),
	}
	target := execution.ExecutionTarget{
		ID: "deployment/dep-a", Revision: 1, Type: execution.TargetModel,
		Snapshot: execution.TargetSnapshot{
			Digest: "sha256:target-a", ModelVersion: "v1", DeploymentRef: "dep-a",
		},
	}
	payload, err := json.Marshal(map[string]any{
		"routeDecisionId": decision.ID,
		"target":          target,
	})
	if err != nil {
		t.Fatal(err)
	}
	command := RouteSelectionCommit{
		Task:     routing.Task,
		Decision: decision,
		Target:   target,
		Event: domain.TaskEvent{
			EventID: "event-route-selected", EventType: "TaskRouteSelected",
			Actor:   domain.TaskEventActor{PrincipalType: "system", SubjectID: "temporal-task-lifecycle"},
			Payload: payload, SchemaVersion: 1,
		},
		Idempotency: domain.IdempotencyRecord{
			TenantID: "tenant-a", ProjectID: "project-a", Operation: "workflow:route-select", Key: "task-1:route:v1",
			RequestDigest: "sha256:route-selection", Status: domain.IdempotencyCompleted,
			CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour),
		},
	}

	result, err := repo.CommitRouteSelection(projectCtx, command)
	if err != nil {
		t.Fatalf("CommitRouteSelection: %v", err)
	}
	if result.Replayed || result.Task.Status != domain.TaskRouting || result.Task.RouteDecisionID != decision.ID {
		t.Fatalf("unexpected route selection result: %#v", result)
	}
	if result.Target.Snapshot.Digest != target.Snapshot.Digest {
		t.Fatalf("frozen target digest=%q want=%q", result.Target.Snapshot.Digest, target.Snapshot.Digest)
	}

	replay, err := repo.CommitRouteSelection(projectCtx, command)
	if err != nil {
		t.Fatalf("route selection replay: %v", err)
	}
	if !replay.Replayed || replay.Decision.ID != decision.ID || replay.Target.Snapshot.Digest != target.Snapshot.Digest {
		t.Fatalf("unexpected replay: %#v", replay)
	}

	conflict := command
	conflict.Idempotency.RequestDigest = "sha256:changed"
	if _, err := repo.CommitRouteSelection(projectCtx, conflict); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("changed route selection request error=%v want ErrIdempotencyConflict", err)
	}

	var eventPayload []byte
	if err := db.QueryRowContext(ctx, `SELECT payload FROM task_events WHERE event_id='event-route-selected'`).Scan(&eventPayload); err != nil {
		t.Fatalf("read route selected event: %v", err)
	}
	if !json.Valid(eventPayload) || !containsJSONText(eventPayload, "sha256:target-a") {
		t.Fatalf("route event must preserve frozen target evidence: %s", eventPayload)
	}

	var status domain.TaskStatus
	var version int64
	var routeID string
	if err := db.QueryRowContext(ctx, `SELECT status, version, route_decision_id FROM tasks WHERE id='task-1'`).Scan(&status, &version, &routeID); err != nil {
		t.Fatalf("read selected route task: %v", err)
	}
	if status != domain.TaskRouting || routeID != decision.ID || version != routing.Task.Version+1 {
		t.Fatalf("route selection changed invalid task state status=%s version=%d route=%s", status, version, routeID)
	}

	rollback := command
	rollback.Task.Version = result.Task.Version
	rollback.Decision.ID = "route-selected-rollback"
	rollback.Event.EventID = "event-route-selected-invalid"
	rollback.Event.Payload = json.RawMessage(`{`)
	rollback.Idempotency.Key = "task-1:route:rollback"
	rollback.Idempotency.RequestDigest = "sha256:rollback"
	if _, err := repo.CommitRouteSelection(projectCtx, rollback); !errors.Is(err, domain.ErrInvalidTaskEvent) {
		t.Fatalf("invalid route selection event error=%v want ErrInvalidTaskEvent", err)
	}
	var rollbackCount int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM route_decisions WHERE id='route-selected-rollback'`).Scan(&rollbackCount); err != nil {
		t.Fatalf("count rollback decision: %v", err)
	}
	if rollbackCount != 0 {
		t.Fatalf("invalid event must roll back route decision, count=%d", rollbackCount)
	}
}

func containsJSONText(body []byte, text string) bool {
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return false
	}
	encoded, err := json.Marshal(value)
	return err == nil && string(encoded) != "" && bytesContains(encoded, []byte(text))
}

func bytesContains(body, needle []byte) bool {
	if len(needle) == 0 {
		return true
	}
	for i := 0; i+len(needle) <= len(body); i++ {
		matched := true
		for j := range needle {
			if body[i+j] != needle[j] {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}
