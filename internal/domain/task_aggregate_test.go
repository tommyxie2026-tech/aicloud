package domain

import (
	"errors"
	"testing"
	"time"
)

func TestNewTaskStartsCreatedAtVersionOne(t *testing.T) {
	now := time.Date(2026, 8, 10, 9, 0, 0, 0, time.UTC)
	task, err := NewTask(NewTaskParams{
		ID: "task-1", AgentID: "agent-1", Input: "inspect", TraceID: "trace-1", Now: now,
	})
	if err != nil {
		t.Fatalf("NewTask returned error: %v", err)
	}
	if task.Status != TaskCreated || task.Version != 1 {
		t.Fatalf("unexpected initial aggregate state: status=%s version=%d", task.Status, task.Version)
	}
	if task.Currency != "USD" || !task.CreatedAt.Equal(now) || !task.UpdatedAt.Equal(now) {
		t.Fatalf("unexpected initial task: %#v", task)
	}
}

func TestTaskTransitionHappyPath(t *testing.T) {
	at := time.Date(2026, 8, 10, 9, 0, 0, 0, time.UTC)
	task, err := NewTask(NewTaskParams{ID: "task-1", Input: "run", Now: at})
	if err != nil {
		t.Fatal(err)
	}

	path := []TaskStatus{TaskPlanning, TaskRouting, TaskExecuting, TaskValidating, TaskCompleted}
	for i, target := range path {
		at = at.Add(time.Second)
		transition, err := task.Transition(TaskTransitionCommand{
			To: target, Actor: "controlplane", Cause: "test", At: at,
		})
		if err != nil {
			t.Fatalf("transition %d to %s: %v", i, target, err)
		}
		if transition.To != target || task.Status != target {
			t.Fatalf("transition mismatch: %#v task=%#v", transition, task)
		}
	}
	if !task.IsTerminal() || task.CompletedAt == nil || !task.CompletedAt.Equal(at) {
		t.Fatalf("terminal completion evidence not set: %#v", task)
	}
}

func TestTaskTransitionApprovalLoop(t *testing.T) {
	at := time.Now().UTC()
	task := Task{Status: TaskExecuting, UpdatedAt: at}
	for _, target := range []TaskStatus{TaskWaitingApproval, TaskExecuting, TaskValidating} {
		at = at.Add(time.Second)
		if _, err := task.Transition(TaskTransitionCommand{To: target, Actor: "worker", Cause: "approval flow", At: at}); err != nil {
			t.Fatalf("transition to %s failed: %v", target, err)
		}
	}
}

func TestTaskTransitionRejectsStateSkipping(t *testing.T) {
	task := Task{Status: TaskCreated}
	_, err := task.Transition(TaskTransitionCommand{
		To: TaskExecuting, Actor: "controlplane", Cause: "skip planning", At: time.Now().UTC(),
	})
	if !errors.Is(err, ErrInvalidTaskTransition) {
		t.Fatalf("error=%v want ErrInvalidTaskTransition", err)
	}
	if task.Status != TaskCreated {
		t.Fatalf("invalid transition mutated task: %s", task.Status)
	}
}

func TestTaskTerminalStateCannotReopen(t *testing.T) {
	task := Task{Status: TaskCompleted}
	_, err := task.Transition(TaskTransitionCommand{
		To: TaskExecuting, Actor: "worker", Cause: "retry", At: time.Now().UTC(),
	})
	if !errors.Is(err, ErrTaskTerminal) {
		t.Fatalf("error=%v want ErrTaskTerminal", err)
	}
}

func TestAnyKnownNonTerminalCanFailCancelOrExpire(t *testing.T) {
	for _, source := range []TaskStatus{
		TaskCreated, TaskPlanning, TaskRouting, TaskExecuting, TaskWaitingApproval, TaskValidating,
	} {
		for _, target := range []TaskStatus{TaskFailed, TaskCancelled, TaskExpired} {
			task := Task{Status: source}
			if _, err := task.Transition(TaskTransitionCommand{
				To: target, Actor: "worker", Cause: "terminal command", At: time.Now().UTC(),
			}); err != nil {
				t.Fatalf("%s -> %s failed: %v", source, target, err)
			}
			if !task.IsTerminal() {
				t.Fatalf("target %s should be terminal", target)
			}
		}
	}
}

func TestTaskTransitionRequiresEvidenceMetadata(t *testing.T) {
	task := Task{Status: TaskCreated}
	now := time.Now().UTC()
	if _, err := task.Transition(TaskTransitionCommand{To: TaskPlanning, Cause: "plan", At: now}); !errors.Is(err, ErrTransitionActor) {
		t.Fatalf("missing actor error=%v", err)
	}
	if _, err := task.Transition(TaskTransitionCommand{To: TaskPlanning, Actor: "worker", At: now}); !errors.Is(err, ErrTransitionCause) {
		t.Fatalf("missing cause error=%v", err)
	}
	if _, err := task.Transition(TaskTransitionCommand{To: TaskPlanning, Actor: "worker", Cause: "plan"}); !errors.Is(err, ErrTransitionTime) {
		t.Fatalf("missing time error=%v", err)
	}
}
