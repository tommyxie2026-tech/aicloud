-- R6 Task transaction kernel.
--
-- This migration introduces immutable TaskEvent business history, a
-- transactional outbox for external delivery intent, and durable command
-- idempotency records. All runtime-visible records carry tenant/project scope
-- and are protected by PostgreSQL row-level security.

CREATE TABLE IF NOT EXISTS task_events (
    event_id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE RESTRICT,
    sequence BIGINT NOT NULL,
    event_type TEXT NOT NULL,
    actor_principal_type TEXT NOT NULL,
    actor_subject_id TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    request_id TEXT,
    trace_id TEXT NOT NULL,
    schema_version INTEGER NOT NULL DEFAULT 1,
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT task_events_sequence_positive_check CHECK (sequence >= 1),
    CONSTRAINT task_events_schema_version_positive_check CHECK (schema_version >= 1),
    CONSTRAINT task_events_task_sequence_unique UNIQUE (task_id, sequence)
);

CREATE INDEX IF NOT EXISTS task_events_scope_task_sequence_idx
    ON task_events(tenant_id, project_id, task_id, sequence);
CREATE INDEX IF NOT EXISTS task_events_scope_created_idx
    ON task_events(tenant_id, project_id, created_at, event_id);

ALTER TABLE task_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE task_events FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS task_events_select_policy ON task_events;
CREATE POLICY task_events_select_policy ON task_events
    FOR SELECT
    USING (
        tenant_id = current_setting('aicloud.tenant_id', true)
        AND project_id = current_setting('aicloud.project_id', true)
    );

DROP POLICY IF EXISTS task_events_insert_policy ON task_events;
CREATE POLICY task_events_insert_policy ON task_events
    FOR INSERT
    WITH CHECK (
        tenant_id = current_setting('aicloud.tenant_id', true)
        AND project_id = current_setting('aicloud.project_id', true)
    );

-- Deliberately no UPDATE or DELETE policy. Runtime TaskEvent history is
-- append-only. Governed retention actions require a separate administrative
-- path using independently managed credentials.

CREATE TABLE IF NOT EXISTS outbox_messages (
    outbox_id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    task_id TEXT REFERENCES tasks(id) ON DELETE RESTRICT,
    aggregate_type TEXT NOT NULL,
    aggregate_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    destination TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    attempts INTEGER NOT NULL DEFAULT 0,
    available_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    delivered_at TIMESTAMPTZ,
    CONSTRAINT outbox_attempts_nonnegative_check CHECK (attempts >= 0),
    CONSTRAINT outbox_status_contract_check CHECK (status IN (
        'pending', 'delivering', 'delivered', 'dead_letter'
    )),
    CONSTRAINT outbox_delivery_idempotency_unique UNIQUE (
        tenant_id, project_id, destination, idempotency_key
    )
);

CREATE INDEX IF NOT EXISTS outbox_dispatch_idx
    ON outbox_messages(status, available_at, created_at, outbox_id);
CREATE INDEX IF NOT EXISTS outbox_scope_task_idx
    ON outbox_messages(tenant_id, project_id, task_id, created_at, outbox_id);

ALTER TABLE outbox_messages ENABLE ROW LEVEL SECURITY;
ALTER TABLE outbox_messages FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS outbox_messages_scope_policy ON outbox_messages;
CREATE POLICY outbox_messages_scope_policy ON outbox_messages
    USING (
        tenant_id = current_setting('aicloud.tenant_id', true)
        AND project_id = current_setting('aicloud.project_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('aicloud.tenant_id', true)
        AND project_id = current_setting('aicloud.project_id', true)
    );

CREATE TABLE IF NOT EXISTS idempotency_records (
    tenant_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    operation TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    request_digest TEXT NOT NULL,
    status TEXT NOT NULL,
    resource_id TEXT,
    response_code INTEGER,
    response_digest TEXT,
    response_payload JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT idempotency_records_pk PRIMARY KEY (
        tenant_id, project_id, operation, idempotency_key
    ),
    CONSTRAINT idempotency_status_contract_check CHECK (status IN (
        'in_progress', 'completed', 'failed_retryable', 'failed_final'
    )),
    CONSTRAINT idempotency_expiry_check CHECK (expires_at > created_at)
);

CREATE INDEX IF NOT EXISTS idempotency_records_expiry_idx
    ON idempotency_records(expires_at, tenant_id, project_id);
CREATE INDEX IF NOT EXISTS idempotency_records_resource_idx
    ON idempotency_records(tenant_id, project_id, resource_id)
    WHERE resource_id IS NOT NULL;

ALTER TABLE idempotency_records ENABLE ROW LEVEL SECURITY;
ALTER TABLE idempotency_records FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS idempotency_records_scope_policy ON idempotency_records;
CREATE POLICY idempotency_records_scope_policy ON idempotency_records
    USING (
        tenant_id = current_setting('aicloud.tenant_id', true)
        AND project_id = current_setting('aicloud.project_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('aicloud.tenant_id', true)
        AND project_id = current_setting('aicloud.project_id', true)
    );
