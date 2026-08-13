package router

import (
	"context"
	"testing"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
)

func TestPlanUsesModelVersionAndDeploymentRuntime(t *testing.T) {
	now := time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)
	version := domain.ModelVersion{
		ID: "version-1", Name: "model", Version: "v1",
		Lifecycle: domain.ModelActive, ApprovalStatus: domain.ApprovalApproved,
		Capabilities: []string{"coding"},
	}
	blocked := domain.Deployment{
		ID: "deployment-blocked", ModelVersionID: version.ID,
		Lifecycle: domain.DeploymentBlocked, RoutingEligible: false,
		Health: domain.HealthHealthy, HealthCheckedAt: &now,
		CapacityAvailable: 10, QuotaRemaining: 10,
		Pricing: domain.PricingProfile{Currency: "USD", InputPerMillion: 0.1, OutputPerMillion: 0.1},
	}
	ready := domain.Deployment{
		ID: "deployment-ready", ModelVersionID: version.ID,
		Lifecycle: domain.DeploymentReady, RoutingEligible: true,
		Health: domain.HealthHealthy, HealthCheckedAt: &now,
		CapacityAvailable: 10, QuotaRemaining: 10,
		Pricing: domain.PricingProfile{Currency: "USD", InputPerMillion: 2, OutputPerMillion: 4},
	}

	r := New(repository.NewMemoryModels(), nil).
		WithModelVersions(repository.NewMemoryModelVersions(version)).
		WithDeployments(repository.NewMemoryDeployments(blocked, ready))
	r.now = func() time.Time { return now }

	decision, err := r.Plan(context.Background(), Request{
		TaskID: "task-version", RouteClass: domain.RouteEfficient,
		RequiredCapabilities: []string{"coding"},
		RequireFreshSignals: true, SignalMaxAge: time.Minute,
		EstimatedInputTokens: 1000, EstimatedOutputTokens: 1000,
	})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}
	if decision.Selected.ModelID != version.ID || decision.Selected.DeploymentID != ready.ID {
		t.Fatalf("unexpected selected route: %#v", decision.Selected)
	}
}

func TestPlanRejectsStaleModelVersionDeployment(t *testing.T) {
	now := time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)
	old := now.Add(-10 * time.Minute)
	version := domain.ModelVersion{
		ID: "version-stale", Version: "v1", Lifecycle: domain.ModelActive,
		ApprovalStatus: domain.ApprovalApproved,
	}
	deployment := domain.Deployment{
		ID: "deployment-stale", ModelVersionID: version.ID,
		Lifecycle: domain.DeploymentReady, RoutingEligible: true,
		Health: domain.HealthHealthy, HealthCheckedAt: &old,
		CapacityAvailable: 10, QuotaRemaining: 10,
	}
	r := New(repository.NewMemoryModels(), nil).
		WithModelVersions(repository.NewMemoryModelVersions(version)).
		WithDeployments(repository.NewMemoryDeployments(deployment))
	r.now = func() time.Time { return now }

	_, err := r.Plan(context.Background(), Request{
		TaskID: "task-stale", RequireFreshSignals: true, SignalMaxAge: time.Minute,
	})
	if err == nil {
		t.Fatal("expected stale deployment to be rejected")
	}
}
