package workflow

import (
	"context"
	"errors"
	"testing"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"go.temporal.io/sdk/temporal"
)

func TestMemoryLifecycleTransitionIsIdempotentByOperationKey(t *testing.T) {
	activities := NewMemoryLifecycleActivities(newLifecycleTask(domain.TaskCreated))
	input := TransitionTaskInput{
		TenantID:        "tenant-a",
		ProjectID:       "project-a",
		TaskID:          "task-a",
		TraceID:         "trace-a",
		ExpectedVersion: 1,
		To:              domain.TaskPlanning,
		Cause:           TaskLifecycleVersion,
		OperationKey:    TransitionOperationKey("task-a", domain.TaskPlanning),
	}
	first, err := activities.TransitionTask(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := activities.TransitionTask(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if first != second || first.Status != domain.TaskPlanning || first.Version != 2 {
		t.Fatalf("first=%+v second=%+v", first, second)
	}
	if activities.OperationCount() != 1 {
		t.Fatalf("operation count=%d", activities.OperationCount())
	}
}

func TestMemoryLifecycleIdempotentReplayStillRequiresScope(t *testing.T) {
	activities := NewMemoryLifecycleActivities(newLifecycleTask(domain.TaskCreated))
	input := TransitionTaskInput{
		TenantID:        "tenant-a",
		ProjectID:       "project-a",
		TaskID:          "task-a",
		TraceID:         "trace-a",
		ExpectedVersion: 1,
		To:              domain.TaskPlanning,
		Cause:           TaskLifecycleVersion,
		OperationKey:    TransitionOperationKey("task-a", domain.TaskPlanning),
	}
	if _, err := activities.TransitionTask(context.Background(), input); err != nil {
		t.Fatal(err)
	}

	input.TenantID = "tenant-other"
	if _, err := activities.TransitionTask(context.Background(), input); !errors.Is(err, ErrLifecycleScope) {
		t.Fatalf("cross-scope idempotent replay error=%v", err)
	}
	if activities.OperationCount() != 1 {
		t.Fatalf("operation count=%d", activities.OperationCount())
	}
}

func TestMemoryLifecycleRejectsOperationKeyReuseForDifferentTransition(t *testing.T) {
	activities := NewMemoryLifecycleActivities(newLifecycleTask(domain.TaskCreated))
	key := "task-a:transition:shared"
	_, err := activities.TransitionTask(context.Background(), TransitionTaskInput{
		TenantID: "tenant-a", ProjectID: "project-a", TaskID: "task-a", TraceID: "trace-a",
		ExpectedVersion: 1, To: domain.TaskPlanning, Cause: TaskLifecycleVersion, OperationKey: key,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = activities.TransitionTask(context.Background(), TransitionTaskInput{
		TenantID: "tenant-a", ProjectID: "project-a", TaskID: "task-a", TraceID: "trace-a",
		ExpectedVersion: 2, To: domain.TaskRouting, Cause: TaskLifecycleVersion, OperationKey: key,
	})
	if !errors.Is(err, ErrOperationKeyConflict) {
		t.Fatalf("error=%v", err)
	}
}

func TestMemoryLifecycleRejectsCrossScopeAccess(t *testing.T) {
	activities := NewMemoryLifecycleActivities(newLifecycleTask(domain.TaskCreated))
	_, err := activities.LoadTask(context.Background(), LoadTaskInput{
		TenantID: "tenant-other", ProjectID: "project-a", TaskID: "task-a", TraceID: "trace-a",
	})
	if !errors.Is(err, ErrLifecycleScope) {
		t.Fatalf("error=%v", err)
	}
}

func TestMemoryLifecycleReturnsTypedStaleVersion(t *testing.T) {
	activities := NewMemoryLifecycleActivities(newLifecycleTask(domain.TaskCreated))
	_, err := activities.TransitionTask(context.Background(), TransitionTaskInput{
		TenantID: "tenant-a", ProjectID: "project-a", TaskID: "task-a", TraceID: "trace-a",
		ExpectedVersion: 99, To: domain.TaskPlanning, Cause: TaskLifecycleVersion,
		OperationKey: TransitionOperationKey("task-a", domain.TaskPlanning),
	})
	var applicationErr *temporal.ApplicationError
	if !errors.As(err, &applicationErr) || applicationErr.Type() != ErrorTypeStaleTaskVersion {
		t.Fatalf("error=%v", err)
	}
}

func TestFailClosedLifecycleBackendHasNoBusinessCapability(t *testing.T) {
	backend := FailClosedLifecycleActivities{}
	_, err := backend.LoadTask(context.Background(), LoadTaskInput{})
	var applicationErr *temporal.ApplicationError
	if !errors.As(err, &applicationErr) || applicationErr.Type() != ErrorTypeLifecycleBackendDisabled || !applicationErr.NonRetryable() {
		t.Fatalf("error=%v", err)
	}
}
