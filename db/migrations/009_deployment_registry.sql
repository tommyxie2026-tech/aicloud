CREATE TABLE IF NOT EXISTS model_deployments (
    id TEXT PRIMARY KEY,
    model_id TEXT NOT NULL,
    model_version TEXT NOT NULL,
    provider TEXT NOT NULL,
    endpoint TEXT NOT NULL DEFAULT '',
    deployment_mode TEXT NOT NULL,
    region TEXT NOT NULL DEFAULT '',
    data_residency TEXT NOT NULL DEFAULT '',
    runtime TEXT NOT NULL DEFAULT '',
    quantization TEXT NOT NULL DEFAULT '',
    pricing_policy_ref TEXT NOT NULL DEFAULT '',
    health_status TEXT NOT NULL DEFAULT 'unknown',
    health_checked_at TIMESTAMPTZ,
    p95_latency_ms BIGINT NOT NULL DEFAULT 0,
    error_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
    quota_remaining BIGINT NOT NULL DEFAULT 0,
    capacity_available BIGINT NOT NULL DEFAULT 0,
    queue_depth BIGINT NOT NULL DEFAULT 0,
    service_tiers JSONB NOT NULL DEFAULT '[]'::jsonb,
    inference_efforts JSONB NOT NULL DEFAULT '[]'::jsonb,
    lifecycle_state TEXT NOT NULL DEFAULT 'discovered',
    routing_eligible BOOLEAN NOT NULL DEFAULT FALSE,
    owner_name TEXT NOT NULL DEFAULT '',
    policy_ref TEXT NOT NULL DEFAULT '',
    replacement_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT model_deployments_model_fk FOREIGN KEY (model_id) REFERENCES models(id)
);

CREATE UNIQUE INDEX IF NOT EXISTS model_deployments_endpoint_uidx
    ON model_deployments(provider, endpoint, model_id, model_version);
CREATE INDEX IF NOT EXISTS model_deployments_model_idx
    ON model_deployments(model_id, model_version);
CREATE INDEX IF NOT EXISTS model_deployments_routing_idx
    ON model_deployments(routing_eligible, lifecycle_state, health_status);
CREATE INDEX IF NOT EXISTS model_deployments_region_idx
    ON model_deployments(region, data_residency);

COMMENT ON TABLE model_deployments IS
    'Mutable runtime deployment state separated from immutable model identity. Historical model columns remain temporarily for compatibility during R5 migration.';
