-- Slice 1: tenant ownership boundary for Task resources.
--
-- Task ownership is kept in a separate table during the v0.1 compatibility
-- migration so existing Task rows and repository contracts remain readable.
-- New externally-created Tasks are bound by ScopedTasks at creation time.
CREATE TABLE IF NOT EXISTS task_ownership (
    task_id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    project_id TEXT NOT NULL DEFAULT '',
    subject_id TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS task_ownership_tenant_project_idx
    ON task_ownership(tenant_id, project_id, created_at, task_id);

-- Defense-in-depth RLS is prepared here. The application repository always
-- applies explicit ownership checks. RLS becomes mandatory once all PostgreSQL
-- access paths use transaction-local aicloud.tenant_id settings.
ALTER TABLE task_ownership ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS task_ownership_tenant_policy ON task_ownership;
CREATE POLICY task_ownership_tenant_policy ON task_ownership
    USING (
        current_setting('aicloud.system_access', true) = 'on'
        OR tenant_id = current_setting('aicloud.tenant_id', true)
    )
    WITH CHECK (
        current_setting('aicloud.system_access', true) = 'on'
        OR tenant_id = current_setting('aicloud.tenant_id', true)
    );
