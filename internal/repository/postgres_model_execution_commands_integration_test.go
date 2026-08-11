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

func TestScopedPostgresModelExecutionRetryAndSuccessFinalization(t *testing.T) {
	db, ctx, projectCtx, repo := modelExecutionFixture(t)
	defer db.Close()
	defer cleanupTaskCommandFixture(t, context.Background(), db)

	now := time.Now().UTC()
	routingTask := routedFixtureTask(t, ctx, db, now)
	executingTask := routingTask
	started, err := executingTask.Transition(domain.TaskTransitionCommand{
		To: domain.TaskExecuting, Actor: "user:user-a", Cause: "execute selected model", At: now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("transition to executing: %v", err)
	}
	begin := ModelExecutionBeginCommit{
		Task:       executingTask,
		Transition: &started,
		Event: &domain.TaskEvent{
			EventID: "event-execution-start", EventType: "TaskExecutionStarted",
			Actor: domain.TaskEventActor{PrincipalType: "user", SubjectID: "user-a"},
			Payload: json.RawMessage(`{"operationId":"model-op-1"}`), SchemaVersion: 1,
		},
		Idempotency: modelExecutionIdempotency(now, domain.IdempotencyInProgress),
	}
	first, err := repo.BeginModelExecution(projectCtx, begin)
	if err != nil {
		t.Fatalf("BeginModelExecution: %v", err)
	}
	if first.Task.Status != domain.TaskExecuting || first.Task.Version != 4 || first.Event == nil || first.Event.Sequence != 1 {
		t.Fatalf("unexpected first execution begin: %#v", first)
	}

	if _, err := repo.BeginModelExecution(projectCtx, begin); !errors.Is(err, ErrIdempotencyInProgress) {
		t.Fatalf("concurrent duplicate error=%v want ErrIdempotencyInProgress", err)
	}

	retryable := modelExecutionIdempotency(now, domain.IdempotencyFailedRetryable)
	retryable.ResourceID = first.Task.ID
	retryable.ResponseCode = 503
	retryable.ResponsePayload = json.RawMessage(`{"error":"provider unavailable"}`)
	if err := repo.MarkModelExecutionRetryable(projectCtx, retryable); err != nil {
		t.Fatalf("MarkModelExecutionRetryable: %v", err)
	}

	retry := ModelExecutionBeginCommit{
		Task:        first.Task,
		Idempotency: modelExecutionIdempotency(now, domain.IdempotencyInProgress),
	}
	reacquired, err := repo.BeginModelExecution(projectCtx, retry)
	if err != nil {
		t.Fatalf("reacquire model execution: %v", err)
	}
	if reacquired.Task.Version != 4 || reacquired.Event != nil {
		t.Fatalf("retry appended duplicate transition evidence: %#v", reacquired)
	}
	var startEvents int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM task_events WHERE task_id='task-1' AND event_type='TaskExecutionStarted'`).Scan(&startEvents); err != nil {
		t.Fatalf("count execution-start events: %v", err)
	}
	if startEvents != 1 {
		t.Fatalf("execution-start event count=%d want=1", startEvents)
	}

	finalTask := reacquired.Task
	validating, err := finalTask.Transition(domain.TaskTransitionCommand{
		To: domain.TaskValidating, Actor: "user:user-a", Cause: "validate model result", At: now.Add(2 * time.Second),
	})
	if err != nil {
		t.Fatalf("transition to validating: %v", err)
	}
	completed, err := finalTask.Transition(domain.TaskTransitionCommand{
		To: domain.TaskCompleted, Actor: "user:user-a", Cause: "model result accepted", At: now.Add(3 * time.Second),
	})
	if err != nil {
		t.Fatalf("transition to completed: %v", err)
	}
	finalTask.Result = `{"ok":true}`
	completedIdempotency := modelExecutionIdempotency(now, domain.IdempotencyCompleted)
	completedIdempotency.ResourceID = finalTask.ID
	completedIdempotency.ResponseCode = 200
	completedIdempotency.ResponsePayload = json.RawMessage(`{"fallback":false}`)
	finalized, err := repo.FinalizeModelExecution(projectCtx, ModelExecutionFinalizeCommit{
		Task:        finalTask,
		Transitions: []domain.TaskTransition{validating, completed},
		Events: []domain.TaskEvent{
			{
				EventID: "event-validation", EventType: "TaskValidationStarted",
				Actor: domain.TaskEventActor{PrincipalType: "user", SubjectID: "user-a"},
				Payload: json.RawMessage(`{"operationId":"model-op-1"}`), SchemaVersion: 1,
			},
			{
				EventID: "event-completed", EventType: "TaskCompleted",
				Actor: domain.TaskEventActor{PrincipalType: "user", SubjectID: "user-a"},
				Payload: json.RawMessage(`{"operationId":"model-op-1"}`), SchemaVersion: 1,
			},
		},
		Idempotency: completedIdempotency,
	})
	if err != nil {
		t.Fatalf("FinalizeModelExecution success: %v", err)
	}
	if finalized.Task.Status != domain.TaskCompleted || finalized.Task.Version != 6 || len(finalized.Events) != 2 {
		t.Fatalf("unexpected successful finalization: %#v", finalized)
	}
	if finalized.Events[0].Sequence != 2 || finalized.Events[1].Sequence != 3 {
		t.Fatalf("unexpected final event sequence: %#v", finalized.Events)
	}
	record, found, err := repo.ResolveIdempotency(projectCtx, IdempotencyLookup{
		TenantID: "tenant-a", ProjectID: "project-a", Operation: "POST /api/v1/tasks/task-1/model",
		Key: "model-command-1", RequestDigest: "sha256:model-request",
	})
	if err != nil || !found || record.Status != domain.IdempotencyCompleted {
		t.Fatalf("completed idempotency record=%#v found=%t err=%v", record, found, err)
	}
}

func TestScopedPostgresModelExecutionFinalFailure(t *testing.T) {
	db, ctx, projectCtx, repo := modelExecutionFixture(t)
	defer db.Close()
	defer cleanupTaskCommandFixture(t, context.Background(), db)

	now := time.Now().UTC()
	routingTask := routedFixtureTask(t, ctx, db, now)
	executingTask := routingTask
	started, err := executingTask.Transition(domain.TaskTransitionCommand{
		To: domain.TaskExecuting, Actor: "user:user-a", Cause: "execute selected model", At: now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("transition to executing: %v", err)
	}
	begun, err := repo.BeginModelExecution(projectCtx, ModelExecutionBeginCommit{
		Task:       executingTask,
		Transition: &started,
		Event: &domain.TaskEvent{
			EventID: "event-execution-start", EventType: "TaskExecutionStarted",
			Actor: domain.TaskEventActor{PrincipalType: "user", SubjectID: "user-a"},
			Payload: json.RawMessage(`{"operationId":"model-op-fail"}`), SchemaVersion: 1,
		},
		Idempotency: modelExecutionIdempotency(now, domain.IdempotencyInProgress),
	})
	if err != nil {
		t.Fatalf("begin failed execution: %v", err)
	}

	failedTask := begun.Task
	failedTask.Result = "schema mismatch"
	failed, err := failedTask.Transition(domain.TaskTransitionCommand{
		To: domain.TaskFailed, Actor: "user:user-a", Cause: "model execution failed", At: now.Add(2 * time.Second),
	})
	if err != nil {
		t.Fatalf("transition to failed: %v", err)
	}
	failedIdempotency := modelExecutionIdempotency(now, domain.IdempotencyFailedFinal)
	failedIdempotency.ResourceID = failedTask.ID
	failedIdempotency.ResponseCode = 500
	failedIdempotency.ResponsePayload = json.RawMessage(`{"error":"schema mismatch"}`)
	result, err := repo.FinalizeModelExecution(projectCtx, ModelExecutionFinalizeCommit{
		Task:        failedTask,
		Transitions: []domain.TaskTransition{failed},
		Events: []domain.TaskEvent{{
			EventID: "event-failed", EventType: "TaskFailed",
			Actor: domain.TaskEventActor{PrincipalType: "user", SubjectID: "user-a"},
			Payload: json.RawMessage(`{"error":"schema mismatch"}`), SchemaVersion: 1,
		}},
		Idempotency: failedIdempotency,
	})
	if err != nil {
		t.Fatalf("FinalizeModelExecution failure: %v", err)
	}
	if result.Task.Status != domain.TaskFailed || result.Task.Version != 5 || len(result.Events) != 1 || result.Events[0].Sequence != 2 {
		t.Fatalf("unexpected failed finalization: %#v", result)
	}
}

func modelExecutionFixture(t *testing.T) (*sql.DB, context.Context, context.Context, *ScopedPostgresTaskCommands) {
	t.Helper()
	dsn := os.Getenv("AICLOUD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("AICLOUD_TEST_DATABASE_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open PostgreSQL: %v", err)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		t.Fatalf("ping PostgreSQL: %v", err)
	}
	cleanupTaskCommandFixture(t, ctx, db)
	createTaskCommandFixture(t, ctx, db)
	principal := identity.Principal{
		Type: identity.PrincipalUser, SubjectID: "user-a", TenantID: "tenant-a", ProjectID: "project-a",
		AuthnMethod: "integration-test", Issuer: "test",
	}
	return db, ctx, identity.WithPrincipal(ctx, principal), NewScopedPostgresTaskCommands(db)
}

func routedFixtureTask(t *testing.T, ctx context.Context, db *sql.DB, now time.Time) domain.Task {
	t.Helper()
	if _, err := db.ExecContext(ctx, `UPDATE tasks SET status='ROUTING', version=3, updated_at=$1 WHERE id='task-1'`, now); err != nil {
		t.Fatalf("prepare routed task fixture: %v", err)
	}
	task := fixtureTask(now)
	task.Status = domain.TaskRouting
	task.Version = 3
	task.UpdatedAt = now
	return task
}

func modelExecutionIdempotency(now time.Time, status domain.IdempotencyStatus) domain.IdempotencyRecord {
	return domain.IdempotencyRecord{
		TenantID: "tenant-a", ProjectID: "project-a", Operation: "POST /api/v1/tasks/task-1/model",
		Key: "model-command-1", RequestDigest: "sha256:model-request", Status: status,
		CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour),
	}
}
