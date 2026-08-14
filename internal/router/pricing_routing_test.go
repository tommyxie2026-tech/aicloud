package router

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
)

func TestPlanSelectsDeploymentByVersionedTaskCost(t *testing.T) {
	now := time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)
	model := domain.Model{
		ID: "model-1", Version: "v1", Lifecycle: domain.ModelActive,
		ApprovalStatus: domain.ApprovalApproved, Capabilities: []string{"coding"},
	}
	first := domain.Deployment{
		ID: "deployment-expensive", ModelID: model.ID, ModelVersion: model.Version,
		Lifecycle: domain.DeploymentReady, RoutingEligible: true,
		Health: domain.HealthHealthy, HealthCheckedAt: &now,
		CapacityAvailable: 10, QuotaRemaining: 10,
	}
	second := domain.Deployment{
		ID: "deployment-efficient", ModelID: model.ID, ModelVersion: model.Version,
		Lifecycle: domain.DeploymentReady, RoutingEligible: true,
		Health: domain.HealthHealthy, HealthCheckedAt: &now,
		CapacityAvailable: 10, QuotaRemaining: 10,
	}
	pricing := repository.NewMemoryPricingPolicies()
	for _, policy := range []domain.PricingPolicy{
		{
			ID: "price-expensive", Version: "v1", DeploymentID: first.ID, Currency: "USD",
			InputPerMillion: 10, OutputPerMillion: 20, EffectiveFrom: now.Add(-time.Hour), CreatedAt: now.Add(-time.Hour),
		},
		{
			ID: "price-efficient", Version: "v3", DeploymentID: second.ID, Currency: "USD",
			InputPerMillion: 1, OutputPerMillion: 2, EffectiveFrom: now.Add(-time.Hour), CreatedAt: now.Add(-time.Hour),
		},
	} {
		if _, err := pricing.Create(context.Background(), policy); err != nil {
			t.Fatalf("create pricing policy: %v", err)
		}
	}
	r := New(repository.NewMemoryModels(model), nil).
		WithDeployments(repository.NewMemoryDeployments(first, second)).
		WithPricingPolicies(pricing)
	r.now = func() time.Time { return now }

	decision, err := r.Plan(context.Background(), Request{
		TaskID: "task-price", RouteClass: domain.RouteEfficient,
		RequiredCapabilities: []string{"coding"},
		EstimatedInputTokens: 1_000_000, EstimatedOutputTokens: 100_000,
	})
	if err != nil {
		t.Fatalf("Plan returned error: %v", err)
	}
	if decision.Selected.DeploymentID != second.ID {
		t.Fatalf("selected deployment = %q, want %q", decision.Selected.DeploymentID, second.ID)
	}
	if decision.Selected.EstimatedCost != 1.2 {
		t.Fatalf("estimated cost = %f, want 1.2", decision.Selected.EstimatedCost)
	}
	if !strings.Contains(decision.Reason, "price-efficient@v3") {
		t.Fatalf("route rationale does not retain pricing version: %q", decision.Reason)
	}
}
