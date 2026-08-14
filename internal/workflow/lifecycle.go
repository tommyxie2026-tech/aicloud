package workflow

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"go.temporal.io/sdk/temporal"
	temporalworkflow "go.temporal.io/sdk/workflow"
)

const (
	TaskWorkflowSchemaVersion = 1
	TaskLifecycleVersion      = "lifecycle-v1"

	ActivityLoadTask     = "aicloud.task.load.v1"
	ActivityTransition   = "aicloud.task.transition.v1"
	ActivityPlanStub     = "aicloud.task.plan.stub.v1"
	ActivityRouteStub    = "aicloud.task.route.stub.v1"
	ActivityExecuteStub  = "aicloud.task.execute.stub.v1"
	ActivityValidateStub = "aicloud.task.validate.stub.v1"

	ErrorTypeStaleTaskVersion         = "STALE_TASK_VERSION"
	ErrorTypeLifecycleBackendDisabled = "LIFECYCLE_BACKEND_NOT_CONFIGURED"
)

var ErrInvalidTaskWorkflowInput = errors.New("invalid Task workflow input")

type TaskWorkflowInput struct {
	SchemaVersion int    `json:"schemaVersion"`
	TenantID      string `json:"tenantId"`
	ProjectID     string `json:"projectId"`
	TaskID        string `json:"taskId"`
	TraceID       string `json:"traceId"`
}

func (i TaskWorkflowInput) Validate() error {
	if i.SchemaVersion != TaskWorkflowSchemaVersion {
		return fmt.Errorf("%w: unsupported schema version %d", ErrInvalidTaskWorkflowInput, i.SchemaVersion)
	}
	if strings.TrimSpace(i.TenantID) == "" {
		return fmt.Errorf("%w: tenant ID is required", ErrInvalidTaskWorkflowInput)
	}
	if strings.TrimSpace(i.ProjectID) == "" {
		return fmt.Errorf("%w: project ID is required", ErrInvalidTaskWorkflowInput)
	}
	if strings.TrimSpace(i.TaskID) == "" {
		return fmt.Errorf("%w: task ID is required", ErrInvalidTaskWorkflowInput)
	}
	if strings.TrimSpace(i.TraceID) == "" {
		return fmt.Errorf("%w: trace ID is required", ErrInvalidTaskWorkflowInput)
	}
	return nil
}

type TaskWorkflowResult struct {
	TaskID          string            `json:"taskId"`
	TraceID         string            `json:"traceId"`
	ObservedStatus  domain.TaskStatus `json:"observedStatus"`
	AlreadyTerminal bool              `json:"alreadyTerminal"`
	Steps           []string          `json:"steps,omitempty"`
}

type LoadTaskInput struct {
	TenantID  string `json:"tenantId"`
	ProjectID string `json:"projectId"`
	TaskID    string `json:"taskId"`
	TraceID   string `json:"traceId"`
}

type TaskSnapshot struct {
	TaskID   string            `json:"taskId"`
	TraceID  string            `json:"traceId"`
	Status   domain.TaskStatus `json:"status"`
	Version  int64             `json:"version"`
	Terminal bool              `json:"terminal"`
}

type TransitionTaskInput struct {
	TenantID        string            `json:"tenantId"`
	ProjectID       string            `json:"projectId"`
	TaskID          string            `json:"taskId"`
	TraceID         string            `json:"traceId"`
	ExpectedVersion int64             `json:"expectedVersion"`
	To              domain.TaskStatus `json:"to"`
	Cause           string            `json:"cause"`
	OperationKey    string            `json:"operationKey"`
}

type StepInput struct {
	TenantID  string `json:"tenantId"`
	ProjectID string `json:"projectId"`
	TaskID    string `json:"taskId"`
	TraceID   string `json:"traceId"`
}

type LifecycleActivities interface {
	LoadTask(context.Context, LoadTaskInput) (TaskSnapshot, error)
	TransitionTask(context.Context, TransitionTaskInput) (TaskSnapshot, error)
	PlanStub(context.Context, StepInput) error
	RouteStub(context.Context, StepInput) error
	ExecuteStub(context.Context, StepInput) error
	ValidateStub(context.Context, StepInput) error
}

func TransitionOperationKey(taskID string, to domain.TaskStatus) string {
	return strings.TrimSpace(taskID) + ":transition:" + string(to) + ":" + TaskLifecycleVersion
}

func TaskLifecycleWorkflow(ctx temporalworkflow.Context, input TaskWorkflowInput) (TaskWorkflowResult, error) {
	if err := input.Validate(); err != nil {
		return TaskWorkflowResult{}, temporal.NewNonRetryableApplicationError(err.Error(), "INVALID_TASK_WORKFLOW_INPUT", err)
	}

	ctx = temporalworkflow.WithActivityOptions(ctx, temporalworkflow.ActivityOptions{
		StartToCloseTimeout:    10 * time.Second,
		ScheduleToCloseTimeout: 60 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2,
			MaximumInterval:    10 * time.Second,
			MaximumAttempts:    3,
		},
	})

	result := TaskWorkflowResult{TaskID: input.TaskID, TraceID: input.TraceID}
	for {
		snapshot, err := loadTaskSnapshot(ctx, input)
		if err != nil {
			return result, err
		}
		result.ObservedStatus = snapshot.Status
		if snapshot.Terminal || isTerminalStatus(snapshot.Status) {
			result.AlreadyTerminal = true
			return result, nil
		}

		switch snapshot.Status {
		case domain.TaskCreated:
			next, err := transitionTask(ctx, input, snapshot, domain.TaskPlanning)
			if err != nil {
				return result, err
			}
			result.ObservedStatus = next.Status
			result.Steps = append(result.Steps, string(domain.TaskPlanning))
		case domain.TaskPlanning:
			if err := executeStub(ctx, ActivityPlanStub, input); err != nil {
				return result, err
			}
			result.Steps = append(result.Steps, "plan")
			next, err := transitionTask(ctx, input, snapshot, domain.TaskRouting)
			if err != nil {
				return result, err
			}
			result.ObservedStatus = next.Status
			result.Steps = append(result.Steps, string(domain.TaskRouting))
		case domain.TaskRouting:
			if err := executeStub(ctx, ActivityRouteStub, input); err != nil {
				return result, err
			}
			result.Steps = append(result.Steps, "route")
			next, err := transitionTask(ctx, input, snapshot, domain.TaskExecuting)
			if err != nil {
				return result, err
			}
			result.ObservedStatus = next.Status
			result.Steps = append(result.Steps, string(domain.TaskExecuting))
		case domain.TaskExecuting:
			if err := executeStub(ctx, ActivityExecuteStub, input); err != nil {
				return result, err
			}
			result.Steps = append(result.Steps, "execute")
			next, err := transitionTask(ctx, input, snapshot, domain.TaskValidating)
			if err != nil {
				return result, err
			}
			result.ObservedStatus = next.Status
			result.Steps = append(result.Steps, string(domain.TaskValidating))
		case domain.TaskValidating:
			if err := executeStub(ctx, ActivityValidateStub, input); err != nil {
				return result, err
			}
			result.Steps = append(result.Steps, "validate")
			next, err := transitionTask(ctx, input, snapshot, domain.TaskCompleted)
			if err != nil {
				return result, err
			}
			result.ObservedStatus = next.Status
			result.Steps = append(result.Steps, string(domain.TaskCompleted))
		case domain.TaskWaitingApproval:
			return result, temporal.NewNonRetryableApplicationError(
				"WAITING_APPROVAL is not handled by the S3C lifecycle",
				"WAITING_APPROVAL_NOT_IMPLEMENTED",
				nil,
			)
		default:
			return result, temporal.NewNonRetryableApplicationError(
				fmt.Sprintf("unsupported Task status %q", snapshot.Status),
				"UNSUPPORTED_TASK_STATUS",
				nil,
			)
		}
	}
}

func loadTaskSnapshot(ctx temporalworkflow.Context, input TaskWorkflowInput) (TaskSnapshot, error) {
	var snapshot TaskSnapshot
	err := temporalworkflow.ExecuteActivity(ctx, ActivityLoadTask, LoadTaskInput{
		TenantID:  input.TenantID,
		ProjectID: input.ProjectID,
		TaskID:    input.TaskID,
		TraceID:   input.TraceID,
	}).Get(ctx, &snapshot)
	return snapshot, err
}

func transitionTask(ctx temporalworkflow.Context, input TaskWorkflowInput, snapshot TaskSnapshot, to domain.TaskStatus) (TaskSnapshot, error) {
	var next TaskSnapshot
	err := temporalworkflow.ExecuteActivity(ctx, ActivityTransition, TransitionTaskInput{
		TenantID:        input.TenantID,
		ProjectID:       input.ProjectID,
		TaskID:          input.TaskID,
		TraceID:         input.TraceID,
		ExpectedVersion: snapshot.Version,
		To:              to,
		Cause:           TaskLifecycleVersion,
		OperationKey:    TransitionOperationKey(input.TaskID, to),
	}).Get(ctx, &next)
	if err == nil {
		return next, nil
	}
	if isApplicationErrorType(err, ErrorTypeStaleTaskVersion) {
		return loadTaskSnapshot(ctx, input)
	}
	return TaskSnapshot{}, err
}

func executeStub(ctx temporalworkflow.Context, activityName string, input TaskWorkflowInput) error {
	return temporalworkflow.ExecuteActivity(ctx, activityName, StepInput{
		TenantID:  input.TenantID,
		ProjectID: input.ProjectID,
		TaskID:    input.TaskID,
		TraceID:   input.TraceID,
	}).Get(ctx, nil)
}

func isApplicationErrorType(err error, errorType string) bool {
	var applicationErr *temporal.ApplicationError
	return errors.As(err, &applicationErr) && applicationErr.Type() == errorType
}

func isTerminalStatus(status domain.TaskStatus) bool {
	switch status {
	case domain.TaskCompleted, domain.TaskFailed, domain.TaskCancelled, domain.TaskExpired:
		return true
	default:
		return false
	}
}
