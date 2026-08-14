package router

import (
	"context"
	"testing"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
)

func TestDeploymentCandidate(t *testing.T) {
	now := time.Now().UTC()
	m := domain.Model{
		ID: "m1", Version: "v1", Lifecycle: domain.ModelActive,
		ApprovalStatus: domain.ApprovalApproved,
	}
	d := domain.Deployment{
		ID: "d1", ModelID: "m1", ModelVersion: "v1",
		Lifecycle: domain.DeploymentReady, RoutingEligible: true,
		Health: domain.HealthHealthy, HealthCheckedAt: &now,
		CapacityAvailable: 1, QuotaRemaining: 1,
	}
	got := DeploymentCandidate(m, d, Request{RequireFreshSignals: true, SignalMaxAge: time.Minute}, now)
	if got.DeploymentID != "d1" || len(got.RejectionReasons) != 0 {
		t.Fatalf("unexpected candidate: %+v", got)
	}

	old := now.Add(-2 * time.Minute)
	d.HealthCheckedAt = &old
	got = DeploymentCandidate(m, d, Request{RequireFreshSignals: true, SignalMaxAge: time.Minute}, now)
	if len(got.RejectionReasons) == 0 {
		t.Fatal("expected stale deployment to be ineligible")
	}
}

func TestPlanSelectsEligibleDeployment(t *testing.T) {
	now := time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)
	model := domain.Model{
		ID: "model-1", Version: "v1", Lifecycle: domain.ModelActive,
		ApprovalStatus: domain.ApprovalApproved, Capabilities: []string{"coding"},
	}
	first := domain.Deployment{
		ID: "deployment-1", ModelID: model.ID, ModelVersion: model.Version,
		Lifecycle: domain.DeploymentReady, RoutingEligible: true,
		Health: domain.HealthUnhealthy, HealthCheckedAt: &now,
		CapacityAvailable: 10, QuotaRemaining: 10,
	}
	second := domain.Deployment{
		ID: "deployment-2", ModelID: model.ID, ModelVersion: model.Version,
		Lifecycle: domain.DeploymentReady, RoutingEligible: true,
		Health: domain.HealthHealthy, HealthCheckedAt: &now,
		CapacityAvailable: 10, QuotaRemaining: 10,
	}
	r := New(repository.NewMemoryModels(model), nil).
		WithDeployments(repository.NewMemoryDeployments(first, second))
	r.now = func() time.Time { return now }

	decision, err := r.Plan(context.Background(), Request{
		TaskID: "task-1", RouteClass: domain.RouteEfficient,
		RequiredCapabilities: []string{"coding"},
		RequireFreshSignals:  true, SignalMaxAge: time.Minute,
	})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}
	if decision.Selected.DeploymentID != second.ID {
		t.Fatalf("selected deployment = %q, want %q", decision.Selected.DeploymentID, second.ID)
	}
}

func TestPlanDeterministicWithDeploymentRegistry(t *testing.T) {
	r := New(repository.NewMemoryModels(), nil).
		WithDeployments(repository.NewMemoryDeployments())
	decision, err := r.Plan(context.Background(), Request{TaskID: "task-d", RouteClass: domain.RouteDeterministic})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}
	if decision.Selected.RouteClass != domain.RouteDeterministic {
		t.Fatalf("unexpected route class: %s", decision.Selected.RouteClass)
	}
}
