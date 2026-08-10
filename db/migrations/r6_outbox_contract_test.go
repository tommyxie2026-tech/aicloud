package migrations

import (
	"strings"
	"testing"
)

func TestOutboxLeaseMigrationDefinesRecoverableDeliveryState(t *testing.T) {
	body, err := migrationFiles.ReadFile("008_outbox_dispatch_leases.sql")
	if err != nil {
		t.Fatalf("read migration 008: %v", err)
	}
	sql := string(body)
	for _, required := range []string{
		"lease_owner TEXT",
		"lease_expires_at TIMESTAMPTZ",
		"last_error TEXT",
		"outbox_lease_state_check",
		"status = 'delivering'",
		"outbox_dispatch_lease_idx",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("outbox lease migration missing invariant %q", required)
		}
	}
}
