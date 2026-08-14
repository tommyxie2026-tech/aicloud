package migrations

import (
	"strings"
	"testing"
)

func TestOutboxRedriveMigrationIsScopedAndAppendOnly(t *testing.T) {
	body, err := migrationFiles.ReadFile("011_outbox_redrive_events.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(body)
	for _, required := range []string{
		"CREATE TABLE IF NOT EXISTS outbox_redrive_events",
		"tenant_id TEXT NOT NULL",
		"project_id TEXT NOT NULL",
		"outbox_id TEXT NOT NULL",
		"previous_attempts INTEGER NOT NULL",
		"previous_last_error TEXT NOT NULL",
		"ALTER TABLE outbox_redrive_events FORCE ROW LEVEL SECURITY",
		"current_setting('aicloud.tenant_id', true)",
		"current_setting('aicloud.project_id', true)",
		"FOR INSERT",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("Outbox redrive migration missing invariant %q", required)
		}
	}
	for _, forbidden := range []string{
		"FOR UPDATE",
		"FOR DELETE",
		"ON DELETE CASCADE",
		"REFERENCES outbox_messages",
	} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("Outbox redrive migration contains forbidden mutable/coupled invariant %q", forbidden)
		}
	}
}
