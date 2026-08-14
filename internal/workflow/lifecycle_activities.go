package workflow

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"go.temporal.io/sdk/temporal"
)

var (
	ErrLifecycleTaskNotFound = errors.New("lifecycle Task not found")
	ErrLifecycleScope        = errors.New("lifecycle Task scope mismatch")
	ErrOperationKeyConflict  = errors.New("lifecycle operation key conflict")
)

type transitionRecord struct {
	To       domain.TaskStatus
	Snapshot TaskSnapshot
}

// MemoryLifecycleActivities is a test-only S3C business-state seam. It models
// Task aggregate transitions, optimistic versions and stable operation-key
// idempotency without any database, provider, Tool or infrastructure effect.
type MemoryLifecycleActivities struct {
	mu              sync.Mutex
	tasks           map[string]domain.Task
	operations      map[string]transitionRecord
	stepCalls       map[string]int
	stepFailures    map[string]int
	terminalOnStep  map[string]domain.TaskStatus
	staleOnceForKey map[string]bool
}

func NewMemoryLifecycleActivities(tasks ...domain.Task) *MemoryLifecycleActivities {
	result := &MemoryLifecycleActivities{
		tasks:           make(map[string]domain.Task, len(tasks)),
		operations:      make(map[string]transitionRecord),
		stepCalls:       make(map[string]int),
		stepFailures:    make(map[string]int),
		terminalOnStep:  make(map[string]domain.TaskStatus),
		staleOnceForKey: make(map[string]bool),
	}
	for _, task := range tasks {
		result.tasks[task.ID] = task
	}
	return result
}

func (a *MemoryLifecycleActivities) LoadTask(_ context.Context, input LoadTaskInput) (TaskSnapshot, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	task, err := a.taskForScope(input.TenantID, input.ProjectID, input.TaskID, input.TraceID)
	if err != nil {
		return TaskSnapshot{}, err
	}
	return snapshotFromTask(task), nil
}

func (a *MemoryLifecycleActivities) TransitionTask(_ context.Context, input TransitionTaskInput) (TaskSnapshot, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if strings.TrimSpace(input.OperationKey) == "" {
		return TaskSnapshot{}, fmt.Errorf("operation key is required")
	}

	// Scope is re-established before both first-application and idempotent replay.
	// This deliberately models the S3D requirement that an idempotency hit can
	// never become a Tenant/Project authorization bypass.
	task, err := a.taskForScope(input.TenantID, input.ProjectID, input.TaskID, input.TraceID)
	if err != nil {
		return TaskSnapshot{}, err
	}
	if record, ok := a.operations[input.OperationKey]; ok {
		if record.To != input.To {
			return TaskSnapshot{}, fmt.Errorf("%w: %s was recorded for %s, requested %s", ErrOperationKeyConflict, input.OperationKey, record.To, input.To)
		}
		return record.Snapshot, nil
	}

	if task.IsTerminal() {
		return snapshotFromTask(task), nil
	}
	if a.staleOnceForKey[input.OperationKey] {
		delete(a.staleOnceForKey, input.OperationKey)
		return TaskSnapshot{}, temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("Task version is stale: expected %d, actual %d", input.ExpectedVersion, task.Version),
			ErrorTypeStaleTaskVersion,
			nil,
		)
	}
	if task.Version != input.ExpectedVersion {
		return TaskSnapshot{}, temporal.NewNonRetryableApplicationError(
			fmt.Sprintf("Task version is stale: expected %d, actual %d", input.ExpectedVersion, task.Version),
			ErrorTypeStaleTaskVersion,
			nil,
		)
	}

	at := time.Now().UTC()
	if _, err := task.Transition(domain.TaskTransitionCommand{
		To:    input.To,
		Actor: "workflow:" + TaskExecutionWorkflowType,
		Cause: input.Cause,
		At:    at,
	}); err != nil {
		return TaskSnapshot{}, err
	}
	task.Version++
	a.tasks[task.ID] = task
	snapshot := snapshotFromTask(task)
	a.operations[input.OperationKey] = transitionRecord{To: input.To, Snapshot: snapshot}
	return snapshot, nil
}

func (a *MemoryLifecycleActivities) PlanStub(_ context.Context, input StepInput) error {
	return a.runStep(ActivityPlanStub, input)
}

func (a *MemoryLifecycleActivities) RouteStub(_ context.Context, input StepInput) error {
	return a.runStep(ActivityRouteStub, input)
}

func (a *MemoryLifecycleActivities) ExecuteStub(_ context.Context, input StepInput) error {
	return a.runStep(ActivityExecuteStub, input)
}

func (a *MemoryLifecycleActivities) ValidateStub(_ context.Context, input StepInput) error {
	return a.runStep(ActivityValidateStub, input)
}

func (a *MemoryLifecycleActivities) SetTransientFailures(activityName string, count int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.stepFailures[activityName] = count
}

func (a *MemoryLifecycleActivities) SetTerminalOnStep(activityName string, status domain.TaskStatus) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.terminalOnStep[activityName] = status
}

func (a *MemoryLifecycleActivities) SetStaleOnce(operationKey string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.staleOnceForKey[operationKey] = true
}

func (a *MemoryLifecycleActivities) StepCalls(activityName string) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.stepCalls[activityName]
}

func (a *MemoryLifecycleActivities) Task(taskID string) (domain.Task, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	task, ok := a.tasks[taskID]
	return task, ok
}

func (a *MemoryLifecycleActivities) OperationCount() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.operations)
}

func (a *MemoryLifecycleActivities) runStep(activityName string, input StepInput) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, err := a.taskForScope(input.TenantID, input.ProjectID, input.TaskID, input.TraceID); err != nil {
		return err
	}
	a.stepCalls[activityName]++
	if remaining := a.stepFailures[activityName]; remaining > 0 {
		a.stepFailures[activityName] = remaining - 1
		return fmt.Errorf("transient %s failure", activityName)
	}
	if terminalStatus, ok := a.terminalOnStep[activityName]; ok {
		delete(a.terminalOnStep, activityName)
		task := a.tasks[input.TaskID]
		if !task.IsTerminal() {
			at := time.Now().UTC()
			if _, err := task.Transition(domain.TaskTransitionCommand{
				To:    terminalStatus,
				Actor: "test:external",
				Cause: "S3C terminal race fixture",
				At:    at,
			}); err != nil {
				return err
			}
			task.Version++
			a.tasks[task.ID] = task
		}
	}
	return nil
}

func (a *MemoryLifecycleActivities) taskForScope(tenantID, projectID, taskID, traceID string) (domain.Task, error) {
	task, ok := a.tasks[taskID]
	if !ok {
		return domain.Task{}, ErrLifecycleTaskNotFound
	}
	if task.TenantID != tenantID || task.ProjectID != projectID || task.TraceID != traceID {
		return domain.Task{}, ErrLifecycleScope
	}
	return task, nil
}

func snapshotFromTask(task domain.Task) TaskSnapshot {
	return TaskSnapshot{
		TaskID:   task.ID,
		TraceID:  task.TraceID,
		Status:   task.Status,
		Version:  task.Version,
		Terminal: task.IsTerminal(),
	}
}

// FailClosedLifecycleActivities is the only S3C runtime Activity backend.
// It deliberately provides no business mutation capability. S3D replaces this
// backend with PostgreSQL/RLS-backed Activities before production dispatch is
// enabled.
type FailClosedLifecycleActivities struct{}

func (FailClosedLifecycleActivities) LoadTask(context.Context, LoadTaskInput) (TaskSnapshot, error) {
	return TaskSnapshot{}, lifecycleBackendDisabledError()
}

func (FailClosedLifecycleActivities) TransitionTask(context.Context, TransitionTaskInput) (TaskSnapshot, error) {
	return TaskSnapshot{}, lifecycleBackendDisabledError()
}

func (FailClosedLifecycleActivities) PlanStub(context.Context, StepInput) error {
	return lifecycleBackendDisabledError()
}

func (FailClosedLifecycleActivities) RouteStub(context.Context, StepInput) error {
	return lifecycleBackendDisabledError()
}

func (FailClosedLifecycleActivities) ExecuteStub(context.Context, StepInput) error {
	return lifecycleBackendDisabledError()
}

func (FailClosedLifecycleActivities) ValidateStub(context.Context, StepInput) error {
	return lifecycleBackendDisabledError()
}

func lifecycleBackendDisabledError() error {
	return temporal.NewNonRetryableApplicationError(
		"S3C lifecycle business backend is not configured; S3D PostgreSQL Activities are required",
		ErrorTypeLifecycleBackendDisabled,
		nil,
	)
}
