package migrations

import (
	"strings"
	"testing"
)

func TestTaskScopeIdentityMigrationRemovesSessionPrivilegeBypass(t *testing.T) {
	body, err := migrationFiles.ReadFile("005_task_scope_identity.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(body)
	for _, required := range []string{
		"ALTER TABLE tasks ADD COLUMN IF NOT EXISTS tenant_id TEXT",
		"ALTER TABLE tasks ADD COLUMN IF NOT EXISTS project_id TEXT",
		"ALTER TABLE tasks ADD COLUMN IF NOT EXISTS created_by TEXT",
		"ALTER TABLE tasks FORCE ROW LEVEL SECURITY",
		"current_setting('aicloud.tenant_id', true)",
		"current_setting('aicloud.project_id', true)",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("migration missing required invariant %q", required)
		}
	}
	if strings.Contains(sql, "aicloud.system_access") {
		t.Fatal("migration must not retain application-controlled system_access bypass")
	}
}
