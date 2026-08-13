ALTER TABLE cost_events
    ADD COLUMN IF NOT EXISTS deployment_id TEXT NOT NULL DEFAULT '';

UPDATE cost_events
SET deployment_id = COALESCE(metadata->>'deployment_id', '')
WHERE deployment_id = '' AND metadata ? 'deployment_id';

CREATE INDEX IF NOT EXISTS cost_events_deployment_idx
    ON cost_events(deployment_id, created_at);

COMMENT ON COLUMN cost_events.deployment_id IS
    'Exact model deployment used for this cost event. New writers also retain the same value in metadata during the R5 compatibility window.';
