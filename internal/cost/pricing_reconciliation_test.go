package cost_test

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/cost"
	"github.com/tommyxie2026-tech/aicloud/internal/domain"
	"github.com/tommyxie2026-tech/aicloud/internal/repository"
	"github.com/tommyxie2026-tech/aicloud/model/provider"
)

func TestReconciliationUsesExactRouteTimePricingVersion(t *testing.T) {
	ctx := context.Background()
	routeTime := time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)
	policies := repository.NewMemoryPricingPolicies()
	v1 := domain.PricingPolicy{
		ID: "price-deployment", Version: "v1", DeploymentID: "deployment-1", Currency: "USD",
		InputPerMillion: 2, OutputPerMillion: 10,
		InferenceEffortFactors: map[domain.InferenceEffort]float64{domain.EffortHigh: 1.5},
		EffectiveFrom: routeTime.Add(-time.Hour), EffectiveTo: timePointer(routeTime.Add(time.Hour)),
		Digest: "sha256:v1", CreatedAt: routeTime.Add(-time.Hour),
	}
	v2 := domain.PricingPolicy{
		ID: "price-deployment", Version: "v2", DeploymentID: "deployment-1", Currency: "USD",
		InputPerMillion: 20, OutputPerMillion: 100,
		InferenceEffortFactors: map[domain.InferenceEffort]float64{domain.EffortHigh: 1.5},
		EffectiveFrom: routeTime.Add(time.Hour), Digest: "sha256:v2", CreatedAt: routeTime.Add(time.Hour),
	}
	for _, policy := range []domain.PricingPolicy{v1, v2} {
		if _, err := policies.Create(ctx, policy); err != nil {
			t.Fatalf("create pricing policy: %v", err)
		}
	}

	events := repository.NewMemoryCostEvents()
	ledger := cost.New(events)
	evidence := domain.RoutePricingEvidence{
		RouteDecisionID: "route-1", DeploymentID: "deployment-1",
		PolicyID: v1.ID, PolicyVersion: v1.Version, PolicyDigest: v1.Digest,
		Quote: domain.PricingQuote{
			PolicyID: v1.ID, PolicyVersion: v1.Version, PolicyDigest: v1.Digest,
			DeploymentID: v1.DeploymentID, Currency: v1.Currency, QuotedAt: routeTime,
		},
		Selected: true, CreatedAt: routeTime,
	}
	actual, err := ledger.RecordReconciledModelUsage(ctx, cost.ModelUsage{
		TaskID: "task-1", TraceID: "trace-1", Provider: "provider-1",
		ModelID: "model-1", ModelVersion: "m1", DeploymentID: "deployment-1",
		Usage: provider.TokenUsage{InputTokens: 1_000_000, OutputTokens: 100_000, TotalTokens: 1_100_000},
		Attempt: 1,
	}, evidence, policies, domain.EffortHigh)
	if err != nil {
		t.Fatalf("RecordReconciledModelUsage returned error: %v", err)
	}
	var total float64
	for _, event := range actual {
		total += event.Amount
		if event.Metadata["pricing_policy_version"] != "v1" || event.Metadata["pricing_policy_digest"] != "sha256:v1" {
			t.Fatalf("cost event lost route-time pricing identity: %#v", event.Metadata)
		}
	}
	if math.Abs(total-4.5) > 0.000001 {
		t.Fatalf("actual reconciled cost = %f, want 4.5 from v1 economics", total)
	}
}

func TestAggregateTaskIncludesRetryAndFallbackAttempts(t *testing.T) {
	ctx := context.Background()
	events := repository.NewMemoryCostEvents()
	for _, event := range []domain.CostEvent{
		{ID: "cost-a", TaskID: "task-retry", TraceID: "trace-1", Component: domain.CostModelInput, Amount: 1.25, Currency: "USD", Attempt: 1, CreatedAt: time.Unix(1, 0)},
		{ID: "cost-b", TaskID: "task-retry", TraceID: "trace-1", Component: domain.CostModelOutput, Amount: 2.75, Currency: "USD", Attempt: 2, CreatedAt: time.Unix(2, 0)},
	} {
		if _, err := events.Append(ctx, event); err != nil {
			t.Fatalf("append cost event: %v", err)
		}
	}
	ledger := cost.New(events)
	total, currency, err := ledger.AggregateTask(ctx, "task-retry")
	if err != nil {
		t.Fatalf("AggregateTask returned error: %v", err)
	}
	if total != 4 || currency != "USD" {
		t.Fatalf("aggregate = %f %s, want 4 USD including both attempts", total, currency)
	}
}

func timePointer(value time.Time) *time.Time { return &value }
