package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidTaskTransition = errors.New("invalid task transition")
	ErrTaskTerminal          = errors.New("task is terminal")
	ErrTransitionActor       = errors.New("task transition actor is required")
	ErrTransitionCause       = errors.New("task transition cause is required")
	ErrTransitionTime        = errors.New("task transition time is required")
)

type TaskTransitionCommand struct {
	To    TaskStatus `json:"to"`
	Actor string     `json:"actor"`
	Cause string     `json:"cause"`
	At    time.Time  `json:"at"`
}

type TaskTransition struct {
	From  TaskStatus `json:"from"`
	To    TaskStatus `json:"to"`
	Actor string     `json:"actor"`
	Cause string     `json:"cause"`
	At    time.Time  `json:"at"`
}

type NewTaskParams struct {
	ID       string
	AgentID  string
	Input    string
	TraceID  string
	Currency string
	Now      time.Time
}

func NewTask(params NewTaskParams) (Task, error) {
	if strings.TrimSpace(params.ID) == "" {
		return Task{}, fmt.Errorf("task id is required")
	}
	if strings.TrimSpace(params.Input) == "" {
		return Task{}, fmt.Errorf("task input is required")
	}
	if params.Now.IsZero() {
		return Task{}, fmt.Errorf("task creation time is required")
	}
	currency := strings.TrimSpace(params.Currency)
	if currency == "" {
		currency = "USD"
	}
	return Task{
		ID:        strings.TrimSpace(params.ID),
		AgentID:   strings.TrimSpace(params.AgentID),
		Input:     params.Input,
		Status:    TaskCreated,
		Version:   1,
		Currency:  currency,
		TraceID:   strings.TrimSpace(params.TraceID),
		CreatedAt: params.Now,
		UpdatedAt: params.Now,
	}, nil
}

func (t Task) IsTerminal() bool {
	switch t.Status {
	case TaskCompleted, TaskFailed, TaskCancelled, TaskExpired:
		return true
	default:
		return false
	}
}

func (t Task) CanTransition(to TaskStatus) bool {
	if t.IsTerminal() {
		return false
	}
	if to == TaskFailed || to == TaskCancelled || to == TaskExpired {
		return isKnownNonTerminal(t.Status)
	}
	switch t.Status {
	case TaskCreated:
		return to == TaskPlanning
	case TaskPlanning:
		return to == TaskRouting
	case TaskRouting:
		return to == TaskExecuting
	case TaskExecuting:
		return to == TaskWaitingApproval || to == TaskValidating
	case TaskWaitingApproval:
		return to == TaskExecuting
	case TaskValidating:
		return to == TaskCompleted
	default:
		return false
	}
}

func (t *Task) Transition(command TaskTransitionCommand) (TaskTransition, error) {
	if t == nil {
		return TaskTransition{}, fmt.Errorf("%w: nil task", ErrInvalidTaskTransition)
	}
	actor := strings.TrimSpace(command.Actor)
	cause := strings.TrimSpace(command.Cause)
	if actor == "" {
		return TaskTransition{}, ErrTransitionActor
	}
	if cause == "" {
		return TaskTransition{}, ErrTransitionCause
	}
	if command.At.IsZero() {
		return TaskTransition{}, ErrTransitionTime
	}
	if t.IsTerminal() {
		return TaskTransition{}, fmt.Errorf("%w: %s", ErrTaskTerminal, t.Status)
	}
	if !t.CanTransition(command.To) {
		return TaskTransition{}, fmt.Errorf("%w: %s -> %s", ErrInvalidTaskTransition, t.Status, command.To)
	}

	transition := TaskTransition{
		From: t.Status, To: command.To, Actor: actor, Cause: cause, At: command.At,
	}
	t.Status = command.To
	t.UpdatedAt = command.At
	if t.IsTerminal() {
		completedAt := command.At
		t.CompletedAt = &completedAt
	}
	return transition, nil
}

func isKnownNonTerminal(status TaskStatus) bool {
	switch status {
	case TaskCreated, TaskPlanning, TaskRouting, TaskExecuting, TaskWaitingApproval, TaskValidating:
		return true
	default:
		return false
	}
}
