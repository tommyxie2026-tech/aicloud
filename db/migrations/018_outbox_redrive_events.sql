-- S3D bounded Outbox dead-letter recovery evidence.
--
-- Dead-letter messages are never revived by global scanning. A redrive is an
-- explicit tenant/project-scoped action against one outbox_id. The mutable
-- Outbox status change and this immutable recovery evidence are committed in
-- the same PostgreSQL transaction by the repository.

CREATE TABLE IF NOT EXISTS outbox_redrive_events (
    redrive_event_id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    outbox_id TEXT NOT NULL,
    actor_principal_type TEXT NOT NULL,
    actor_subject_id TEXT NOT NULL,
    reason TEXT NOT NULL,
    previous_attempts INTEGER NOT NULL,
    previous_last_error TEXT NOT NULL DEFAULT '',
    redriven_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT outbox_redrive_attempts_nonnegative_check CHECK (previous_attempts >= 0),
    CONSTRAINT outbox_redrive_reason_nonempty_check CHECK (reason <> '')
);

CREATE INDEX IF NOT EXISTS outbox_redrive_scope_message_time_idx
    ON outbox_redrive_events(tenant_id, project_id, outbox_id, redriven_at, redrive_event_id);

ALTER TABLE outbox_redrive_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE outbox_redrive_events FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS outbox_redrive_events_select_policy ON outbox_redrive_events;
CREATE POLICY outbox_redrive_events_select_policy ON outbox_redrive_events
    FOR SELECT
    USING (
        tenant_id = current_setting('aicloud.tenant_id', true)
        AND project_id = current_setting('aicloud.project_id', true)
    );

DROP POLICY IF EXISTS outbox_redrive_events_insert_policy ON outbox_redrive_events;
CREATE POLICY outbox_redrive_events_insert_policy ON outbox_redrive_events
    FOR INSERT
    WITH CHECK (
        tenant_id = current_setting('aicloud.tenant_id', true)
        AND project_id = current_setting('aicloud.project_id', true)
    );

-- Deliberately no UPDATE or DELETE policy. Recovery history is append-only and
-- may outlive operational Outbox retention, so no foreign key or cascade is
-- attached to outbox_messages.
