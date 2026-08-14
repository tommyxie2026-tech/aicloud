package migrations

import (
	"strings"
	"testing"
)

func TestPricingPolicyMigrationContract(t *testing.T) {
	body, err := migrationFiles.ReadFile("015_pricing_policies.sql")
	if err != nil {
		t.Fatalf("read migration 015: %v", err)
	}
	sql := string(body)
	for _, required := range []string{
		"PRIMARY KEY (id, version)",
		"deployment_id TEXT NOT NULL REFERENCES model_deployments(id)",
		"context_bands JSONB",
		"service_tier_factors JSONB",
		"inference_effort_factors JSONB",
		"capacity_pricing JSONB",
		"self_hosted_allocation JSONB",
		"effective_from TIMESTAMPTZ NOT NULL",
		"effective_to TIMESTAMPTZ",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration 015 missing invariant %q", required)
		}
	}
}
