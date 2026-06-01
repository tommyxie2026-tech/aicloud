package routing

import (
	"testing"

	"github.com/tommyxie2026-tech/aicloud/model/mock"
	"github.com/tommyxie2026-tech/aicloud/model/provider"
)

func TestStaticRouterSelectsMockFallbackForMediumPlan(t *testing.T) {
	router := NewStaticRouter(
		[]provider.ModelProvider{mock.NewProvider()},
		map[string]ProviderScore{
			"mock": {ProviderName: "mock", AverageScore: 95, SafetyFailures: 0, SchemaFailures: 0},
		},
		DefaultRoutingPolicy(),
	)

	decision, err := router.Route(RouteRequest{
		RequestID:       "route-test-001",
		TaskType:        TaskGeneratePlan,
		RiskHint:        RiskMedium,
		Environment:     EnvironmentDev,
		DataSensitivity: DataSensitivityInternal,
	})
	if err != nil {
		t.Fatalf("Route returned error: %v", err)
	}
	if decision == nil {
		t.Fatalf("expected decision, got nil")
	}
	if decision.Blocked {
		t.Fatalf("expected route to be allowed, got blocked: %s", decision.BlockedReason)
	}
	if decision.SelectedProvider != "mock" {
		t.Fatalf("expected selected provider mock, got %s", decision.SelectedProvider)
	}
}

func TestStaticRouterBlocksRestrictedData(t *testing.T) {
	router := NewStaticRouter(
		[]provider.ModelProvider{mock.NewProvider()},
		map[string]ProviderScore{"mock": {ProviderName: "mock", AverageScore: 95}},
		DefaultRoutingPolicy(),
	)

	decision, err := router.Route(RouteRequest{
		RequestID:       "route-test-002",
		TaskType:        TaskGeneratePlan,
		RiskHint:        RiskMedium,
		Environment:     EnvironmentDev,
		DataSensitivity: DataSensitivityRestricted,
	})
	if err == nil {
		t.Fatalf("expected restricted data error")
	}
	if decision == nil || !decision.Blocked {
		t.Fatalf("expected blocked decision, got %#v", decision)
	}
}

func TestStaticRouterRoutesRiskClassificationToDeterministicPolicy(t *testing.T) {
	router := NewStaticRouter(nil, nil, DefaultRoutingPolicy())

	decision, err := router.Route(RouteRequest{
		RequestID:       "route-test-003",
		TaskType:        TaskRiskClassification,
		RiskHint:        RiskMedium,
		Environment:     EnvironmentDev,
		DataSensitivity: DataSensitivityInternal,
	})
	if err != nil {
		t.Fatalf("Route returned error: %v", err)
	}
	if decision.SelectedProvider != "deterministic-policy" {
		t.Fatalf("expected deterministic-policy, got %s", decision.SelectedProvider)
	}
}

func TestStaticRouterRequiresEvaluation(t *testing.T) {
	router := NewStaticRouter(
		[]provider.ModelProvider{mock.NewProvider()},
		nil,
		DefaultRoutingPolicy(),
	)

	decision, err := router.Route(RouteRequest{
		RequestID:       "route-test-004",
		TaskType:        TaskGeneratePlan,
		RiskHint:        RiskMedium,
		Environment:     EnvironmentDev,
		DataSensitivity: DataSensitivityInternal,
	})
	if err == nil {
		t.Fatalf("expected no safe provider error when evaluation score is missing")
	}
	if decision == nil || !decision.Blocked {
		t.Fatalf("expected blocked decision, got %#v", decision)
	}
}
