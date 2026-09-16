-- ECP-C1 Task <-> Execution durable lineage.
--
-- This migration adds immutable execution parent records above the existing
-- execution_node_runtime (017) and execution_attempts (018) tables. It does not
-- replace tasks/task_events or Temporal history. All runtime-facing records are
-- tenant/project scoped and protected by PostgreSQL RLS.

CREATE TABLE IF NOT EXISTS execution_goals (
    tenant_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    goal_id TEXT NOT NULL,
    task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE RESTRICT,
    objective TEXT NOT NULL,
    acceptance_criteria JSONB NOT NULL DEFAULT '[]'::jsonb,
    constraints JSONB NOT NULL DEFAULT '{}'::jsonb,
    budget JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT execution_goals_pk PRIMARY KEY (tenant_id, project_id, goal_id),
    CONSTRAINT execution_goals_task_lineage_unique UNIQUE (
        tenant_id, project_id, goal_id, task_id
    ),
    CONSTRAINT execution_goals_objective_nonempty CHECK (length(btrim(objective)) > 0)
);

CREATE INDEX IF NOT EXISTS execution_goals_scope_task_idx
    ON execution_goals(tenant_id, project_id, task_id, created_at, goal_id);

ALTER TABLE execution_goals ENABLE ROW LEVEL SECURITY;
ALTER TABLE execution_goals FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS execution_goals_scope_policy ON execution_goals;
CREATE POLICY execution_goals_scope_policy ON execution_goals
    USING (
        tenant_id = current_setting('aicloud.tenant_id', true)
        AND project_id = current_setting('aicloud.project_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('aicloud.tenant_id', true)
        AND project_id = current_setting('aicloud.project_id', true)
    );

CREATE TABLE IF NOT EXISTS execution_plans (
    tenant_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    plan_id TEXT NOT NULL,
    task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE RESTRICT,
    goal_id TEXT NOT NULL,
    revision BIGINT NOT NULL,
    parent_revision BIGINT NOT NULL DEFAULT 0,
    plan_json JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT execution_plans_pk PRIMARY KEY (
        tenant_id, project_id, plan_id, revision
    ),
    CONSTRAINT execution_plans_task_goal_lineage_unique UNIQUE (
        tenant_id, project_id, plan_id, revision, task_id, goal_id
    ),
    CONSTRAINT execution_plans_goal_fk FOREIGN KEY (
        tenant_id, project_id, goal_id, task_id
    ) REFERENCES execution_goals(
        tenant_id, project_id, goal_id, task_id
    ) ON DELETE RESTRICT,
    CONSTRAINT execution_plans_revision_positive CHECK (revision >= 1),
    CONSTRAINT execution_plans_parent_revision_nonnegative CHECK (parent_revision >= 0),
    CONSTRAINT execution_plans_parent_revision_order CHECK (
        parent_revision = 0 OR parent_revision < revision
    )
);

CREATE INDEX IF NOT EXISTS execution_plans_scope_task_revision_idx
    ON execution_plans(tenant_id, project_id, task_id, created_at, plan_id, revision);
CREATE INDEX IF NOT EXISTS execution_plans_scope_goal_revision_idx
    ON execution_plans(tenant_id, project_id, goal_id, plan_id, revision);

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

CREATE TABLE IF NOT EXISTS executions (
    tenant_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    execution_id TEXT NOT NULL,
    task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE RESTRICT,
    goal_id TEXT NOT NULL,
    plan_id TEXT NOT NULL,
    plan_revision BIGINT NOT NULL,
    principal TEXT NOT NULL,
    phase TEXT NOT NULL,
    policy_context JSONB NOT NULL DEFAULT '{}'::jsonb,
    budget_state JSONB NOT NULL DEFAULT '{}'::jsonb,
    status_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    accounting JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT executions_pk PRIMARY KEY (tenant_id, project_id, execution_id),
    CONSTRAINT executions_goal_fk FOREIGN KEY (
        tenant_id, project_id, goal_id, task_id
    ) REFERENCES execution_goals(
        tenant_id, project_id, goal_id, task_id
    ) ON DELETE RESTRICT,
    CONSTRAINT executions_plan_fk FOREIGN KEY (
        tenant_id, project_id, plan_id, plan_revision, task_id, goal_id
    ) REFERENCES execution_plans(
        tenant_id, project_id, plan_id, revision, task_id, goal_id
    ) ON DELETE RESTRICT,
    CONSTRAINT executions_plan_revision_positive CHECK (plan_revision >= 1),
    CONSTRAINT executions_principal_nonempty CHECK (length(btrim(principal)) > 0),
    CONSTRAINT executions_phase_contract CHECK (phase IN (
        'CREATED', 'PLANNING', 'READY', 'RUNNING', 'VERIFYING',
        'REPLANNING', 'SUCCEEDED', 'FAILED', 'CANCELLED'
    )),
    CONSTRAINT executions_time_order CHECK (updated_at >= created_at)
);

CREATE INDEX IF NOT EXISTS executions_scope_task_created_idx
    ON executions(tenant_id, project_id, task_id, created_at, execution_id);
CREATE INDEX IF NOT EXISTS executions_scope_phase_updated_idx
    ON executions(tenant_id, project_id, phase, updated_at, execution_id);

ALTER TABLE executions ENABLE ROW LEVEL SECURITY;
ALTER TABLE executions FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS executions_scope_policy ON executions;
CREATE POLICY executions_scope_policy ON executions
    USING (
        tenant_id = current_setting('aicloud.tenant_id', true)
        AND project_id = current_setting('aicloud.project_id', true)
    )
    WITH CHECK (
        tenant_id = current_setting('aicloud.tenant_id', true)
        AND project_id = current_setting('aicloud.project_id', true)
    );

COMMENT ON TABLE execution_goals IS
    'Immutable execution-time snapshots of canonical Task intent.';
COMMENT ON TABLE execution_plans IS
    'Immutable, versioned bounded-DAG execution plans for a Task Goal snapshot.';
COMMENT ON TABLE executions IS
    'Durable parent execution lineage. Node mutable state and attempts remain in migrations 017/018.';
