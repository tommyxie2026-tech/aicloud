package migrations

import (
	"strings"
	"testing"
)

func TestTaskEventOutboxIdempotencyMigrationContract(t *testing.T) {
	body, err := migrationFiles.ReadFile("007_task_event_outbox_idempotency.sql")
	if err != nil {
		t.Fatalf("read migration 007: %v", err)
	}
	sql := string(body)
	for _, required := range []string{
		"CREATE TABLE IF NOT EXISTS task_events",
		"UNIQUE (task_id, sequence)",
		"ALTER TABLE task_events FORCE ROW LEVEL SECURITY",
		"FOR SELECT",
		"FOR INSERT",
		"CREATE TABLE IF NOT EXISTS outbox_messages",
		"'pending', 'delivering', 'delivered', 'dead_letter'",
		"UNIQUE (\n        tenant_id, project_id, destination, idempotency_key",
		"ALTER TABLE outbox_messages FORCE ROW LEVEL SECURITY",
		"CREATE TABLE IF NOT EXISTS idempotency_records",
		"tenant_id, project_id, operation, idempotency_key",
		"'in_progress', 'completed', 'failed_retryable', 'failed_final'",
		"ALTER TABLE idempotency_records FORCE ROW LEVEL SECURITY",
		"current_setting('aicloud.tenant_id', true)",
		"current_setting('aicloud.project_id', true)",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration 007 missing invariant %q", required)
		}
	}

	if strings.Contains(sql, "CREATE POLICY task_events_update") || strings.Contains(sql, "CREATE POLICY task_events_delete") {
		t.Fatal("runtime TaskEvent schema must not define UPDATE/DELETE policies")
	}
}
