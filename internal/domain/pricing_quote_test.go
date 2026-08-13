package domain

import (
	"math"
	"testing"
	"time"
)

func TestQuotePricingAppliesContextCacheAndRoutingFactors(t *testing.T) {
	now := time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)
	inputBand := 4.0
	outputBand := 12.0
	policy := PricingPolicy{
		ID: "policy-1", Version: "v2", DeploymentID: "deployment-1", Currency: "USD",
		InputPerMillion: 2, OutputPerMillion: 10, CacheHitPerMillion: 0.5,
		ContextBands: []PricingContextBand{{MinTokens: 128000, InputPerMillion: &inputBand, OutputPerMillion: &outputBand}},
		BatchFactor: 0.5,
		ServiceTierFactors: map[ServiceTier]float64{TierPriority: 1.2},
		InferenceEffortFactors: map[InferenceEffort]float64{EffortHigh: 1.5},
		EffectiveFrom: now.Add(-time.Hour), CreatedAt: now.Add(-time.Hour),
	}
	quote, err := QuotePricing(policy, PricingUsageEstimate{
		InputTokens: 1_000_000, OutputTokens: 100_000, CacheHitInputTokens: 400_000,
		ContextTokens: 200_000, Batch: true, ServiceTier: TierPriority, InferenceEffort: EffortHigh,
	}, now)
	if err != nil {
		t.Fatalf("QuotePricing returned error: %v", err)
	}
	if quote.PolicyID != policy.ID || quote.PolicyVersion != policy.Version || quote.DeploymentID != policy.DeploymentID {
		t.Fatalf("pricing identity changed: %#v", quote)
	}
	if math.Abs(quote.Total-3.42) > 0.000001 {
		t.Fatalf("quote total = %f, want 3.42", quote.Total)
	}
}

func TestQuotePricingRejectsExpiredPolicy(t *testing.T) {
	now := time.Now().UTC()
	end := now.Add(-time.Minute)
	policy := PricingPolicy{
		ID: "policy-1", Version: "v1", DeploymentID: "deployment-1", Currency: "USD",
		EffectiveFrom: now.Add(-time.Hour), EffectiveTo: &end,
	}
	if _, err := QuotePricing(policy, PricingUsageEstimate{}, now); err == nil {
		t.Fatal("expected expired pricing policy to be rejected")
	}
}
