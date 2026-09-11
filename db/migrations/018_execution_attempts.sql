-- Durable execution attempt history for AI Execution OS R1.
--
-- One row is created atomically with every successful node claim. It is never
-- reused by a later fence. Retry/reclaim creates a new attempt and preserves the
-- prior attempt as historical execution evidence.
--
-- First-time deployment of this migration requires execution workers to be
-- drained. A RUNNING row created by the pre-018 runtime has a fence but no
-- durable Attempt, so silently hot-migrating it would create an audit gap.
DO $$
BEGIN
    IF to_regclass('execution_attempts') IS NULL THEN
        IF to_regclass('execution_node_runtime') IS NULL THEN
            RAISE EXCEPTION 'migration 018 requires execution_node_runtime from migration 017';
        END IF;
        IF EXISTS (SELECT 1 FROM execution_node_runtime WHERE state = 'RUNNING') THEN
            RAISE EXCEPTION 'migration 018 requires all execution node leases to be drained before first apply';
        END IF;
    END IF;
END
$$;

CREATE TABLE IF NOT EXISTS execution_attempts (
    tenant_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    attempt_id TEXT NOT NULL,
    execution_id TEXT NOT NULL,
    plan_revision BIGINT NOT NULL,
    node_id TEXT NOT NULL,
    attempt_number INTEGER NOT NULL,
    lease_fence BIGINT NOT NULL,
    lease_owner TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'PENDING',

    target_id TEXT,
    target_revision BIGINT,
    target_snapshot_digest TEXT,
    policy_decision_id TEXT,
    budget_reservation_id TEXT,
    idempotency_key TEXT,

    error_class TEXT,
    input_tokens BIGINT NOT NULL DEFAULT 0,
    output_tokens BIGINT NOT NULL DEFAULT 0,
    cost NUMERIC(20,8) NOT NULL DEFAULT 0,
    duration_ms BIGINT NOT NULL DEFAULT 0,

    claimed_at TIMESTAMPTZ NOT NULL,
    started_at TIMESTAMPTZ,
    effect_started_at TIMESTAMPTZ,
    effect_committed_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT execution_attempts_pk PRIMARY KEY (tenant_id, project_id, attempt_id),
    CONSTRAINT execution_attempts_scope_attempt_unique UNIQUE (
        tenant_id, project_id, execution_id, plan_revision, node_id, attempt_number
    ),
    CONSTRAINT execution_attempts_scope_fence_unique UNIQUE (
        tenant_id, project_id, execution_id, plan_revision, node_id, lease_fence
    ),
    CONSTRAINT execution_attempt_plan_revision_positive CHECK (plan_revision >= 1),
    CONSTRAINT execution_attempt_number_positive CHECK (attempt_number >= 1),
    CONSTRAINT execution_attempt_fence_positive CHECK (lease_fence >= 1),
    CONSTRAINT execution_attempt_status_contract CHECK (status IN (
        'PENDING', 'RUNNING', 'SUCCEEDED', 'FAILED', 'CANCELLED', 'ABANDONED'
    )),
    CONSTRAINT execution_attempt_target_tuple_contract CHECK (
        (
            target_id IS NULL AND target_revision IS NULL AND target_snapshot_digest IS NULL
        ) OR (
            target_id IS NOT NULL AND target_revision IS NOT NULL AND target_revision >= 1
            AND NULLIF(target_snapshot_digest, '') IS NOT NULL
        )
    ),
    CONSTRAINT execution_attempt_lifecycle_contract CHECK (
        (status = 'PENDING' AND started_at IS NULL AND finished_at IS NULL)
        OR (status = 'RUNNING' AND started_at IS NOT NULL AND finished_at IS NULL)
        OR (status IN ('SUCCEEDED','FAILED','CANCELLED','ABANDONED') AND finished_at IS NOT NULL)
    ),
    CONSTRAINT execution_attempt_started_target_contract CHECK (
        started_at IS NULL OR (
            target_id IS NOT NULL AND target_revision IS NOT NULL
            AND NULLIF(target_snapshot_digest, '') IS NOT NULL
        )
    ),
    CONSTRAINT execution_attempt_effect_commit_contract CHECK (
        effect_committed_at IS NULL OR effect_started_at IS NOT NULL
    ),
    CONSTRAINT execution_attempt_usage_nonnegative CHECK (
        input_tokens >= 0 AND output_tokens >= 0 AND cost >= 0 AND duration_ms >= 0
    )
);

CREATE INDEX IF NOT EXISTS execution_attempts_node_history_idx
    ON execution_attempts(
        tenant_id, project_id, execution_id, plan_revision, node_id, attempt_number
    );

CREATE INDEX IF NOT EXISTS execution_attempts_status_idx
    ON execution_attempts(tenant_id, project_id, status, updated_at);

ALTER TABLE execution_attempts ENABLE ROW LEVEL SECURITY;
ALTER TABLE execution_attempts FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS execution_attempts_scope_policy ON execution_attempts;
CREATE POLICY execution_attempts_scope_policy ON execution_attempts
    USING (
        tenant_id = current_setting('aicloud.tenant_id', true)
        AND project_id = current_setting('aicloud.project_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('aicloud.tenant_id', true)
        AND project_id = current_setting('aicloud.project_id', true)
    );

COMMENT ON TABLE execution_attempts IS
    'Durable append-oriented node attempt history. Each successful claim/fence receives a distinct attempt row.';
COMMENT ON COLUMN execution_attempts.status IS
    'ABANDONED means the attempt lost execution authority before terminal completion; it is not equivalent to FAILED.';
