package workflow

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/identity"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
)

type reconciliationTasks struct {
	domain.TaskRepository
	task      domain.Task
	getCalls  int
	principal identity.Principal
}

func (r *reconciliationTasks) Get(ctx context.Context, id string) (domain.Task, error) {
	r.getCalls++
	principal, err := identity.RequireProject(ctx)
	if err != nil {
		return domain.Task{}, err
	}
	r.principal = principal
	if r.task.ID != id || r.task.TenantID != principal.TenantID || r.task.ProjectID != principal.ProjectID {
		return domain.Task{}, repository.ErrNotFound
	}
	return r.task, nil
}

type reconciliationObserver struct {
	observation ExecutionObservation
	err         error
	workflowID  string
	calls       int
}

func (o *reconciliationObserver) ObserveExecution(_ context.Context, workflowID string) (ExecutionObservation, error) {
	o.calls++
	o.workflowID = workflowID
	if o.err != nil {
		return ExecutionObservation{}, o.err
	}
	return o.observation, nil
}

func TestReconciliationClassificationNeverInfersBusinessTerminalState(t *testing.T) {
	cases := []struct {
		name      string
		status    domain.TaskStatus
		execution ExecutionCondition
		want      ReconciliationClassification
	}{
		{name: "nonterminal open", status: domain.TaskExecuting, execution: ExecutionOpen, want: ReconciliationHealthy},
		{name: "nonterminal missing", status: domain.TaskExecuting, execution: ExecutionMissing, want: ReconciliationRecoveryRequired},
		{name: "nonterminal closed success", status: domain.TaskValidating, execution: ExecutionClosedSuccessful, want: ReconciliationRecoveryRequired},
		{name: "nonterminal failed", status: domain.TaskPlanning, execution: ExecutionClosedFailed, want: ReconciliationRecoveryRequired},
		{name: "terminal open", status: domain.TaskCancelled, execution: ExecutionOpen, want: ReconciliationEnsureCancel},
		{name: "terminal missing", status: domain.TaskCompleted, execution: ExecutionMissing, want: ReconciliationConsistent},
		{name: "terminal failed", status: domain.TaskFailed, execution: ExecutionClosedFailed, want: ReconciliationConsistent},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			task := reconciliationTask(tc.status)
			decision := classifyReconciliation(task, ExecutionObservation{
				WorkflowID: "task/" + task.ID,
				RunID:      "run-a",
				Condition:  tc.execution,
			})
			if decision.Classification != tc.want {
				t.Fatalf("classification=%q want=%q decision=%+v", decision.Classification, tc.want, decision)
			}
			if decision.TaskStatus != tc.status {
				t.Fatalf("reconciliation changed business status: got=%s want=%s", decision.TaskStatus, tc.status)
			}
		})
	}
}

func TestTaskReconcilerRequiresProjectScopeAndAttestedTaskIdentity(t *testing.T) {
	task := reconciliationTask(domain.TaskExecuting)
	tasks := &reconciliationTasks{task: task}
	observer := &reconciliationObserver{observation: ExecutionObservation{
		WorkflowID: "task/" + task.ID,
		RunID:      "run-a",
		Condition:  ExecutionOpen,
	}}
	reconciler, err := NewTaskReconciler(tasks, observer)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := reconciler.Inspect(context.Background(), task.ID); !errors.Is(err, identity.ErrPrincipalRequired) {
		t.Fatalf("missing project principal error=%v", err)
	}
	if tasks.getCalls != 0 || observer.calls != 0 {
		t.Fatalf("reconciliation touched dependencies without project identity: get=%d observe=%d", tasks.getCalls, observer.calls)
	}

	ctx := identity.WithPrincipal(context.Background(), identity.Principal{
		Type: identity.PrincipalServiceAccount, SubjectID: "reconciler-a",
		TenantID: "tenant-a", ProjectID: "project-a", AuthnMethod: "internal_workload_identity", Issuer: "aicloud",
	})
	decision, err := reconciler.Inspect(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Classification != ReconciliationHealthy || observer.workflowID != "task/"+task.ID {
		t.Fatalf("unexpected reconciliation decision=%+v observed=%q", decision, observer.workflowID)
	}
	if tasks.principal.TenantID != "tenant-a" || tasks.principal.ProjectID != "project-a" {
		t.Fatalf("Task read escaped scoped principal: %+v", tasks.principal)
	}
}

func TestTaskReconcilerRejectsCrossScopeAndWorkflowObservationMismatch(t *testing.T) {
	task := reconciliationTask(domain.TaskExecuting)
	tasks := &reconciliationTasks{task: task}
	observer := &reconciliationObserver{observation: ExecutionObservation{
		WorkflowID: "task/other",
		Condition:  ExecutionOpen,
	}}
	reconciler, err := NewTaskReconciler(tasks, observer)
	if err != nil {
		t.Fatal(err)
	}

	wrongProject := identity.WithPrincipal(context.Background(), identity.Principal{
		Type: identity.PrincipalServiceAccount, SubjectID: "reconciler-a",
		TenantID: "tenant-a", ProjectID: "project-b", AuthnMethod: "internal_workload_identity", Issuer: "aicloud",
	})
	if _, err := reconciler.Inspect(wrongProject, task.ID); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("cross-project reconciliation error=%v want ErrNotFound", err)
	}
	if observer.calls != 0 {
		t.Fatalf("cross-project Task reached execution observer, calls=%d", observer.calls)
	}

	correctProject := identity.WithPrincipal(context.Background(), identity.Principal{
		Type: identity.PrincipalServiceAccount, SubjectID: "reconciler-a",
		TenantID: "tenant-a", ProjectID: "project-a", AuthnMethod: "internal_workload_identity", Issuer: "aicloud",
	})
	if _, err := reconciler.Inspect(correctProject, task.ID); err == nil {
		t.Fatal("mismatched workflow observation was accepted")
	}
}

func reconciliationTask(status domain.TaskStatus) domain.Task {
	now := time.Date(2026, 8, 14, 8, 0, 0, 0, time.UTC)
	var completedAt *time.Time
	if status == domain.TaskCompleted || status == domain.TaskFailed || status == domain.TaskCancelled || status == domain.TaskExpired {
		completedAt = &now
	}
	return domain.Task{
		ID: "task-reconcile-a", TenantID: "tenant-a", ProjectID: "project-a", CreatedBy: "user-a",
		AgentID: "agent-a", Input: "reconcile", Status: status, Version: 3, TraceID: "trace-reconcile-a",
		Currency: "USD", CreatedAt: now, UpdatedAt: now, CompletedAt: completedAt,
	}
}
