-- S1 contract compliance: Task owns tenant/project/creator identity directly.
-- The previous task_ownership table remains only as a migration bridge.

ALTER TABLE tasks ADD COLUMN IF NOT EXISTS tenant_id TEXT;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS project_id TEXT;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS created_by TEXT;

-- Backfill rows created by the S1 prototype from the bridge table.
UPDATE tasks AS t
SET tenant_id = o.tenant_id,
    project_id = o.project_id,
    created_by = o.subject_id
FROM task_ownership AS o
WHERE o.task_id = t.id
  AND (t.tenant_id IS NULL OR t.project_id IS NULL OR t.created_by IS NULL);

-- Contract freeze requires all Task rows to be scoped. Abort rather than
-- silently inventing ownership for legacy/orphan rows.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM tasks
        WHERE tenant_id IS NULL OR tenant_id = ''
           OR project_id IS NULL OR project_id = ''
           OR created_by IS NULL OR created_by = ''
    ) THEN
        RAISE EXCEPTION 'unscoped task rows exist; backfill ownership before applying 005_task_scope_identity.sql';
    END IF;
END
$$;

ALTER TABLE tasks ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE tasks ALTER COLUMN project_id SET NOT NULL;
ALTER TABLE tasks ALTER COLUMN created_by SET NOT NULL;

CREATE INDEX IF NOT EXISTS tasks_tenant_project_created_idx
    ON tasks(tenant_id, project_id, created_at, id);

ALTER TABLE tasks ENABLE ROW LEVEL SECURITY;
ALTER TABLE tasks FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tasks_tenant_project_policy ON tasks;
CREATE POLICY tasks_tenant_project_policy ON tasks
    USING (
        tenant_id = current_setting('aicloud.tenant_id', true)
        AND project_id = current_setting('aicloud.project_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('aicloud.tenant_id', true)
        AND project_id = current_setting('aicloud.project_id', true)
    );

-- Remove the prototype session-flag privilege bypass from the bridge table.
-- If the bridge table is queried during migration support, it is still strictly
-- tenant/project scoped.
DROP POLICY IF EXISTS task_ownership_tenant_policy ON task_ownership;
CREATE POLICY task_ownership_tenant_policy ON task_ownership
    USING (
        tenant_id = current_setting('aicloud.tenant_id', true)
        AND project_id = current_setting('aicloud.project_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('aicloud.tenant_id', true)
        AND project_id = current_setting('aicloud.project_id', true)
    );
