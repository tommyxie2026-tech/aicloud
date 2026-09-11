package migrations

import (
	"strings"
	"testing"
)

func TestExecutionNodeLeaseMigrationDefinesFencingAndSafetyInvariants(t *testing.T) {
	body, err := migrationFiles.ReadFile("017_execution_node_runtime_leases.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(body)
	for _, required := range []string{
		"CREATE TABLE IF NOT EXISTS execution_node_runtime",
		"plan_revision BIGINT NOT NULL",
		"lease_token TEXT",
		"lease_fence BIGINT NOT NULL DEFAULT 0",
		"attempt_number INTEGER NOT NULL DEFAULT 0",
		"effect_started_at TIMESTAMPTZ",
		"effect_committed_at TIMESTAMPTZ",
		"effect_class <> 'NON_IDEMPOTENT_MUTATION' OR retry_safe = FALSE",
		"ALTER TABLE execution_node_runtime FORCE ROW LEVEL SECURITY",
		"current_setting('aicloud.tenant_id', true)",
		"current_setting('aicloud.project_id', true)",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("execution node lease migration missing invariant %q", required)
		}
	}
}
