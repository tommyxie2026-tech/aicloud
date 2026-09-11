-- R1 AI Execution OS distributed node runtime.
--
-- A row represents the mutable runtime state for one node in one immutable plan
-- revision. Lease token + monotonically increasing fence prevent stale workers
-- from mutating a newer attempt. This is fencing, not exactly-once execution;
-- side-effect safety still depends on EffectClass and idempotency semantics.

CREATE TABLE IF NOT EXISTS execution_node_runtime (
    tenant_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    execution_id TEXT NOT NULL,
    plan_revision BIGINT NOT NULL,
    node_id TEXT NOT NULL,
    state TEXT NOT NULL DEFAULT 'PENDING',
    effect_class TEXT NOT NULL DEFAULT 'PURE',
    retry_safe BOOLEAN NOT NULL DEFAULT TRUE,
    idempotency_key TEXT,
    attempt_number INTEGER NOT NULL DEFAULT 0,
    lease_owner TEXT,
    lease_token TEXT,
    lease_fence BIGINT NOT NULL DEFAULT 0,
    claimed_at TIMESTAMPTZ,
    heartbeat_at TIMESTAMPTZ,
    lease_expires_at TIMESTAMPTZ,
    effect_started_at TIMESTAMPTZ,
    effect_committed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT execution_node_runtime_pk PRIMARY KEY (
        tenant_id, project_id, execution_id, plan_revision, node_id
    ),
    CONSTRAINT execution_node_plan_revision_positive CHECK (plan_revision >= 1),
    CONSTRAINT execution_node_attempt_nonnegative CHECK (attempt_number >= 0),
    CONSTRAINT execution_node_lease_fence_nonnegative CHECK (lease_fence >= 0),
    CONSTRAINT execution_node_state_contract CHECK (state IN (
        'PENDING', 'READY', 'RUNNING', 'WAITING_APPROVAL',
        'SUCCEEDED', 'FAILED', 'SKIPPED', 'BLOCKED', 'CANCELLED'
    )),
    CONSTRAINT execution_node_effect_contract CHECK (effect_class IN (
        'PURE', 'READ_ONLY', 'IDEMPOTENT_MUTATION', 'NON_IDEMPOTENT_MUTATION'
    )),
    CONSTRAINT execution_node_idempotent_key_contract CHECK (
        effect_class <> 'IDEMPOTENT_MUTATION' OR NULLIF(idempotency_key, '') IS NOT NULL
    ),
    CONSTRAINT execution_node_non_idempotent_retry_contract CHECK (
        effect_class <> 'NON_IDEMPOTENT_MUTATION' OR retry_safe = FALSE
    ),
    CONSTRAINT execution_node_lease_tuple_contract CHECK (
        (
            lease_owner IS NULL AND lease_token IS NULL AND claimed_at IS NULL
            AND heartbeat_at IS NULL AND lease_expires_at IS NULL
        ) OR (
            lease_owner IS NOT NULL AND lease_token IS NOT NULL AND claimed_at IS NOT NULL
            AND heartbeat_at IS NOT NULL AND lease_expires_at IS NOT NULL
            AND lease_expires_at > claimed_at
        )
    ),
    CONSTRAINT execution_node_effect_commit_contract CHECK (
        effect_committed_at IS NULL OR effect_started_at IS NOT NULL
    )
);

CREATE INDEX IF NOT EXISTS execution_node_runtime_ready_idx
    ON execution_node_runtime(tenant_id, project_id, state, updated_at, execution_id, node_id)
    WHERE state = 'READY';

CREATE INDEX IF NOT EXISTS execution_node_runtime_expired_lease_idx
    ON execution_node_runtime(tenant_id, project_id, lease_expires_at, execution_id, node_id)
    WHERE state = 'RUNNING' AND lease_expires_at IS NOT NULL;

ALTER TABLE execution_node_runtime ENABLE ROW LEVEL SECURITY;
ALTER TABLE execution_node_runtime FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS execution_node_runtime_scope_policy ON execution_node_runtime;
CREATE POLICY execution_node_runtime_scope_policy ON execution_node_runtime
    USING (
        tenant_id = current_setting('aicloud.tenant_id', true)
        AND project_id = current_setting('aicloud.project_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('aicloud.tenant_id', true)
        AND project_id = current_setting('aicloud.project_id', true)
    );

COMMENT ON COLUMN execution_node_runtime.lease_fence IS
    'Monotonically increases on every successful claim/reclaim. Stale workers must not mutate rows protected by a newer fence.';
COMMENT ON COLUMN execution_node_runtime.retry_safe IS
    'Whether an expired attempt may be automatically reclaimed according to the frozen plan/effect semantics.';
