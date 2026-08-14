CREATE TABLE IF NOT EXISTS deployment_lifecycle_events (
    id TEXT PRIMARY KEY,
    deployment_id TEXT NOT NULL REFERENCES model_deployments(id),
    from_state TEXT NOT NULL,
    to_state TEXT NOT NULL,
    announced_at TIMESTAMPTZ,
    effective_at TIMESTAMPTZ NOT NULL,
    evidence_ref TEXT NOT NULL DEFAULT '',
    replacement_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    quota_remaining BIGINT,
    rate_limit_ref TEXT NOT NULL DEFAULT '',
    routing_eligible BOOLEAN NOT NULL,
    migration_state TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS deployment_lifecycle_events_deployment_idx
    ON deployment_lifecycle_events(deployment_id, effective_at, created_at);
