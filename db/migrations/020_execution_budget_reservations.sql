-- ECP-C4 durable distributed budget coordination.
--
-- Reservations are attempt-scoped and share the same tenant/project RLS boundary
-- as executions. The executions.budget_state JSON remains the canonical counter
-- projection; this table provides durable reservation identity and crash
-- reconciliation evidence for distributed workers.

CREATE TABLE IF NOT EXISTS execution_budget_reservations (
    tenant_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    reservation_id TEXT NOT NULL,
    execution_id TEXT NOT NULL,
    node_id TEXT NOT NULL,

    estimate_cost DOUBLE PRECISION NOT NULL DEFAULT 0,
    estimate_node_attempts INTEGER NOT NULL DEFAULT 0,
    estimate_frontier_calls INTEGER NOT NULL DEFAULT 0,
    estimate_tool_calls INTEGER NOT NULL DEFAULT 0,

    actual_cost DOUBLE PRECISION,
    actual_node_attempts INTEGER,
    actual_frontier_calls INTEGER,
    actual_tool_calls INTEGER,

    status TEXT NOT NULL DEFAULT 'OPEN',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT execution_budget_reservations_pk PRIMARY KEY (
        tenant_id, project_id, reservation_id
    ),
    CONSTRAINT execution_budget_reservations_execution_fk FOREIGN KEY (
        tenant_id, project_id, execution_id
    ) REFERENCES executions(
        tenant_id, project_id, execution_id
    ) ON DELETE RESTRICT,
    CONSTRAINT execution_budget_reservations_identity_nonempty CHECK (
        length(btrim(reservation_id)) > 0
        AND length(btrim(execution_id)) > 0
        AND length(btrim(node_id)) > 0
    ),
    CONSTRAINT execution_budget_reservations_estimate_nonnegative CHECK (
        estimate_cost >= 0
        AND estimate_node_attempts >= 0
        AND estimate_frontier_calls >= 0
        AND estimate_tool_calls >= 0
    ),
    CONSTRAINT execution_budget_reservations_actual_nonnegative CHECK (
        (actual_cost IS NULL OR actual_cost >= 0)
        AND (actual_node_attempts IS NULL OR actual_node_attempts >= 0)
        AND (actual_frontier_calls IS NULL OR actual_frontier_calls >= 0)
        AND (actual_tool_calls IS NULL OR actual_tool_calls >= 0)
    ),
    CONSTRAINT execution_budget_reservations_status_contract CHECK (
        status IN ('OPEN', 'SETTLED', 'RELEASED', 'ABANDONED_ESTIMATE')
    ),
    CONSTRAINT execution_budget_reservations_close_contract CHECK (
        (status = 'OPEN' AND closed_at IS NULL)
        OR (status <> 'OPEN' AND closed_at IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS execution_budget_reservations_execution_status_idx
    ON execution_budget_reservations(
        tenant_id, project_id, execution_id, status, created_at, reservation_id
    );

CREATE INDEX IF NOT EXISTS execution_budget_reservations_execution_node_idx
    ON execution_budget_reservations(
        tenant_id, project_id, execution_id, node_id, created_at, reservation_id
    );

ALTER TABLE execution_budget_reservations ENABLE ROW LEVEL SECURITY;
ALTER TABLE execution_budget_reservations FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS execution_budget_reservations_scope_policy
    ON execution_budget_reservations;
CREATE POLICY execution_budget_reservations_scope_policy
    ON execution_budget_reservations
    USING (
        tenant_id = current_setting('aicloud.tenant_id', true)
        AND project_id = current_setting('aicloud.project_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('aicloud.tenant_id', true)
        AND project_id = current_setting('aicloud.project_id', true)
    );

COMMENT ON TABLE execution_budget_reservations IS
    'Durable attempt-scoped budget reservations. OPEN rows are reconciled against abandoned attempts before subsequent reservations.';
