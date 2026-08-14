package repository

import (
	"context"
	"testing"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

func TestMemoryPricingPoliciesResolveVersionByTime(t *testing.T) {
	store := NewMemoryPricingPolicies()
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	cutover := start.Add(7 * 24 * time.Hour)
	end := cutover
	first := domain.PricingPolicy{
		ID: "policy", Version: "v1", DeploymentID: "deployment-1", Currency: "USD",
		EffectiveFrom: start, EffectiveTo: &end,
	}
	second := domain.PricingPolicy{
		ID: "policy", Version: "v2", DeploymentID: "deployment-1", Currency: "USD",
		EffectiveFrom: cutover,
	}
	if _, err := store.Create(context.Background(), first); err != nil {
		t.Fatalf("create first policy: %v", err)
	}
	if _, err := store.Create(context.Background(), second); err != nil {
		t.Fatalf("create second policy: %v", err)
	}
	old, err := store.Resolve(context.Background(), "deployment-1", start.Add(time.Hour))
	if err != nil || old.Version != "v1" {
		t.Fatalf("historical resolve = %#v err=%v", old, err)
	}
	current, err := store.Resolve(context.Background(), "deployment-1", cutover.Add(time.Hour))
	if err != nil || current.Version != "v2" {
		t.Fatalf("current resolve = %#v err=%v", current, err)
	}
}
