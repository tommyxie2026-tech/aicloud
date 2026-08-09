-- AI Cloud v0.1 reference schema.
-- Production migrations should be generated from this contract and applied as forward-only migrations.

CREATE TABLE tenants (
  id text PRIMARY KEY,
  organization_id text,
  name text NOT NULL,
  status text NOT NULL DEFAULT 'active',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  version bigint NOT NULL DEFAULT 1
);

CREATE TABLE projects (
  id text PRIMARY KEY,
  tenant_id text NOT NULL REFERENCES tenants(id),
  name text NOT NULL,
  status text NOT NULL DEFAULT 'active',
  default_policy_id text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  version bigint NOT NULL DEFAULT 1,
  UNIQUE (tenant_id, name)
);

CREATE TABLE subjects (
  id text PRIMARY KEY,
  tenant_id text NOT NULL REFERENCES tenants(id),
  type text NOT NULL CHECK (type IN ('user','service','agent')),
  external_subject text NOT NULL,
  status text NOT NULL DEFAULT 'active',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  version bigint NOT NULL DEFAULT 1,
  UNIQUE (tenant_id, type, external_subject)
);

CREATE TABLE models (
  id text PRIMARY KEY,
  owner_tenant_id text REFERENCES tenants(id),
  name text NOT NULL,
  visibility text NOT NULL CHECK (visibility IN ('global','private','restricted')),
  description text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  version bigint NOT NULL DEFAULT 1
);

CREATE TABLE model_versions (
  id text PRIMARY KEY,
  model_id text NOT NULL REFERENCES models(id),
  owner_tenant_id text REFERENCES tenants(id),
  provider_id text NOT NULL,
  provider_model_ref text NOT NULL,
  version_ref text,
  deployment_mode text NOT NULL,
  capabilities jsonb NOT NULL DEFAULT '{}'::jsonb,
  context_limits jsonb NOT NULL DEFAULT '{}'::jsonb,
  pricing jsonb NOT NULL DEFAULT '{}'::jsonb,
  residency jsonb NOT NULL DEFAULT '{}'::jsonb,
  license jsonb NOT NULL DEFAULT '{}'::jsonb,
  provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
  risk_level text NOT NULL DEFAULT 'medium',
  lifecycle_state text NOT NULL DEFAULT 'draft',
  admission_state text NOT NULL DEFAULT 'discovered',
  artifact_digest text,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_model_versions_route ON model_versions(model_id, admission_state, lifecycle_state);

CREATE TABLE provider_endpoints (
  id text PRIMARY KEY,
  tenant_id text REFERENCES tenants(id),
  provider_type text NOT NULL,
  endpoint_ref text NOT NULL,
  region text,
  credential_ref text,
  config jsonb NOT NULL DEFAULT '{}'::jsonb,
  enabled boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  version bigint NOT NULL DEFAULT 1
);

CREATE TABLE tasks (
  id text PRIMARY KEY,
  tenant_id text NOT NULL REFERENCES tenants(id),
  project_id text NOT NULL REFERENCES projects(id),
  agent_id text NOT NULL,
  subject_id text NOT NULL REFERENCES subjects(id),
  trace_id text NOT NULL,
  idempotency_key text,
  goal text NOT NULL,
  input jsonb NOT NULL DEFAULT '{}'::jsonb,
  constraints jsonb NOT NULL DEFAULT '{}'::jsonb,
  status text NOT NULL,
  result jsonb,
  failure_code text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  version bigint NOT NULL DEFAULT 1
);
CREATE UNIQUE INDEX uq_tasks_idempotency ON tasks(tenant_id, project_id, idempotency_key) WHERE idempotency_key IS NOT NULL;
CREATE INDEX idx_tasks_tenant_project_status ON tasks(tenant_id, project_id, status, created_at DESC);

CREATE TABLE task_events (
  id text PRIMARY KEY,
  tenant_id text NOT NULL REFERENCES tenants(id),
  project_id text NOT NULL REFERENCES projects(id),
  task_id text NOT NULL REFERENCES tasks(id),
  sequence bigint NOT NULL,
  event_type text NOT NULL,
  payload jsonb NOT NULL DEFAULT '{}'::jsonb,
  occurred_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (task_id, sequence)
);
CREATE INDEX idx_task_events_tenant_task ON task_events(tenant_id, task_id, sequence);

CREATE TABLE route_decisions (
  id text PRIMARY KEY,
  tenant_id text NOT NULL REFERENCES tenants(id),
  project_id text NOT NULL REFERENCES projects(id),
  task_id text NOT NULL REFERENCES tasks(id),
  trace_id text NOT NULL,
  request_hash text NOT NULL,
  selected_model_version_id text REFERENCES model_versions(id),
  selected_provider_endpoint_id text REFERENCES provider_endpoints(id),
  eligible_candidates jsonb NOT NULL DEFAULT '[]'::jsonb,
  rejected_candidates jsonb NOT NULL DEFAULT '[]'::jsonb,
  score_breakdown jsonb NOT NULL DEFAULT '{}'::jsonb,
  fallback_chain jsonb NOT NULL DEFAULT '[]'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE approvals (
  id text PRIMARY KEY,
  tenant_id text NOT NULL REFERENCES tenants(id),
  project_id text NOT NULL REFERENCES projects(id),
  task_id text NOT NULL REFERENCES tasks(id),
  reason text NOT NULL,
  risk_level text NOT NULL,
  action_digest text NOT NULL,
  requested_by text NOT NULL,
  decided_by text,
  decision text NOT NULL DEFAULT 'pending',
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  decided_at timestamptz
);

CREATE TABLE tool_invocations (
  id text PRIMARY KEY,
  tenant_id text NOT NULL REFERENCES tenants(id),
  project_id text NOT NULL REFERENCES projects(id),
  task_id text NOT NULL REFERENCES tasks(id),
  tool_id text NOT NULL,
  action text NOT NULL,
  resource_ref text NOT NULL,
  policy_decision_id text NOT NULL,
  credential_lease_ref text,
  idempotency_key text NOT NULL,
  input_hash text NOT NULL,
  status text NOT NULL,
  result_ref text,
  started_at timestamptz NOT NULL DEFAULT now(),
  ended_at timestamptz,
  UNIQUE (tenant_id, project_id, tool_id, idempotency_key)
);

CREATE TABLE cost_events (
  id text PRIMARY KEY,
  tenant_id text NOT NULL REFERENCES tenants(id),
  project_id text NOT NULL REFERENCES projects(id),
  task_id text NOT NULL REFERENCES tasks(id),
  trace_id text NOT NULL,
  source_type text NOT NULL,
  source_id text NOT NULL,
  provider_id text,
  model_version_id text REFERENCES model_versions(id),
  usage jsonb NOT NULL DEFAULT '{}'::jsonb,
  currency text NOT NULL DEFAULT 'USD',
  amount numeric(20,8) NOT NULL DEFAULT 0,
  pricing_version text,
  occurred_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_cost_events_tenant_task ON cost_events(tenant_id, task_id, occurred_at);

CREATE TABLE audit_events (
  id text PRIMARY KEY,
  tenant_id text NOT NULL REFERENCES tenants(id),
  project_id text REFERENCES projects(id),
  trace_id text NOT NULL,
  subject_id text NOT NULL,
  action text NOT NULL,
  resource_type text NOT NULL,
  resource_id text,
  decision text,
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  occurred_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_audit_events_tenant_trace ON audit_events(tenant_id, trace_id, occurred_at);

CREATE TABLE artifacts (
  id text PRIMARY KEY,
  tenant_id text NOT NULL REFERENCES tenants(id),
  project_id text NOT NULL REFERENCES projects(id),
  task_id text NOT NULL REFERENCES tasks(id),
  kind text NOT NULL,
  object_key text NOT NULL,
  digest text NOT NULL,
  size_bytes bigint NOT NULL DEFAULT 0,
  classification text NOT NULL DEFAULT 'internal',
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_artifacts_tenant_task ON artifacts(tenant_id, task_id, created_at);

-- RLS: application transaction must SET LOCAL aicloud.tenant_id = '<trusted tenant id>'.
ALTER TABLE tasks ENABLE ROW LEVEL SECURITY;
ALTER TABLE task_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE approvals ENABLE ROW LEVEL SECURITY;
ALTER TABLE tool_invocations ENABLE ROW LEVEL SECURITY;
ALTER TABLE cost_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE artifacts ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_tasks ON tasks USING (tenant_id = current_setting('aicloud.tenant_id', true));
CREATE POLICY tenant_task_events ON task_events USING (tenant_id = current_setting('aicloud.tenant_id', true));
CREATE POLICY tenant_approvals ON approvals USING (tenant_id = current_setting('aicloud.tenant_id', true));
CREATE POLICY tenant_tool_invocations ON tool_invocations USING (tenant_id = current_setting('aicloud.tenant_id', true));
CREATE POLICY tenant_cost_events ON cost_events USING (tenant_id = current_setting('aicloud.tenant_id', true));
CREATE POLICY tenant_audit_events ON audit_events USING (tenant_id = current_setting('aicloud.tenant_id', true));
CREATE POLICY tenant_artifacts ON artifacts USING (tenant_id = current_setting('aicloud.tenant_id', true));
