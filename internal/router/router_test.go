package router

import (
	"context"
	"testing"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
)

func TestRouterSelectsApprovedLowerCostHealthyModel(t *testing.T) {
	now := time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)
	models := repository.NewMemoryModels(
		model("efficient", 1, now),
		model("flagship", 20, now),
	)
	decisions := repository.NewMemoryRouteDecisions()
	r := New(models, decisions)
	r.now = func() time.Time { return now }

	decision, err := r.Decide(context.Background(), Request{
		TaskID:               "task-1",
		RouteClass:            domain.RouteEfficient,
		RequiredCapabilities: []string{"coding"},
		EstimatedInputTokens:  1000,
		EstimatedOutputTokens: 1000,
		Budget:                1,
		RequireFreshSignals:   true,
	})
	if err != nil {
		t.Fatalf("Decide returned error: %v", err)
	}
	if decision.Selected.ModelID != "efficient" {
		t.Fatalf("selected model = %s", decision.Selected.ModelID)
	}
	stored, err := decisions.Get(context.Background(), decision.ID)
	if err != nil || stored.TaskID != "task-1" {
		t.Fatalf("route decision was not persisted: %#v %v", stored, err)
	}
}

func TestRouterRejectsUnapprovedAndOverBudgetModels(t *testing.T) {
	now := time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)
	unapproved := model("unapproved", 1, now)
	unapproved.ApprovalStatus = domain.ApprovalReviewed
	expensive := model("expensive", 1000, now)
	r := New(repository.NewMemoryModels(unapproved, expensive), nil)
	r.now = func() time.Time { return now }

	_, err := r.Decide(context.Background(), Request{
		TaskID:               "task-2",
		RequiredCapabilities: []string{"coding"},
		EstimatedInputTokens:  1_000_000,
		EstimatedOutputTokens: 1_000_000,
		Budget:                1,
		RequireFreshSignals:   true,
	})
	if err == nil {
		t.Fatal("expected routing to fail")
	}
}

func TestRouterSupportsDeterministicRoute(t *testing.T) {
	r := New(repository.NewMemoryModels(), repository.NewMemoryRouteDecisions())
	decision, err := r.Decide(context.Background(), Request{TaskID: "task-3", RouteClass: domain.RouteDeterministic})
	if err != nil {
		t.Fatalf("Decide returned error: %v", err)
	}
	if decision.Selected.RouteClass != domain.RouteDeterministic || decision.Selected.ModelID != "" {
		t.Fatalf("unexpected deterministic decision: %#v", decision.Selected)
	}
}

func model(id string, price float64, checkedAt time.Time) domain.Model {
	return domain.Model{
		ID:                id,
		Name:              id,
		Version:           "v1",
		Provider:          "test",
		Lifecycle:         domain.ModelActive,
		Capabilities:      []string{"coding"},
		Pricing:           domain.PricingProfile{Currency: "USD", InputPerMillion: price, OutputPerMillion: price},
		Health:            domain.HealthHealthy,
		HealthCheckedAt:   &checkedAt,
		QuotaRemaining:    100,
		CapacityAvailable: 10,
		ApprovalStatus:    domain.ApprovalApproved,
	}
}
