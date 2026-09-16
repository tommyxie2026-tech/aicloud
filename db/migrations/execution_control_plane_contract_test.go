package migrations

import (
	"strings"
	"testing"
)

func TestExecutionControlPlaneMigrationDefinesDurableTaskLineage(t *testing.T) {
	body, err := migrationFiles.ReadFile("019_execution_control_plane.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	sql := string(body)
	for _, required := range []string{
		"CREATE TABLE IF NOT EXISTS execution_goals",
		"CREATE TABLE IF NOT EXISTS execution_plans",
		"CREATE TABLE IF NOT EXISTS executions",
		"task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE RESTRICT",
		"execution_goals_task_lineage_unique",
		"execution_plans_task_goal_lineage_unique",
		"CONSTRAINT execution_plans_goal_fk FOREIGN KEY",
		"CONSTRAINT executions_goal_fk FOREIGN KEY",
		"CONSTRAINT executions_plan_fk FOREIGN KEY",
		"ALTER TABLE execution_goals FORCE ROW LEVEL SECURITY",
		"ALTER TABLE execution_plans FORCE ROW LEVEL SECURITY",
		"ALTER TABLE executions FORCE ROW LEVEL SECURITY",
		"current_setting('aicloud.tenant_id', true)",
		"current_setting('aicloud.project_id', true)",
		"'REPLANNING'",
		"'VERIFYING'",
	} {
		if !strings.Contains(sql, required) {
			t.Fatalf("execution control-plane migration missing invariant %q", required)
		}
	}
	if strings.Contains(sql, "CREATE TABLE IF NOT EXISTS execution_events") {
		t.Fatal("execution control plane must reuse canonical task_events instead of creating a second event stream")
	}
}
