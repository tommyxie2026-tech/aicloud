package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

func TestMigrationVersionPrefixesAreUnique(t *testing.T) {
	entries, err := fs.ReadDir(migrationFiles, ".")
	if err != nil {
		t.Fatalf("read migrations: %v", err)
	}
	seen := map[string]string{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		parts := strings.SplitN(entry.Name(), "_", 2)
		if len(parts) != 2 || len(parts[0]) != 3 {
			t.Fatalf("migration %q must use a three-digit version prefix", entry.Name())
		}
		if previous, ok := seen[parts[0]]; ok {
			t.Fatalf("migration version %s is duplicated by %q and %q", parts[0], previous, entry.Name())
		}
		seen[parts[0]] = entry.Name()
	}
}

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

func TestTaskAggregateMigrationDefinesCanonicalStatesAndVersion(t *testing.T) {
	body, err := migrationFiles.ReadFile("006_task_aggregate_state.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(body)
	for _, required := range []string{
		"ALTER TABLE tasks ADD COLUMN IF NOT EXISTS version BIGINT",
		"WHEN 'PENDING' THEN 'CREATED'",
		"WHEN 'RUNNING' THEN 'EXECUTING'",
		"ALTER TABLE tasks ALTER COLUMN version SET NOT NULL",
		"CHECK (version >= 1)",
		"WAITING_APPROVAL",
		"VALIDATING",
		"CANCELLED",
		"EXPIRED",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("task aggregate migration missing invariant %q", required)
		}
	}
}
