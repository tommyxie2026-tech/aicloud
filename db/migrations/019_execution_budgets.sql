-- Shared durable Budget state for AI Execution OS R1.
--
-- Every Reserve/Settle/Release operation locks the parent execution_budgets row
-- before mutating reservation/counter state. This serializes concurrent workers
-- against one execution budget without relying on process-local mutexes.

CREATE TABLE IF NOT EXISTS execution_budgets (
    tenant_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    execution_id TEXT NOT NULL,

    max_cost NUMERIC(20,8) NOT NULL DEFAULT 0,
    max_duration_ms BIGINT NOT NULL DEFAULT 0,
    max_node_attempts BIGINT NOT NULL DEFAULT 0,
    max_frontier_calls BIGINT NOT NULL DEFAULT 0,
    max_tool_calls BIGINT NOT NULL DEFAULT 0,

    reserved_cost NUMERIC(20,8) NOT NULL DEFAULT 0,
    reserved_duration_ms BIGINT NOT NULL DEFAULT 0,
    reserved_node_attempts BIGINT NOT NULL DEFAULT 0,
    reserved_frontier_calls BIGINT NOT NULL DEFAULT 0,
    reserved_tool_calls BIGINT NOT NULL DEFAULT 0,

    consumed_cost NUMERIC(20,8) NOT NULL DEFAULT 0,
    consumed_duration_ms BIGINT NOT NULL DEFAULT 0,
    consumed_node_attempts BIGINT NOT NULL DEFAULT 0,
    consumed_frontier_calls BIGINT NOT NULL DEFAULT 0,
    consumed_tool_calls BIGINT NOT NULL DEFAULT 0,

    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT execution_budgets_pk PRIMARY KEY (tenant_id, project_id, execution_id),
    CONSTRAINT execution_budget_limits_nonnegative CHECK (
        max_cost >= 0 AND max_duration_ms >= 0 AND max_node_attempts >= 0
        AND max_frontier_calls >= 0 AND max_tool_calls >= 0
    ),
    CONSTRAINT execution_budget_reserved_nonnegative CHECK (
        reserved_cost >= 0 AND reserved_duration_ms >= 0 AND reserved_node_attempts >= 0
        AND reserved_frontier_calls >= 0 AND reserved_tool_calls >= 0
    ),
    CONSTRAINT execution_budget_consumed_nonnegative CHECK (
        consumed_cost >= 0 AND consumed_duration_ms >= 0 AND consumed_node_attempts >= 0
        AND consumed_frontier_calls >= 0 AND consumed_tool_calls >= 0
    ),
    CONSTRAINT execution_budget_version_positive CHECK (version >= 1)
);

CREATE TABLE IF NOT EXISTS execution_budget_reservations (
    tenant_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    reservation_id TEXT NOT NULL,
    execution_id TEXT NOT NULL,
    node_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'OPEN',

    estimate_cost NUMERIC(20,8) NOT NULL DEFAULT 0,
    estimate_duration_ms BIGINT NOT NULL DEFAULT 0,
    estimate_node_attempts BIGINT NOT NULL DEFAULT 0,
    estimate_frontier_calls BIGINT NOT NULL DEFAULT 0,
    estimate_tool_calls BIGINT NOT NULL DEFAULT 0,

    actual_cost NUMERIC(20,8) NOT NULL DEFAULT 0,
    actual_duration_ms BIGINT NOT NULL DEFAULT 0,
    actual_node_attempts BIGINT NOT NULL DEFAULT 0,
    actual_frontier_calls BIGINT NOT NULL DEFAULT 0,
    actual_tool_calls BIGINT NOT NULL DEFAULT 0,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT execution_budget_reservations_pk PRIMARY KEY (
        tenant_id, project_id, reservation_id
    ),
    CONSTRAINT execution_budget_reservation_budget_fk FOREIGN KEY (
        tenant_id, project_id, execution_id
    ) REFERENCES execution_budgets(tenant_id, project_id, execution_id) ON DELETE CASCADE,
    CONSTRAINT execution_budget_reservation_status CHECK (
        status IN ('OPEN', 'SETTLED', 'RELEASED')
    ),
    CONSTRAINT execution_budget_reservation_estimate_nonnegative CHECK (
        estimate_cost >= 0 AND estimate_duration_ms >= 0 AND estimate_node_attempts >= 0
        AND estimate_frontier_calls >= 0 AND estimate_tool_calls >= 0
    ),
    CONSTRAINT execution_budget_reservation_actual_nonnegative CHECK (
        actual_cost >= 0 AND actual_duration_ms >= 0 AND actual_node_attempts >= 0
        AND actual_frontier_calls >= 0 AND actual_tool_calls >= 0
    ),
    CONSTRAINT execution_budget_reservation_lifecycle CHECK (
        (status = 'OPEN' AND closed_at IS NULL)
        OR (status IN ('SETTLED', 'RELEASED') AND closed_at IS NOT NULL)
    ),
    CONSTRAINT execution_budget_release_has_no_actual CHECK (
        status <> 'RELEASED'
        OR (
            actual_cost = 0 AND actual_duration_ms = 0 AND actual_node_attempts = 0
            AND actual_frontier_calls = 0 AND actual_tool_calls = 0
        )
    )
);

CREATE INDEX IF NOT EXISTS execution_budget_reservations_open_idx
    ON execution_budget_reservations(tenant_id, project_id, execution_id, created_at)
    WHERE status = 'OPEN';

ALTER TABLE execution_budgets ENABLE ROW LEVEL SECURITY;
ALTER TABLE execution_budgets FORCE ROW LEVEL SECURITY;
ALTER TABLE execution_budget_reservations ENABLE ROW LEVEL SECURITY;
ALTER TABLE execution_budget_reservations FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS execution_budgets_scope_policy ON execution_budgets;
CREATE POLICY execution_budgets_scope_policy ON execution_budgets
    USING (
        tenant_id = current_setting('aicloud.tenant_id', true)
        AND project_id = current_setting('aicloud.project_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('aicloud.tenant_id', true)
        AND project_id = current_setting('aicloud.project_id', true)
    );

DROP POLICY IF EXISTS execution_budget_reservations_scope_policy ON execution_budget_reservations;
CREATE POLICY execution_budget_reservations_scope_policy ON execution_budget_reservations
    USING (
        tenant_id = current_setting('aicloud.tenant_id', true)
        AND project_id = current_setting('aicloud.project_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('aicloud.tenant_id', true)
        AND project_id = current_setting('aicloud.project_id', true)
    );

COMMENT ON TABLE execution_budgets IS
    'Shared execution budget counters serialized by row-level locking.';
COMMENT ON TABLE execution_budget_reservations IS
    'Attempt-scoped durable budget reservations. reservation_id should be derived from durable AttemptRef.';
