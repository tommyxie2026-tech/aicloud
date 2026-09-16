package workflow

import (
	"context"
	"errors"
	"testing"
	"time"

	execution "github.com/tommyxie2026-tech/aicloud/execution"
	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
	"go.temporal.io/sdk/temporal"
)

type executionTaskRepository struct{ task domain.Task }

func (r *executionTaskRepository) List(context.Context) ([]domain.Task, error) {
	return []domain.Task{r.task}, nil
}
func (r *executionTaskRepository) Get(_ context.Context, id string) (domain.Task, error) {
	if r.task.ID != id {
		return domain.Task{}, repository.ErrNotFound
	}
	return r.task, nil
}
func (r *executionTaskRepository) Create(_ context.Context, task domain.Task) (domain.Task, error) {
	r.task = task
	return task, nil
}
func (r *executionTaskRepository) Update(_ context.Context, task domain.Task) (domain.Task, error) {
	r.task = task
	return task, nil
}

type executionLineageStub struct {
	goal      execution.Goal
	plan      execution.ExecutionPlan
	execution execution.Execution
	created   int
}

func (s *executionLineageStub) GetGoal(_ context.Context, id execution.GoalID) (execution.Goal, error) {
	if s.goal.ID != id {
		return execution.Goal{}, repository.ErrNotFound
	}
	return s.goal, nil
}
func (s *executionLineageStub) GetPlan(_ context.Context, id execution.PlanID, revision int64) (execution.ExecutionPlan, error) {
	if s.plan.ID != id || s.plan.Revision != revision {
		return execution.ExecutionPlan{}, repository.ErrNotFound
	}
	return s.plan, nil
}
func (s *executionLineageStub) CreateExecution(_ context.Context, item execution.Execution) (execution.Execution, error) {
	s.created++
	s.execution = item
	return item, nil
}
func (s *executionLineageStub) GetExecution(_ context.Context, id execution.ExecutionID) (execution.Execution, error) {
	if s.execution.ID != id {
		return execution.Execution{}, repository.ErrNotFound
	}
	return s.execution, nil
}

type executionSelectionStub struct {
	result repository.RouteSelectionResult
}

func (s *executionSelectionStub) ResolveRouteSelection(context.Context, repository.IdempotencyLookup) (repository.RouteSelectionResult, bool, error) {
	return s.result, true, nil
}
func (s *executionSelectionStub) CommitRouteSelection(context.Context, repository.RouteSelectionCommit) (repository.RouteSelectionResult, error) {
	return repository.RouteSelectionResult{}, errors.New("unexpected route commit during execution")
}

type executionRuntimeStub struct {
	record     execution.NodeRuntimeRecord
	getMissing bool
	attempts   []execution.AttemptRecord
	registers  int
	requeues   int
}

func (s *executionRuntimeStub) Register(_ context.Context, record execution.NodeRuntimeRecord) error {
	s.registers++
	s.record = record
	s.getMissing = false
	return nil
}
func (s *executionRuntimeStub) Get(_ context.Context, id execution.ExecutionID, revision int64, node execution.NodeID) (execution.NodeRuntimeRecord, *execution.NodeLease, error) {
	if s.getMissing {
		return execution.NodeRuntimeRecord{}, nil, repository.ErrNodeRuntimeNotFound
	}
	if s.record.ExecutionRef != id || s.record.PlanRevision != revision || s.record.NodeRef != node {
		return execution.NodeRuntimeRecord{}, nil, repository.ErrNodeRuntimeNotFound
	}
	return s.record, nil, nil
}
func (s *executionRuntimeStub) RequeueFailed(_ context.Context, id execution.ExecutionID, revision int64, node execution.NodeID) error {
	if s.record.ExecutionRef != id || s.record.PlanRevision != revision || s.record.NodeRef != node {
		return repository.ErrNodeRuntimeNotFound
	}
	s.requeues++
	s.record.State = execution.NodeReady
	return nil
}
func (s *executionRuntimeStub) ListAttempts(context.Context, execution.ExecutionID, int64, execution.NodeID) ([]execution.AttemptRecord, error) {
	return append([]execution.AttemptRecord(nil), s.attempts...), nil
}

type executionWorkerStub struct {
	calls    int
	lastExec execution.Execution
	lastNode execution.PreparedNode
	runtime  *executionRuntimeStub
	err      error
}

func (s *executionWorkerStub) Execute(_ context.Context, item execution.Execution, node execution.PreparedNode) (execution.WorkerExecutionResult, error) {
	s.calls++
	s.lastExec = item
	s.lastNode = node
	if s.err != nil {
		return execution.WorkerExecutionResult{}, s.err
	}
	if s.runtime != nil {
		s.runtime.record.State = execution.NodeSucceeded
	}
	return execution.WorkerExecutionResult{}, nil
}

func TestDurableExecutionActivitySucceededReplaySkipsWorker(t *testing.T) {
	activity, task, plan, runtime, worker := executionActivityFixture(t)
	lineage := activity.Lineage.(*executionLineageStub)
	lineage.execution = fixtureExecution(task, plan, time.Date(2026, 9, 16, 8, 0, 0, 0, time.UTC))
	runtime.record = fixtureRuntime(lineage.execution, plan, execution.NodeSucceeded)

	if err := activity.Execute(context.Background(), executionStepInput(task)); err != nil {
		t.Fatalf("Execute replay returned error: %v", err)
	}
	if worker.calls != 0 {
		t.Fatalf("succeeded replay invoked worker %d times", worker.calls)
	}
	if lineage.created != 0 {
		t.Fatalf("succeeded replay recreated Execution %d times", lineage.created)
	}
}

func TestDurableExecutionActivityUsesExactFrozenBinding(t *testing.T) {
	activity, task, plan, runtime, worker := executionActivityFixture(t)
	runtime.getMissing = true

	if err := activity.Execute(context.Background(), executionStepInput(task)); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if worker.calls != 1 || runtime.registers != 1 {
		t.Fatalf("worker=%d registers=%d want 1/1", worker.calls, runtime.registers)
	}
	binding := activity.Selection.(*executionSelectionStub).result
	if worker.lastExec.PlanRef != binding.Plan || worker.lastNode.Node.ID != binding.Node {
		t.Fatalf("worker did not receive frozen plan/node: exec=%+v node=%s binding=%+v/%s", worker.lastExec.PlanRef, worker.lastNode.Node.ID, binding.Plan, binding.Node)
	}
	if worker.lastNode.Policy.ID != binding.Policy.ID || worker.lastNode.Target.ID != binding.Target.ID || worker.lastNode.Target.Snapshot.Digest != binding.Target.Snapshot.Digest {
		t.Fatalf("worker binding drifted: %+v", worker.lastNode)
	}
	if worker.lastNode.Estimate.Cost != binding.Decision.Selected.EstimatedCost || worker.lastNode.Estimate.NodeAttempts != 1 || worker.lastNode.Estimate.FrontierCalls != 1 {
		t.Fatalf("unexpected frozen estimate: %+v", worker.lastNode.Estimate)
	}
	if plan.ID != binding.Plan.PlanID {
		t.Fatalf("fixture plan mismatch: %s vs %s", plan.ID, binding.Plan.PlanID)
	}
}

func TestDurableExecutionActivityRequeuesRetryableReadOnlyFailure(t *testing.T) {
	activity, task, plan, runtime, worker := executionActivityFixture(t)
	lineage := activity.Lineage.(*executionLineageStub)
	lineage.execution = fixtureExecution(task, plan, time.Date(2026, 9, 16, 8, 0, 0, 0, time.UTC))
	runtime.record = fixtureRuntime(lineage.execution, plan, execution.NodeFailed)
	runtime.record.AttemptNumber = 1
	runtime.attempts = []execution.AttemptRecord{{
		Attempt: execution.ExecutionAttempt{
			ID:            execution.AttemptID("attempt-1"),
			ExecutionRef:  lineage.execution.ID,
			NodeRef:       plan.Nodes[0].ID,
			AttemptNumber: 1,
			Status:        execution.AttemptFailed,
			ErrorClass:    execution.ErrorTransient,
		},
		PlanRevision: plan.Revision,
	}}

	if err := activity.Execute(context.Background(), executionStepInput(task)); err != nil {
		t.Fatalf("retry Execute returned error: %v", err)
	}
	if runtime.requeues != 1 || worker.calls != 1 {
		t.Fatalf("retryable failure requeues=%d worker=%d want 1/1", runtime.requeues, worker.calls)
	}
}

func TestDurableExecutionActivityDoesNotRetryPermanentFailure(t *testing.T) {
	activity, task, plan, runtime, worker := executionActivityFixture(t)
	lineage := activity.Lineage.(*executionLineageStub)
	lineage.execution = fixtureExecution(task, plan, time.Date(2026, 9, 16, 8, 0, 0, 0, time.UTC))
	runtime.record = fixtureRuntime(lineage.execution, plan, execution.NodeFailed)
	runtime.record.AttemptNumber = 1
	runtime.attempts = []execution.AttemptRecord{{
		Attempt: execution.ExecutionAttempt{
			ID:            execution.AttemptID("attempt-1"),
			ExecutionRef:  lineage.execution.ID,
			NodeRef:       plan.Nodes[0].ID,
			AttemptNumber: 1,
			Status:        execution.AttemptFailed,
			ErrorClass:    execution.ErrorPermanent,
		},
		PlanRevision: plan.Revision,
	}}

	err := activity.Execute(context.Background(), executionStepInput(task))
	if err == nil {
		t.Fatal("expected retry-not-allowed error")
	}
	var applicationErr *temporal.ApplicationError
	if !errors.As(err, &applicationErr) || applicationErr.Type() != ErrorTypeExecutionRetryNotAllowed {
		t.Fatalf("error=%v want Temporal %s", err, ErrorTypeExecutionRetryNotAllowed)
	}
	if runtime.requeues != 0 || worker.calls != 0 {
		t.Fatalf("permanent failure requeues=%d worker=%d want 0/0", runtime.requeues, worker.calls)
	}
}

func executionActivityFixture(t *testing.T) (DurableExecutionActivity, domain.Task, execution.ExecutionPlan, *executionRuntimeStub, *executionWorkerStub) {
	t.Helper()
	fixed := time.Date(2026, 9, 16, 8, 0, 0, 0, time.UTC)
	task := domain.Task{
		ID: "task-execute", TenantID: "tenant-a", ProjectID: "project-a", CreatedBy: "user-a",
		Input: "diagnose production latency", Status: domain.TaskExecuting, Version: 5,
		Currency: "USD", TraceID: "trace-execute", CreatedAt: fixed.Add(-time.Minute), UpdatedAt: fixed,
	}
	goal := execution.Goal{
		ID: execution.GoalID("goal/task-execute/v1"), TaskRef: task.ID, Objective: task.Input,
		Budget: execution.BudgetLimit{MaxCost: 5, MaxNodeAttempts: 3, MaxFrontierCalls: 3},
	}
	plan := execution.ExecutionPlan{
		ID: execution.PlanID("plan/task-execute/model/v1"), GoalRef: goal.ID, Revision: 1,
		Nodes: []execution.ExecutionNode{{
			ID: execution.NodeID("model/main"), Type: execution.NodeModel,
			RetryPolicy: execution.RetryPolicy{MaxAttempts: 3, RetryOn: []execution.ErrorClass{execution.ErrorTransient, execution.ErrorTargetUnavailable, execution.ErrorTimeout}},
			Effect:      execution.EffectSpec{Class: execution.EffectPure, RetrySafe: true},
		}},
		CreatedAt: fixed.Add(-30 * time.Second),
	}
	if err := execution.ValidatePlan(plan); err != nil {
		t.Fatalf("fixture plan invalid: %v", err)
	}
	binding := repository.RouteSelectionResult{
		Plan: execution.PlanRevisionRef{PlanID: plan.ID, Revision: plan.Revision}, Node: plan.Nodes[0].ID,
		Decision: domain.RouteDecision{
			ID: "route-execute", TaskID: task.ID,
			Selected:  domain.RouteCandidate{ModelID: "model-a", ModelVersion: "v1", RouteClass: domain.RouteEfficient, EstimatedCost: 0.04},
			CreatedAt: fixed.Add(-10 * time.Second),
		},
		Policy: execution.PolicyDecisionRecord{
			ID: execution.PolicyDecisionID("policy-execute"), PlanRevision: plan.Revision, NodeRef: plan.Nodes[0].ID, Decision: execution.PolicyAllow,
		},
		Target: execution.ExecutionTarget{
			ID: execution.TargetID("model/model-a@v1"), Revision: 1, Type: execution.TargetModel,
			Snapshot: execution.TargetSnapshot{Digest: "sha256:frozen-target", ModelVersion: "v1"},
		},
	}
	runtime := &executionRuntimeStub{}
	worker := &executionWorkerStub{runtime: runtime}
	activity := DurableExecutionActivity{
		Tasks:     &executionTaskRepository{task: task},
		Lineage:   &executionLineageStub{goal: goal, plan: plan},
		Selection: &executionSelectionStub{result: binding},
		Runtime:   runtime,
		Worker:    worker,
		Now:       func() time.Time { return fixed },
	}
	return activity, task, plan, runtime, worker
}

func fixtureExecution(task domain.Task, plan execution.ExecutionPlan, at time.Time) execution.Execution {
	return execution.Execution{
		ID:      execution.ExecutionID("execution/" + task.ID + "/" + string(plan.ID) + "@1/" + ExecutionActivityVersion),
		TaskRef: task.ID, GoalRef: plan.GoalRef,
		PlanRef:   execution.PlanRevisionRef{PlanID: plan.ID, Revision: plan.Revision},
		Identity:  execution.Identity{Principal: "system:temporal-task-lifecycle", Tenant: task.TenantID},
		Status:    execution.ExecutionStatus{Phase: execution.ExecutionRunning},
		CreatedAt: at, UpdatedAt: at,
	}
}

func fixtureRuntime(item execution.Execution, plan execution.ExecutionPlan, state execution.NodeState) execution.NodeRuntimeRecord {
	node := plan.Nodes[0]
	return execution.NodeRuntimeRecord{
		ExecutionRef: item.ID, PlanRevision: plan.Revision, NodeRef: node.ID, State: state,
		EffectClass: node.Effect.Class, RetrySafe: node.Effect.RetrySafe, IdempotencyKey: node.Effect.IdempotencyKey,
	}
}

func executionStepInput(task domain.Task) StepInput {
	return StepInput{TenantID: task.TenantID, ProjectID: task.ProjectID, TaskID: task.ID, TraceID: task.TraceID}
}
