ALTER TABLE models ADD COLUMN IF NOT EXISTS version TEXT NOT NULL DEFAULT 'v1';
ALTER TABLE models ADD COLUMN IF NOT EXISTS endpoint TEXT NOT NULL DEFAULT '';
ALTER TABLE models ADD COLUMN IF NOT EXISTS deployment_mode TEXT NOT NULL DEFAULT '';
ALTER TABLE models ADD COLUMN IF NOT EXISTS lifecycle_state TEXT NOT NULL DEFAULT 'draft';
ALTER TABLE models ADD COLUMN IF NOT EXISTS health_status TEXT NOT NULL DEFAULT 'unknown';
ALTER TABLE models ADD COLUMN IF NOT EXISTS health_checked_at TIMESTAMPTZ;
ALTER TABLE models ADD COLUMN IF NOT EXISTS p95_latency_ms BIGINT NOT NULL DEFAULT 0;
ALTER TABLE models ADD COLUMN IF NOT EXISTS error_rate DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE models ADD COLUMN IF NOT EXISTS quota_remaining BIGINT NOT NULL DEFAULT 0;
ALTER TABLE models ADD COLUMN IF NOT EXISTS capacity_available BIGINT NOT NULL DEFAULT 0;
ALTER TABLE models ADD COLUMN IF NOT EXISTS queue_depth BIGINT NOT NULL DEFAULT 0;
ALTER TABLE models ADD COLUMN IF NOT EXISTS service_tiers JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE models ADD COLUMN IF NOT EXISTS inference_efforts JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE models ADD COLUMN IF NOT EXISTS evaluation_version TEXT NOT NULL DEFAULT '';
ALTER TABLE models ADD COLUMN IF NOT EXISTS license_evidence JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE models ADD COLUMN IF NOT EXISTS provenance JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE models ADD COLUMN IF NOT EXISTS artifact_digest TEXT NOT NULL DEFAULT '';
ALTER TABLE models ADD COLUMN IF NOT EXISTS approval_status TEXT NOT NULL DEFAULT 'discovered';
ALTER TABLE models ADD COLUMN IF NOT EXISTS data_residency TEXT NOT NULL DEFAULT '';
ALTER TABLE models ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE UNIQUE INDEX IF NOT EXISTS models_provider_name_version_uidx ON models(provider, name, version);
CREATE INDEX IF NOT EXISTS models_runtime_state_idx ON models(lifecycle_state, health_status, approval_status);

ALTER TABLE tasks ADD COLUMN IF NOT EXISTS estimated_cost DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS actual_cost DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS currency TEXT NOT NULL DEFAULT '';
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS route_decision_id TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS route_decisions (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    selected JSONB NOT NULL,
    candidates JSONB NOT NULL DEFAULT '[]'::jsonb,
    reason TEXT NOT NULL,
    fallback_chain JSONB NOT NULL DEFAULT '[]'::jsonb,
    evidence_version TEXT NOT NULL DEFAULT '',
    policy_version TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS route_decisions_task_idx ON route_decisions(task_id, created_at);

CREATE TABLE IF NOT EXISTS cost_events (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    trace_id TEXT NOT NULL,
    component TEXT NOT NULL,
    provider TEXT NOT NULL DEFAULT '',
    model_id TEXT NOT NULL DEFAULT '',
    model_version TEXT NOT NULL DEFAULT '',
    quantity DOUBLE PRECISION NOT NULL,
    unit TEXT NOT NULL,
    unit_price DOUBLE PRECISION NOT NULL,
    amount DOUBLE PRECISION NOT NULL,
    currency TEXT NOT NULL,
    attempt INTEGER NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS cost_events_task_idx ON cost_events(task_id, created_at);
CREATE INDEX IF NOT EXISTS cost_events_trace_idx ON cost_events(trace_id, created_at);
