package workflow

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	execution "github.com/tommyxie2026-tech/aicloud/execution"
	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
	"github.com/tommyxie2026-tech/aicloud/internal/router"
	"go.temporal.io/sdk/temporal"
)

type routingTaskRepository struct {
	task domain.Task
}

func (r *routingTaskRepository) List(context.Context) ([]domain.Task, error) {
	return []domain.Task{r.task}, nil
}

func (r *routingTaskRepository) Get(_ context.Context, id string) (domain.Task, error) {
	if r.task.ID != id {
		return domain.Task{}, repository.ErrNotFound
	}
	return r.task, nil
}

func (r *routingTaskRepository) Create(_ context.Context, task domain.Task) (domain.Task, error) {
	r.task = task
	return task, nil
}

func (r *routingTaskRepository) Update(_ context.Context, task domain.Task) (domain.Task, error) {
	r.task = task
	return task, nil
}

type routingLineageStub struct {
	plans []execution.ExecutionPlan
	goal  execution.Goal
}

func (s *routingLineageStub) ListPlansByTask(context.Context, string) ([]execution.ExecutionPlan, error) {
	return append([]execution.ExecutionPlan(nil), s.plans...), nil
}

func (s *routingLineageStub) GetGoal(_ context.Context, id execution.GoalID) (execution.Goal, error) {
	if s.goal.ID != id {
		return execution.Goal{}, repository.ErrNotFound
	}
	return s.goal, nil
}

type routingPolicyStub struct {
	calls    int
	decision execution.PolicyDecision
}

func (s *routingPolicyStub) Evaluate(_ context.Context, task domain.Task, _ execution.Goal, plan execution.ExecutionPlan, node execution.ExecutionNode) (execution.PolicyDecisionRecord, error) {
	s.calls++
	return execution.PolicyDecisionRecord{
		ID:            execution.PolicyDecisionID("policy-test"),
		PolicySetRef:  "test",
		PolicyVersion: "policy-test-v1",
		PlanRevision:  plan.Revision,
		NodeRef:       node.ID,
		Principal:     "test",
		Action:        "route:model-target",
		Resource:      task.ID,
		Decision:      s.decision,
	}, nil
}

type routingPlannerStub struct {
	calls    int
	decision domain.RouteDecision
	last     router.Request
}

func (s *routingPlannerStub) Plan(_ context.Context, request router.Request) (domain.RouteDecision, error) {
	s.calls++
	s.last = request
	decision := s.decision
	if decision.TaskID == "" {
		decision.TaskID = request.TaskID
	}
	return decision, nil
}

type routeSelectionStub struct {
	replay       repository.RouteSelectionResult
	found        bool
	resolveCalls int
	commitCalls  int
	committed    repository.RouteSelectionCommit
}

func (s *routeSelectionStub) ResolveRouteSelection(context.Context, repository.IdempotencyLookup) (repository.RouteSelectionResult, bool, error) {
	s.resolveCalls++
	return s.replay, s.found, nil
}

func (s *routeSelectionStub) CommitRouteSelection(_ context.Context, command repository.RouteSelectionCommit) (repository.RouteSelectionResult, error) {
	s.commitCalls++
	s.committed = command
	return repository.RouteSelectionResult{
		Task:        command.Task,
		Decision:    command.Decision,
		Target:      command.Target,
		Event:       command.Event,
		Idempotency: command.Idempotency,
	}, nil
}

func TestDurableRoutingActivityReplaySkipsVolatilePolicyAndRouter(t *testing.T) {
	activity, task, _, _ := routingActivityFixture(t)
	policy := activity.Policy.(*routingPolicyStub)
	planner := activity.Planner.(*routingPlannerStub)
	selection := activity.Selection.(*routeSelectionStub)
	selection.found = true
	selection.replay = repository.RouteSelectionResult{
		Decision: domain.RouteDecision{ID: "route-frozen", TaskID: task.ID},
		Target: execution.ExecutionTarget{
			ID: execution.TargetID("model/model-a@v1"),
			Snapshot: execution.TargetSnapshot{Digest: "sha256:frozen"},
		},
		Replayed: true,
	}

	if err := activity.Route(context.Background(), routingStepInput(task)); err != nil {
		t.Fatalf("Route replay returned error: %v", err)
	}
	if selection.resolveCalls != 1 || selection.commitCalls != 0 {
		t.Fatalf("replay must resolve once and never commit: resolve=%d commit=%d", selection.resolveCalls, selection.commitCalls)
	}
	if policy.calls != 0 || planner.calls != 0 {
		t.Fatalf("replay must skip volatile policy/router: policy=%d router=%d", policy.calls, planner.calls)
	}
}

func TestDurableRoutingActivityPolicyDenySkipsRouterAndCommit(t *testing.T) {
	activity, task, _, _ := routingActivityFixture(t)
	policy := activity.Policy.(*routingPolicyStub)
	policy.decision = execution.PolicyDeny
	planner := activity.Planner.(*routingPlannerStub)
	selection := activity.Selection.(*routeSelectionStub)

	err := activity.Route(context.Background(), routingStepInput(task))
	if err == nil {
		t.Fatal("expected policy denial")
	}
	var applicationErr *temporal.ApplicationError
	if !errors.As(err, &applicationErr) || applicationErr.Type() != ErrorTypeRoutePolicyDenied {
		t.Fatalf("error=%v want Temporal %s", err, ErrorTypeRoutePolicyDenied)
	}
	if policy.calls != 1 || planner.calls != 0 || selection.commitCalls != 0 {
		t.Fatalf("deny must stop before router/commit: policy=%d router=%d commit=%d", policy.calls, planner.calls, selection.commitCalls)
	}
}

func TestDurableRoutingActivityAllowFreezesTargetAndCommits(t *testing.T) {
	activity, task, plan, fixed := routingActivityFixture(t)
	planner := activity.Planner.(*routingPlannerStub)
	planner.decision = domain.RouteDecision{
		ID:     "route-test",
		TaskID: task.ID,
		Selected: domain.RouteCandidate{
			ModelID: "model-a", ModelVersion: "v1", RouteClass: domain.RouteEfficient, EstimatedCost: 0.02,
		},
		Candidates: []domain.RouteCandidate{{ModelID: "model-a", ModelVersion: "v1"}},
		Reason:     "test selection",
		CreatedAt:  fixed,
	}
	selection := activity.Selection.(*routeSelectionStub)

	if err := activity.Route(context.Background(), routingStepInput(task)); err != nil {
		t.Fatalf("Route returned error: %v", err)
	}
	if planner.calls != 1 || selection.commitCalls != 1 {
		t.Fatalf("allow must route and commit exactly once: router=%d commit=%d", planner.calls, selection.commitCalls)
	}
	if planner.last.PolicyVersion != "policy-test-v1" {
		t.Fatalf("router request policy version=%q", planner.last.PolicyVersion)
	}
	if planner.last.EvidenceVersion != "plan:"+string(plan.ID)+"@1" {
		t.Fatalf("router request evidence version=%q", planner.last.EvidenceVersion)
	}
	committed := selection.committed
	if committed.Decision.ID != "route-test" || committed.Target.ID == "" {
		t.Fatalf("route selection evidence incomplete: %+v", committed)
	}
	if !strings.HasPrefix(committed.Target.Snapshot.Digest, "sha256:") || len(committed.Target.Snapshot.Digest) <= len("sha256:") {
		t.Fatalf("target digest must be frozen SHA-256 evidence: %q", committed.Target.Snapshot.Digest)
	}
	if committed.Event.EventType != "TaskRouteSelected" || len(committed.Event.Payload) == 0 {
		t.Fatalf("route selection TaskEvent incomplete: %+v", committed.Event)
	}
	if committed.Idempotency.Key == "" || !strings.HasPrefix(committed.Idempotency.RequestDigest, "sha256:") {
		t.Fatalf("route selection idempotency evidence incomplete: %+v", committed.Idempotency)
	}
}

func TestBaselineRoutingPolicyDeniesMutation(t *testing.T) {
	activity, task, plan, _ := routingActivityFixture(t)
	goal, err := activity.Lineage.GetGoal(context.Background(), plan.GoalRef)
	if err != nil {
		t.Fatal(err)
	}
	node := plan.Nodes[0]
	node.Effect = execution.EffectSpec{
		Class:          execution.EffectIdempotentMutation,
		IdempotencyKey: "mutation-test",
		RetrySafe:      true,
	}
	decision, err := (BaselineRoutingPolicy{}).Evaluate(context.Background(), task, goal, plan, node)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Decision != execution.PolicyDeny {
		t.Fatalf("mutation decision=%s want=%s", decision.Decision, execution.PolicyDeny)
	}
}

func routingActivityFixture(t *testing.T) (DurableRoutingActivity, domain.Task, execution.ExecutionPlan, time.Time) {
	t.Helper()
	fixed := time.Date(2026, 9, 16, 7, 30, 0, 0, time.UTC)
	task := domain.Task{
		ID:        "task-route",
		TenantID:  "tenant-a",
		ProjectID: "project-a",
		CreatedBy: "user-a",
		Input:     "diagnose production latency",
		Status:    domain.TaskRouting,
		Version:   3,
		Currency:  "USD",
		TraceID:   "trace-route",
		CreatedAt: fixed.Add(-time.Minute),
		UpdatedAt: fixed.Add(-time.Second),
	}
	goal := execution.Goal{
		ID:        execution.GoalID("goal/task-route/v1"),
		TaskRef:   task.ID,
		Objective: task.Input,
		Budget:    execution.BudgetLimit{MaxCost: 5},
	}
	plan := execution.ExecutionPlan{
		ID:       execution.PlanID("plan/task-route/model/v1"),
		GoalRef:  goal.ID,
		Revision: 1,
		Nodes: []execution.ExecutionNode{{
			ID:   execution.NodeID("model/main"),
			Type: execution.NodeModel,
			Effect: execution.EffectSpec{
				Class:     execution.EffectPure,
				RetrySafe: true,
			},
		}},
		CreatedAt: fixed.Add(-30 * time.Second),
	}
	if err := execution.ValidatePlan(plan); err != nil {
		t.Fatalf("fixture plan invalid: %v", err)
	}
	policy := &routingPolicyStub{decision: execution.PolicyAllow}
	planner := &routingPlannerStub{}
	selection := &routeSelectionStub{}
	activity := DurableRoutingActivity{
		Tasks:          &routingTaskRepository{task: task},
		Lineage:        &routingLineageStub{plans: []execution.ExecutionPlan{plan}, goal: goal},
		Policy:         policy,
		Planner:        planner,
		Selection:      selection,
		RequestBuilder: BaselineRouteRequestBuilder{},
		Now:            func() time.Time { return fixed },
	}
	return activity, task, plan, fixed
}

func routingStepInput(task domain.Task) StepInput {
	return StepInput{
		TenantID:  task.TenantID,
		ProjectID: task.ProjectID,
		TaskID:    task.ID,
		TraceID:   task.TraceID,
	}
}
