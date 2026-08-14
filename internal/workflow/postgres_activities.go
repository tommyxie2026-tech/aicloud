package workflow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/identity"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
	tracepkg "github.com/tommyxie2026-tech/aicloud/internal/trace"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
)

const (
	WorkflowWorkerSubject          = "aicloud-workflow-worker"
	WorkflowWorkerAuthnMethod      = "internal_workload_identity"
	WorkflowWorkerIssuer          = "aicloud"
	ActivityTransitionOperationV1 = "workflow.activity.task-transition.v1"

	ErrorTypeActivityAttestation      = "ACTIVITY_EXECUTION_ATTESTATION_FAILED"
	ErrorTypeActivityScope            = "ACTIVITY_SCOPE_INVALID"
	ErrorTypeActivityTaskNotFound     = "ACTIVITY_TASK_NOT_FOUND"
	ErrorTypeActivityInput            = "ACTIVITY_INPUT_INVALID"
	ErrorTypeActivityTransition       = "ACTIVITY_TRANSITION_INVALID"
	ErrorTypeActivityIdempotency      = "ACTIVITY_IDEMPOTENCY_CONFLICT"
	ErrorTypeActivityReplayCorruption = "ACTIVITY_IDEMPOTENCY_REPLAY_INVALID"
)

type ActivityTrustConfig struct {
	Namespace            string
	TaskQueue            string
	WorkerSubject        string
	IdempotencyRetention time.Duration
}

func (c ActivityTrustConfig) normalized() (ActivityTrustConfig, error) {
	c.Namespace = strings.TrimSpace(c.Namespace)
	c.TaskQueue = strings.TrimSpace(c.TaskQueue)
	c.WorkerSubject = strings.TrimSpace(c.WorkerSubject)
	if c.Namespace == "" {
		return ActivityTrustConfig{}, fmt.Errorf("Temporal Activity namespace is required")
	}
	if c.TaskQueue == "" {
		return ActivityTrustConfig{}, fmt.Errorf("Temporal Activity task queue is required")
	}
	if c.WorkerSubject == "" {
		c.WorkerSubject = WorkflowWorkerSubject
	}
	if c.IdempotencyRetention <= 0 {
		c.IdempotencyRetention = 30 * 24 * time.Hour
	}
	return c, nil
}

type ActivityExecutionInfo struct {
	WorkflowID   string
	WorkflowRunID string
	WorkflowType string
	Namespace    string
	TaskQueue    string
	ActivityID   string
	ActivityType string
	Attempt      int32
}

type activityInfoProvider func(context.Context) ActivityExecutionInfo

type PostgresLifecycleActivities struct {
	tasks    domain.TaskRepository
	commands repository.TaskCommandStore
	trust    ActivityTrustConfig
	info     activityInfoProvider
	now      func() time.Time
}

func NewPostgresLifecycleActivities(
	tasks domain.TaskRepository,
	commands repository.TaskCommandStore,
	trust ActivityTrustConfig,
) (*PostgresLifecycleActivities, error) {
	return newPostgresLifecycleActivities(tasks, commands, trust, temporalActivityExecutionInfo)
}

func newPostgresLifecycleActivities(
	tasks domain.TaskRepository,
	commands repository.TaskCommandStore,
	trust ActivityTrustConfig,
	info activityInfoProvider,
) (*PostgresLifecycleActivities, error) {
	if tasks == nil {
		return nil, fmt.Errorf("Task repository is required")
	}
	if commands == nil {
		return nil, fmt.Errorf("Task command store is required")
	}
	if info == nil {
		return nil, fmt.Errorf("Activity execution-info provider is required")
	}
	normalized, err := trust.normalized()
	if err != nil {
		return nil, err
	}
	return &PostgresLifecycleActivities{
		tasks:    tasks,
		commands: commands,
		trust:    normalized,
		info:     info,
		now:      time.Now,
	}, nil
}

func (a *PostgresLifecycleActivities) LoadTask(ctx context.Context, input LoadTaskInput) (TaskSnapshot, error) {
	if err := validateLoadTaskInput(input); err != nil {
		return TaskSnapshot{}, nonRetryable(ErrorTypeActivityInput, err)
	}
	_, principal, err := a.attest(ctx, ActivityLoadTask, input.TenantID, input.ProjectID, input.TaskID)
	if err != nil {
		return TaskSnapshot{}, err
	}

	scopedCtx := identity.WithPrincipal(ctx, principal)
	task, err := a.tasks.Get(scopedCtx, input.TaskID)
	if err != nil {
		return TaskSnapshot{}, a.mapScopedReadError(err)
	}
	if err := validateTaskCorrelation(task, input.TenantID, input.ProjectID, input.TaskID, input.TraceID); err != nil {
		return TaskSnapshot{}, nonRetryable(ErrorTypeActivityScope, err)
	}
	return snapshotFromTask(task), nil
}

func (a *PostgresLifecycleActivities) TransitionTask(ctx context.Context, input TransitionTaskInput) (TaskSnapshot, error) {
	if err := validateTransitionTaskInput(input); err != nil {
		return TaskSnapshot{}, nonRetryable(ErrorTypeActivityInput, err)
	}
	execution, principal, err := a.attest(ctx, ActivityTransition, input.TenantID, input.ProjectID, input.TaskID)
	if err != nil {
		return TaskSnapshot{}, err
	}
	if expectedKey := TransitionOperationKey(input.TaskID, input.To); input.OperationKey != expectedKey {
		return TaskSnapshot{}, nonRetryable(
			ErrorTypeActivityInput,
			fmt.Errorf("operation key %q does not match canonical key %q", input.OperationKey, expectedKey),
		)
	}

	scopedCtx := identity.WithPrincipal(ctx, principal)
	current, err := a.tasks.Get(scopedCtx, input.TaskID)
	if err != nil {
		return TaskSnapshot{}, a.mapScopedReadError(err)
	}
	if err := validateTaskCorrelation(current, input.TenantID, input.ProjectID, input.TaskID, input.TraceID); err != nil {
		return TaskSnapshot{}, nonRetryable(ErrorTypeActivityScope, err)
	}
	if current.IsTerminal() {
		return snapshotFromTask(current), nil
	}
	if current.Version != input.ExpectedVersion {
		return TaskSnapshot{}, staleTaskVersionError(input.ExpectedVersion, current.Version)
	}

	now := a.now().UTC()
	next := current
	transition, err := next.Transition(domain.TaskTransitionCommand{
		To:    input.To,
		Actor: principal.SubjectID,
		Cause: input.Cause,
		At:    now,
	})
	if err != nil {
		return TaskSnapshot{}, nonRetryable(ErrorTypeActivityTransition, err)
	}

	eventType, ok := domain.CanonicalTaskStateEvent(input.To)
	if !ok {
		return TaskSnapshot{}, nonRetryable(
			ErrorTypeActivityTransition,
			fmt.Errorf("no canonical Task event exists for target state %q", input.To),
		)
	}
	eventPayload, err := json.Marshal(map[string]any{
		"fromStatus":   transition.From,
		"toStatus":     transition.To,
		"cause":        transition.Cause,
		"operationKey": input.OperationKey,
		"workflowId":   execution.WorkflowID,
		"workflowType": execution.WorkflowType,
	})
	if err != nil {
		return TaskSnapshot{}, fmt.Errorf("encode lifecycle TaskEvent payload: %w", err)
	}

	committedSnapshot := snapshotFromTask(next)
	committedSnapshot.Version = input.ExpectedVersion + 1
	responsePayload, err := json.Marshal(committedSnapshot)
	if err != nil {
		return TaskSnapshot{}, fmt.Errorf("encode lifecycle idempotency response: %w", err)
	}
	requestDigest, err := lifecycleTransitionDigest(input, execution)
	if err != nil {
		return TaskSnapshot{}, fmt.Errorf("digest lifecycle transition: %w", err)
	}
	responseDigest := sha256Digest(responsePayload)

	commit := repository.TaskCommandCommit{
		Task:            next,
		ExpectedVersion: input.ExpectedVersion,
		Event: domain.TaskEvent{
			EventID:   tracepkg.NewID("task-event"),
			EventType: eventType,
			Actor: domain.TaskEventActor{
				PrincipalType: string(principal.Type),
				SubjectID:     principal.SubjectID,
			},
			Payload:       eventPayload,
			SchemaVersion: 1,
			OccurredAt:    now,
			CreatedAt:     now,
		},
		Idempotency: domain.IdempotencyRecord{
			TenantID:        principal.TenantID,
			ProjectID:       principal.ProjectID,
			Operation:       ActivityTransitionOperationV1,
			Key:             input.OperationKey,
			RequestDigest:   requestDigest,
			Status:          domain.IdempotencyCompleted,
			ResourceID:      input.TaskID,
			ResponseCode:    200,
			ResponseDigest:  responseDigest,
			ResponsePayload: responsePayload,
			CreatedAt:       now,
			ExpiresAt:       now.Add(a.trust.IdempotencyRetention),
		},
	}

	result, err := a.commands.CommitTransition(scopedCtx, commit)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrVersionConflict):
			return TaskSnapshot{}, staleTaskVersionError(input.ExpectedVersion, current.Version)
		case errors.Is(err, repository.ErrIdempotencyConflict):
			return TaskSnapshot{}, nonRetryable(ErrorTypeActivityIdempotency, err)
		case errors.Is(err, domain.ErrInvalidTaskTransition), errors.Is(err, domain.ErrTaskTerminal):
			return TaskSnapshot{}, nonRetryable(ErrorTypeActivityTransition, err)
		default:
			return TaskSnapshot{}, err
		}
	}
	if result.Replayed {
		return decodeReplaySnapshot(result.Idempotency.ResponsePayload, input)
	}
	if err := validateTaskCorrelation(result.Task, input.TenantID, input.ProjectID, input.TaskID, input.TraceID); err != nil {
		return TaskSnapshot{}, nonRetryable(ErrorTypeActivityReplayCorruption, err)
	}
	return snapshotFromTask(result.Task), nil
}

// Stub execution methods remain fail-closed until S3E supplies real governed
// planning/routing/execution/validation semantics.
func (*PostgresLifecycleActivities) PlanStub(context.Context, StepInput) error {
	return lifecycleBackendDisabledError()
}

func (*PostgresLifecycleActivities) RouteStub(context.Context, StepInput) error {
	return lifecycleBackendDisabledError()
}

func (*PostgresLifecycleActivities) ExecuteStub(context.Context, StepInput) error {
	return lifecycleBackendDisabledError()
}

func (*PostgresLifecycleActivities) ValidateStub(context.Context, StepInput) error {
	return lifecycleBackendDisabledError()
}

func (a *PostgresLifecycleActivities) attest(
	ctx context.Context,
	expectedActivity, tenantID, projectID, taskID string,
) (ActivityExecutionInfo, identity.Principal, error) {
	info := a.info(ctx)
	expectedWorkflowID, err := WorkflowID(taskID)
	if err != nil {
		return ActivityExecutionInfo{}, identity.Principal{}, nonRetryable(ErrorTypeActivityAttestation, err)
	}
	if info.WorkflowType != TaskExecutionWorkflowType {
		return ActivityExecutionInfo{}, identity.Principal{}, nonRetryable(
			ErrorTypeActivityAttestation,
			fmt.Errorf("unexpected workflow type %q", info.WorkflowType),
		)
	}
	if info.WorkflowID != expectedWorkflowID {
		return ActivityExecutionInfo{}, identity.Principal{}, nonRetryable(
			ErrorTypeActivityAttestation,
			fmt.Errorf("workflow identity does not match Task identity"),
		)
	}
	if info.Namespace != a.trust.Namespace {
		return ActivityExecutionInfo{}, identity.Principal{}, nonRetryable(
			ErrorTypeActivityAttestation,
			fmt.Errorf("unexpected Temporal namespace"),
		)
	}
	if info.TaskQueue != a.trust.TaskQueue {
		return ActivityExecutionInfo{}, identity.Principal{}, nonRetryable(
			ErrorTypeActivityAttestation,
			fmt.Errorf("unexpected Temporal task queue"),
		)
	}
	if info.ActivityType != expectedActivity {
		return ActivityExecutionInfo{}, identity.Principal{}, nonRetryable(
			ErrorTypeActivityAttestation,
			fmt.Errorf("unexpected activity type %q", info.ActivityType),
		)
	}
	if strings.TrimSpace(info.WorkflowRunID) == "" || strings.TrimSpace(info.ActivityID) == "" {
		return ActivityExecutionInfo{}, identity.Principal{}, nonRetryable(
			ErrorTypeActivityAttestation,
			fmt.Errorf("Temporal execution identity is incomplete"),
		)
	}

	principal := identity.Principal{
		Type:       identity.PrincipalServiceAccount,
		SubjectID:  a.trust.WorkerSubject,
		TenantID:   strings.TrimSpace(tenantID),
		ProjectID:  strings.TrimSpace(projectID),
		AuthnMethod: WorkflowWorkerAuthnMethod,
		Issuer:     WorkflowWorkerIssuer,
	}
	if err := principal.Validate(); err != nil {
		return ActivityExecutionInfo{}, identity.Principal{}, nonRetryable(ErrorTypeActivityScope, err)
	}
	if principal.ProjectID == "" {
		return ActivityExecutionInfo{}, identity.Principal{}, nonRetryable(
			ErrorTypeActivityScope,
			fmt.Errorf("project ID is required for workflow worker"),
		)
	}
	return info, principal, nil
}

func temporalActivityExecutionInfo(ctx context.Context) ActivityExecutionInfo {
	info := activity.GetInfo(ctx)
	workflowType := ""
	if info.WorkflowType != nil {
		workflowType = info.WorkflowType.Name
	}
	return ActivityExecutionInfo{
		WorkflowID:    info.WorkflowExecution.ID,
		WorkflowRunID: info.WorkflowExecution.RunID,
		WorkflowType:  workflowType,
		Namespace:     info.Namespace,
		TaskQueue:     info.TaskQueue,
		ActivityID:    info.ActivityID,
		ActivityType:  info.ActivityType.Name,
		Attempt:       info.Attempt,
	}
}

func validateLoadTaskInput(input LoadTaskInput) error {
	if strings.TrimSpace(input.TenantID) == "" || strings.TrimSpace(input.ProjectID) == "" ||
		strings.TrimSpace(input.TaskID) == "" || strings.TrimSpace(input.TraceID) == "" {
		return fmt.Errorf("tenant, project, Task and trace identity are required")
	}
	return nil
}

func validateTransitionTaskInput(input TransitionTaskInput) error {
	if err := validateLoadTaskInput(LoadTaskInput{
		TenantID: input.TenantID, ProjectID: input.ProjectID, TaskID: input.TaskID, TraceID: input.TraceID,
	}); err != nil {
		return err
	}
	if input.ExpectedVersion <= 0 {
		return fmt.Errorf("expected Task version must be greater than zero")
	}
	if strings.TrimSpace(string(input.To)) == "" || strings.TrimSpace(input.Cause) == "" || strings.TrimSpace(input.OperationKey) == "" {
		return fmt.Errorf("target state, cause and operation key are required")
	}
	return nil
}

func validateTaskCorrelation(task domain.Task, tenantID, projectID, taskID, traceID string) error {
	if task.ID != taskID || task.TenantID != tenantID || task.ProjectID != projectID || task.TraceID != traceID {
		return fmt.Errorf("Task correlation does not match attested Activity scope")
	}
	return nil
}

func (a *PostgresLifecycleActivities) mapScopedReadError(err error) error {
	if errors.Is(err, repository.ErrNotFound) {
		return nonRetryable(ErrorTypeActivityTaskNotFound, fmt.Errorf("Task is not available in the attested scope"))
	}
	return err
}

func lifecycleTransitionDigest(input TransitionTaskInput, execution ActivityExecutionInfo) (string, error) {
	payload, err := json.Marshal(struct {
		TenantID        string            `json:"tenantId"`
		ProjectID       string            `json:"projectId"`
		TaskID          string            `json:"taskId"`
		TraceID         string            `json:"traceId"`
		ExpectedVersion int64             `json:"expectedVersion"`
		To              domain.TaskStatus `json:"to"`
		Cause           string            `json:"cause"`
		OperationKey    string            `json:"operationKey"`
		WorkflowType    string            `json:"workflowType"`
		WorkflowID      string            `json:"workflowId"`
	}{
		TenantID: input.TenantID, ProjectID: input.ProjectID, TaskID: input.TaskID, TraceID: input.TraceID,
		ExpectedVersion: input.ExpectedVersion, To: input.To, Cause: input.Cause, OperationKey: input.OperationKey,
		WorkflowType: execution.WorkflowType, WorkflowID: execution.WorkflowID,
	})
	if err != nil {
		return "", err
	}
	return sha256Digest(payload), nil
}

func sha256Digest(payload []byte) string {
	sum := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func decodeReplaySnapshot(payload json.RawMessage, input TransitionTaskInput) (TaskSnapshot, error) {
	if len(payload) == 0 {
		return TaskSnapshot{}, nonRetryable(
			ErrorTypeActivityReplayCorruption,
			fmt.Errorf("idempotency replay response is empty"),
		)
	}
	var snapshot TaskSnapshot
	if err := json.Unmarshal(payload, &snapshot); err != nil {
		return TaskSnapshot{}, nonRetryable(ErrorTypeActivityReplayCorruption, err)
	}
	if snapshot.TaskID != input.TaskID || snapshot.TraceID != input.TraceID || snapshot.Version <= 0 {
		return TaskSnapshot{}, nonRetryable(
			ErrorTypeActivityReplayCorruption,
			fmt.Errorf("idempotency replay TaskSnapshot correlation is invalid"),
		)
	}
	return snapshot, nil
}

func staleTaskVersionError(expected, actual int64) error {
	return temporal.NewNonRetryableApplicationError(
		fmt.Sprintf("Task version is stale: expected %d, actual %d", expected, actual),
		ErrorTypeStaleTaskVersion,
		nil,
	)
}

func nonRetryable(errorType string, err error) error {
	return temporal.NewNonRetryableApplicationError(err.Error(), errorType, err)
}
