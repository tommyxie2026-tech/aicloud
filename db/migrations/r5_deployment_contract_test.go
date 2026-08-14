package migrations

import (
	"strings"
	"testing"
)

func TestDeploymentRegistryMigrationContract(t *testing.T) {
	body, err := migrationFiles.ReadFile("009_deployment_registry.sql")
	if err != nil {
		t.Fatalf("read migration 009: %v", err)
	}
	sql := string(body)
	for _, required := range []string{
		"CREATE TABLE IF NOT EXISTS model_deployments",
		"model_id TEXT NOT NULL",
		"model_version TEXT NOT NULL",
		"pricing_policy_ref TEXT NOT NULL",
		"health_checked_at TIMESTAMPTZ",
		"routing_eligible BOOLEAN NOT NULL",
		"model_deployments_model_idx",
		"model_deployments_routing_idx",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration 009 missing invariant %q", required)
		}
	}
}

func TestCostDeploymentMigrationContract(t *testing.T) {
	body, err := migrationFiles.ReadFile("010_cost_event_deployment.sql")
	if err != nil {
		t.Fatalf("read migration 010: %v", err)
	}
	sql := string(body)
	for _, required := range []string{
		"ADD COLUMN IF NOT EXISTS deployment_id",
		"metadata->>'deployment_id'",
		"cost_events_deployment_idx",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration 010 missing invariant %q", required)
		}
	}
}
