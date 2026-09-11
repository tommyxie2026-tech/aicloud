package migrations

import (
	"strings"
	"testing"
)

func TestExecutionAttemptMigrationDefinesDurableHistoryInvariants(t *testing.T) {
	body, err := migrationFiles.ReadFile("018_execution_attempts.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(body)
	for _, required := range []string{
		"CREATE TABLE IF NOT EXISTS execution_attempts",
		"attempt_number INTEGER NOT NULL",
		"lease_fence BIGINT NOT NULL",
		"'ABANDONED'",
		"execution_attempts_scope_attempt_unique",
		"execution_attempts_scope_fence_unique",
		"execution_attempt_lifecycle_contract",
		"execution_attempt_started_target_contract",
		"execution_attempt_usage_nonnegative",
		"requires all execution node leases to be drained",
		"ALTER TABLE execution_attempts FORCE ROW LEVEL SECURITY",
		"current_setting('aicloud.tenant_id', true)",
		"current_setting('aicloud.project_id', true)",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("execution attempt migration missing invariant %q", required)
		}
	}
}
