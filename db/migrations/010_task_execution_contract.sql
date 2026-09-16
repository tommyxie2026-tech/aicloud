-- R1 Task Execution Contract.
--
-- Task lifecycle history remains canonical in task_events (007). This migration
-- adds only versioned plan and run projections. Plan/run lifecycle facts MUST be
-- appended to task_events with planId/runId in the event payload; do not create
-- a second event stream.

CREATE TABLE IF NOT EXISTS execution_plans (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE RESTRICT,
    revision INTEGER NOT NULL,
    plan_json JSONB NOT NULL,
    policy_decision_json JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT execution_plans_revision_positive_check CHECK (revision >= 1),
    CONSTRAINT execution_plans_task_revision_unique UNIQUE (task_id, revision)
);

CREATE INDEX IF NOT EXISTS execution_plans_scope_task_revision_idx
    ON execution_plans(tenant_id, project_id, task_id, revision DESC);

ALTER TABLE execution_plans ENABLE ROW LEVEL SECURITY;
ALTER TABLE execution_plans FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS execution_plans_scope_policy ON execution_plans;
CREATE POLICY execution_plans_scope_policy ON execution_plans
    USING (
        tenant_id = current_setting('aicloud.tenant_id', true)
        AND project_id = current_setting('aicloud.project_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('aicloud.tenant_id', true)
        AND project_id = current_setting('aicloud.project_id', true)
    );

CREATE TABLE IF NOT EXISTS execution_runs (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE RESTRICT,
    plan_id TEXT NOT NULL REFERENCES execution_plans(id) ON DELETE RESTRICT,
    attempt INTEGER NOT NULL DEFAULT 1,
    phase TEXT NOT NULL,
    usage_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    cost_usd DOUBLE PRECISION NOT NULL DEFAULT 0,
    duration_ms BIGINT NOT NULL DEFAULT 0,
    outcome_success BOOLEAN,
    artifact_ref TEXT,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT execution_runs_attempt_positive_check CHECK (attempt >= 1),
    CONSTRAINT execution_runs_cost_nonnegative_check CHECK (cost_usd >= 0),
    CONSTRAINT execution_runs_duration_nonnegative_check CHECK (duration_ms >= 0),
    CONSTRAINT execution_runs_phase_contract_check CHECK (phase IN (
        'accepted', 'running', 'suspended', 'succeeded', 'failed', 'cancelled'
    )),
    CONSTRAINT execution_runs_plan_attempt_unique UNIQUE (plan_id, attempt)
);

CREATE INDEX IF NOT EXISTS execution_runs_scope_task_created_idx
    ON execution_runs(tenant_id, project_id, task_id, created_at, id);
CREATE INDEX IF NOT EXISTS execution_runs_scope_phase_idx
    ON execution_runs(tenant_id, project_id, phase, created_at, id);

ALTER TABLE execution_runs ENABLE ROW LEVEL SECURITY;
ALTER TABLE execution_runs FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS execution_runs_scope_policy ON execution_runs;
CREATE POLICY execution_runs_scope_policy ON execution_runs
    USING (
        tenant_id = current_setting('aicloud.tenant_id', true)
        AND project_id = current_setting('aicloud.project_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('aicloud.tenant_id', true)
        AND project_id = current_setting('aicloud.project_id', true)
    );
