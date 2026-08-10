-- R5 Task Aggregate convergence.
--
-- Existing S1 rows may still contain the prototype PENDING/RUNNING states.
-- This migration maps them explicitly into the frozen aggregate lifecycle and
-- introduces optimistic-concurrency versioning.

ALTER TABLE tasks ADD COLUMN IF NOT EXISTS version BIGINT;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS completed_at TIMESTAMPTZ;

UPDATE tasks
SET status = CASE status
    WHEN 'PENDING' THEN 'CREATED'
    WHEN 'RUNNING' THEN 'EXECUTING'
    ELSE status
END;

UPDATE tasks
SET version = 1
WHERE version IS NULL OR version < 1;

UPDATE tasks
SET completed_at = COALESCE(completed_at, updated_at, created_at, NOW())
WHERE status IN ('COMPLETED', 'FAILED', 'CANCELLED', 'EXPIRED')
  AND completed_at IS NULL;

ALTER TABLE tasks ALTER COLUMN version SET DEFAULT 1;
ALTER TABLE tasks ALTER COLUMN version SET NOT NULL;

ALTER TABLE tasks DROP CONSTRAINT IF EXISTS tasks_version_positive_check;
ALTER TABLE tasks ADD CONSTRAINT tasks_version_positive_check
    CHECK (version >= 1);

ALTER TABLE tasks DROP CONSTRAINT IF EXISTS tasks_status_contract_check;
ALTER TABLE tasks ADD CONSTRAINT tasks_status_contract_check
    CHECK (status IN (
        'CREATED',
        'PLANNING',
        'ROUTING',
        'EXECUTING',
        'WAITING_APPROVAL',
        'VALIDATING',
        'COMPLETED',
        'FAILED',
        'CANCELLED',
        'EXPIRED'
    ));

CREATE INDEX IF NOT EXISTS tasks_scope_status_created_idx
    ON tasks(tenant_id, project_id, status, created_at, id);
