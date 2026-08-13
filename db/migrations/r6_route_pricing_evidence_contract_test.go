package migrations

import (
	"strings"
	"testing"
)

func TestRoutePricingEvidenceMigration(t *testing.T) {
	body, err := migrationFiles.ReadFile("016_route_pricing_evidence.sql")
	if err != nil {
		t.Fatalf("read migration 016: %v", err)
	}
	sql := string(body)
	for _, required := range []string{
		"CREATE TABLE IF NOT EXISTS route_pricing_evidence",
		"route_decision_id TEXT NOT NULL REFERENCES route_decisions(id) ON DELETE CASCADE",
		"deployment_id TEXT NOT NULL REFERENCES model_deployments(id)",
		"FOREIGN KEY (policy_id, policy_version)",
		"policy_snapshot JSONB NOT NULL",
		"quote JSONB NOT NULL",
		"CREATE OR REPLACE FUNCTION capture_route_pricing_evidence()",
		"AFTER INSERT ON route_decisions",
		"effective_from <= NEW.created_at",
		"to_jsonb(policy_record)",
		"NEW.selected->>'estimatedCost'",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration 016 missing invariant %q", required)
		}
	}
}
