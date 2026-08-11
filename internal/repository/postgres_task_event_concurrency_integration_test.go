//go:build integration

package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/identity"
)

func TestTaskEventSequenceRemainsContiguousUnderConcurrentCommands(t *testing.T) {
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
	defer cleanupTaskCommandFixture(t, context.Background(), db)
	createTaskCommandFixture(t, ctx, db)

	principal := identity.Principal{
		Type: identity.PrincipalUser, SubjectID: "user-a", TenantID: "tenant-a", ProjectID: "project-a",
		AuthnMethod: "integration-test", Issuer: "test",
	}
	projectCtx := identity.WithPrincipal(ctx, principal)
	repo := NewScopedPostgresTaskCommands(db)
	now := time.Now().UTC()
	task := fixtureTask(now)

	for index, target := range []domain.TaskStatus{domain.TaskPlanning, domain.TaskRouting, domain.TaskExecuting} {
		task = commitStressTransition(t, projectCtx, repo, task, target, fmt.Sprintf("seed-%d", index), now.Add(time.Duration(index+1)*time.Millisecond))
	}

	const rounds = 12
	const contenders = 8
	for round := 0; round < rounds; round++ {
		target := domain.TaskWaitingApproval
		if task.Status == domain.TaskWaitingApproval {
			target = domain.TaskExecuting
		}
		base := task
		type outcome struct {
			task domain.Task
			err  error
		}
		outcomes := make(chan outcome, contenders)
		var wg sync.WaitGroup
		for contender := 0; contender < contenders; contender++ {
			wg.Add(1)
			go func(contender int) {
				defer wg.Done()
				candidate := base
				at := now.Add(time.Duration(100+round)*time.Millisecond)
				transition, transitionErr := candidate.Transition(domain.TaskTransitionCommand{
					To: target, Actor: "user:user-a",
					Cause: fmt.Sprintf("concurrency round %d", round), At: at,
				})
				if transitionErr != nil {
					outcomes <- outcome{err: transitionErr}
					return
				}
				eventType, _ := domain.CanonicalTaskStateEvent(target)
				payload, _ := json.Marshal(map[string]any{"round": round, "contender": contender})
				result, commitErr := repo.CommitTransition(projectCtx, TaskCommandCommit{
					Task: candidate,
					Transition: transition,
					Event: domain.TaskEvent{
						EventID: fmt.Sprintf("event-round-%d-%d", round, contender), EventType: eventType,
						Actor: domain.TaskEventActor{PrincipalType: "user", SubjectID: "user-a"},
						Payload: payload, SchemaVersion: 1,
					},
					Idempotency: domain.IdempotencyRecord{
						TenantID: "tenant-a", ProjectID: "project-a", Operation: fmt.Sprintf("stress:%d", round),
						Key: fmt.Sprintf("command-%d-%d", round, contender), RequestDigest: fmt.Sprintf("sha256:%d", round),
						Status: domain.IdempotencyCompleted, CreatedAt: at, ExpiresAt: at.Add(time.Hour),
					},
				})
				outcomes <- outcome{task: result.Task, err: commitErr}
			}(contender)
		}
		wg.Wait()
		close(outcomes)

		successes := 0
		conflicts := 0
		for item := range outcomes {
			switch {
			case item.err == nil:
				successes++
				task = item.task
			case errors.Is(item.err, ErrVersionConflict):
				conflicts++
			default:
				t.Fatalf("round %d unexpected concurrent command error: %v", round, item.err)
			}
		}
		if successes != 1 || conflicts != contenders-1 {
			t.Fatalf("round %d successes=%d conflicts=%d", round, successes, conflicts)
		}
	}

	rows, err := db.QueryContext(ctx, `SELECT sequence FROM task_events WHERE task_id='task-1' ORDER BY sequence`)
	if err != nil {
		t.Fatalf("list TaskEvent sequences: %v", err)
	}
	defer rows.Close()
	sequence := int64(1)
	count := 0
	for rows.Next() {
		var got int64
		if err := rows.Scan(&got); err != nil {
			t.Fatalf("scan TaskEvent sequence: %v", err)
		}
		if got != sequence {
			t.Fatalf("TaskEvent sequence gap/duplicate at position %d: got=%d want=%d", count, got, sequence)
		}
		sequence++
		count++
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate TaskEvent sequences: %v", err)
	}
	if count != 3+rounds {
		t.Fatalf("TaskEvent count=%d want=%d", count, 3+rounds)
	}
}

func commitStressTransition(t *testing.T, ctx context.Context, repo *ScopedPostgresTaskCommands, task domain.Task, target domain.TaskStatus, key string, at time.Time) domain.Task {
	t.Helper()
	transition, err := task.Transition(domain.TaskTransitionCommand{
		To: target, Actor: "user:user-a", Cause: key, At: at,
	})
	if err != nil {
		t.Fatalf("prepare %s transition: %v", target, err)
	}
	eventType, ok := domain.CanonicalTaskStateEvent(target)
	if !ok {
		t.Fatalf("missing canonical event for %s", target)
	}
	payload, _ := json.Marshal(map[string]string{"target": string(target)})
	result, err := repo.CommitTransition(ctx, TaskCommandCommit{
		Task: task, Transition: transition,
		Event: domain.TaskEvent{
			EventID: "event-" + key, EventType: eventType,
			Actor: domain.TaskEventActor{PrincipalType: "user", SubjectID: "user-a"},
			Payload: payload, SchemaVersion: 1,
		},
		Idempotency: domain.IdempotencyRecord{
			TenantID: "tenant-a", ProjectID: "project-a", Operation: "stress:seed", Key: key,
			RequestDigest: "sha256:" + key, Status: domain.IdempotencyCompleted,
			CreatedAt: at, ExpiresAt: at.Add(time.Hour),
		},
	})
	if err != nil {
		t.Fatalf("commit %s transition: %v", target, err)
	}
	return result.Task
}
