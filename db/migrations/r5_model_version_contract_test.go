package migrations

import (
	"strings"
	"testing"
)

func TestModelVersionDeploymentBoundary(t *testing.T) {
	body, err := migrationFiles.ReadFile("011_model_version_deployment_boundary.sql")
	if err != nil {
		t.Fatalf("read migration 011: %v", err)
	}
	sql := string(body)
	for _, required := range []string{
		"CREATE TABLE IF NOT EXISTS model_versions",
		"ADD COLUMN IF NOT EXISTS model_version_id TEXT",
		"FOREIGN KEY (model_version_id) REFERENCES model_versions(id)",
		"ADD COLUMN IF NOT EXISTS pricing JSONB",
		"model_deployments_model_version_id_idx",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration 011 missing invariant %q", required)
		}
	}

	start := strings.Index(sql, "CREATE TABLE IF NOT EXISTS model_versions")
	end := strings.Index(sql[start:], ");")
	if start < 0 || end < 0 {
		t.Fatal("model_versions table definition not found")
	}
	definition := sql[start : start+end]
	for _, forbidden := range []string{"endpoint", "health_status", "quota_remaining", "capacity_available", "queue_depth", "pricing JSONB", "data_residency"} {
		if strings.Contains(definition, forbidden) {
			t.Fatalf("model_versions must not contain runtime field %q", forbidden)
		}
	}
}
