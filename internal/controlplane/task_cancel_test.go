package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/identity"
	"github.com/tommyxie2026-tech/aicloud/internal/modelservice"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
	"github.com/tommyxie2026-tech/aicloud/internal/workflow"
)

type cancelTaskRepository struct {
	domain.TaskRepository
	task     domain.Task
	commands repository.TaskCommandStore
	getCalls int
}

func (r *cancelTaskRepository) Get(ctx context.Context, id string) (domain.Task, error) {
	r.getCalls++
	principal, err := identity.RequireProject(ctx)
	if err != nil {
		return domain.Task{}, err
	}
	if r.task.ID != id || r.task.TenantID != principal.TenantID || r.task.ProjectID != principal.ProjectID {
		return domain.Task{}, repository.ErrNotFound
	}
	return r.task, nil
}

func (r *cancelTaskRepository) TaskCommands() repository.TaskCommandStore { return r.commands }

type cancelCommandStore struct {
	repository.TaskCommandStore
	resolveRecord domain.IdempotencyRecord
	resolveFound  bool
	resolveErr    error
	resolveCalls  int
	commitResult  repository.TaskCommandCommitResult
	commitErr     error
	commitCalls   int
	commit        repository.TaskCommandCommit
}

func (s *cancelCommandStore) ResolveIdempotency(_ context.Context, _ repository.IdempotencyLookup) (domain.IdempotencyRecord, bool, error) {
	s.resolveCalls++
	return s.resolveRecord, s.resolveFound, s.resolveErr
}

func (s *cancelCommandStore) CommitTransition(_ context.Context, commit repository.TaskCommandCommit) (repository.TaskCommandCommitResult, error) {
	s.commitCalls++
	s.commit = commit
	return s.commitResult, s.commitErr
}

func TestCancelTaskIdempotentCommitsBusinessCancellationAndOutboxTogether(t *testing.T) {
	current := cancelTaskFixture(domain.TaskExecuting, 4)
	commands := &cancelCommandStore{}
	repo := &cancelTaskRepository{task: current, commands: commands}
	service := New(modelservice.New(repository.NewMemoryModels()), repo, workflow.NoopEngine{})

	committed := current
	at := time.Now().UTC()
	if _, err := committed.Transition(domain.TaskTransitionCommand{
		To: domain.TaskCancelled, Actor: "user-a", Cause: "operator requested", At: at,
	}); err != nil {
		t.Fatal(err)
	}
	committed.Version = 5
	commands.commitResult = repository.TaskCommandCommitResult{Task: committed}

	result, err := service.CancelTaskIdempotent(
		projectContext(),
		"task-a",
		4,
		"operator requested",
		CommandMetadata{IdempotencyKey: "cancel-a", RequestDigest: "transport-digest", RequestID: "request-a"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Task.Status != domain.TaskCancelled || result.Task.Version != 5 || result.Replayed {
		t.Fatalf("unexpected result: %+v", result)
	}
	if commands.resolveCalls != 1 || commands.commitCalls != 1 || repo.getCalls != 1 {
		t.Fatalf("resolve=%d get=%d commit=%d", commands.resolveCalls, repo.getCalls, commands.commitCalls)
	}
	commit := commands.commit
	if commit.ExpectedVersion != 4 || commit.Event.EventType != "TaskCancelled" || commit.Event.RequestID != "request-a" {
		t.Fatalf("unexpected cancellation commit: %+v", commit)
	}
	if len(commit.Outbox) != 1 || commit.Outbox[0].Destination != "workflow.cancel" {
		t.Fatalf("unexpected cancellation Outbox: %+v", commit.Outbox)
	}
	if commit.Outbox[0].IdempotencyKey != "workflow-cancel:tenant-a:task-a" {
		t.Fatalf("cancel delivery key=%q", commit.Outbox[0].IdempotencyKey)
	}
	if commit.Idempotency.Operation != taskCancelOperationV1 || commit.Idempotency.Key != "cancel-a" || len(commit.Idempotency.ResponsePayload) == 0 {
		t.Fatalf("unexpected command idempotency: %+v", commit.Idempotency)
	}
	var payload struct {
		TaskID  string `json:"taskId"`
		TraceID string `json:"traceId"`
		Reason  string `json:"reason"`
	}
	if err := json.Unmarshal(commit.Outbox[0].Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.TaskID != "task-a" || payload.TraceID != "trace-a" || payload.Reason != "operator requested" {
		t.Fatalf("unexpected cancellation payload: %+v", payload)
	}
}

func TestCancelTaskLostResponseRetryResolvesIdempotencyBeforeTaskRead(t *testing.T) {
	cancelled := cancelTaskFixture(domain.TaskCancelled, 5)
	responsePayload, _ := json.Marshal(cancelled)
	commands := &cancelCommandStore{
		resolveFound: true,
		resolveRecord: domain.IdempotencyRecord{
			TenantID: "tenant-a", ProjectID: "project-a", Operation: taskCancelOperationV1,
			Key: "cancel-a", ResponsePayload: responsePayload,
		},
	}
	repo := &cancelTaskRepository{task: cancelled, commands: commands}
	service := New(modelservice.New(repository.NewMemoryModels()), repo, workflow.NoopEngine{})

	result, err := service.CancelTaskIdempotent(
		projectContext(), "task-a", 4, "operator requested",
		CommandMetadata{IdempotencyKey: "cancel-a", RequestDigest: "transport-digest"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Replayed || result.Task.Status != domain.TaskCancelled || result.Task.Version != 5 {
		t.Fatalf("unexpected replay: %+v", result)
	}
	if repo.getCalls != 0 || commands.commitCalls != 0 {
		t.Fatalf("replay touched Task state: get=%d commit=%d", repo.getCalls, commands.commitCalls)
	}
}

func TestCancelTaskCannotRewriteOtherTerminalState(t *testing.T) {
	completed := cancelTaskFixture(domain.TaskCompleted, 5)
	commands := &cancelCommandStore{}
	repo := &cancelTaskRepository{task: completed, commands: commands}
	service := New(modelservice.New(repository.NewMemoryModels()), repo, workflow.NoopEngine{})

	_, err := service.CancelTaskIdempotent(
		projectContext(), "task-a", 5, "too late",
		CommandMetadata{IdempotencyKey: "cancel-late", RequestDigest: "transport-digest"},
	)
	if !errors.Is(err, domain.ErrTaskTerminal) {
		t.Fatalf("error=%v", err)
	}
	if commands.commitCalls != 0 {
		t.Fatalf("completed Task was rewritten, commits=%d", commands.commitCalls)
	}
}

func TestCancelTaskIdempotencyConflictDoesNotReadOrMutateTask(t *testing.T) {
	commands := &cancelCommandStore{resolveErr: repository.ErrIdempotencyConflict}
	repo := &cancelTaskRepository{task: cancelTaskFixture(domain.TaskExecuting, 4), commands: commands}
	service := New(modelservice.New(repository.NewMemoryModels()), repo, workflow.NoopEngine{})

	_, err := service.CancelTaskIdempotent(
		projectContext(), "task-a", 4, "operator requested",
		CommandMetadata{IdempotencyKey: "cancel-a", RequestDigest: "transport-digest"},
	)
	if !errors.Is(err, repository.ErrIdempotencyConflict) {
		t.Fatalf("error=%v", err)
	}
	if repo.getCalls != 0 || commands.commitCalls != 0 {
		t.Fatalf("conflict touched Task state: get=%d commit=%d", repo.getCalls, commands.commitCalls)
	}
}

func projectContext() context.Context {
	return identity.WithPrincipal(context.Background(), identity.Principal{
		Type: identity.PrincipalUser, SubjectID: "user-a", TenantID: "tenant-a", ProjectID: "project-a",
		AuthnMethod: "test", Issuer: "test",
	})
}

func cancelTaskFixture(status domain.TaskStatus, version int64) domain.Task {
	now := time.Now().UTC()
	completedAt := (*time.Time)(nil)
	if status == domain.TaskCompleted || status == domain.TaskFailed || status == domain.TaskCancelled || status == domain.TaskExpired {
		completedAt = &now
	}
	return domain.Task{
		ID: "task-a", TenantID: "tenant-a", ProjectID: "project-a", CreatedBy: "user-a",
		AgentID: "agent-a", Input: "cancel me", Status: status, Version: version, TraceID: "trace-a",
		Currency: "USD", CreatedAt: now, UpdatedAt: now, CompletedAt: completedAt,
	}
}
