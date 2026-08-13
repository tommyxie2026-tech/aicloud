package migrations

import (
	"strings"
	"testing"
)

func TestDeploymentLifecycleEvidenceMigration(t *testing.T) {
	body, err := migrationFiles.ReadFile("014_deployment_lifecycle_events.sql")
	if err != nil {
		t.Fatalf("read migration 014: %v", err)
	}
	sql := string(body)
	for _, required := range []string{
		"CREATE TABLE IF NOT EXISTS deployment_lifecycle_events",
		"deployment_id TEXT NOT NULL REFERENCES model_deployments(id)",
		"announced_at TIMESTAMPTZ",
		"effective_at TIMESTAMPTZ NOT NULL",
		"evidence_ref TEXT",
		"replacement_ids JSONB",
		"quota_remaining BIGINT",
		"rate_limit_ref TEXT",
		"routing_eligible BOOLEAN NOT NULL",
		"migration_state TEXT",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration 014 missing invariant %q", required)
		}
	}
}
