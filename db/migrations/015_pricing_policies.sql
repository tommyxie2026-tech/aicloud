CREATE TABLE IF NOT EXISTS pricing_policies (
    id TEXT NOT NULL,
    version TEXT NOT NULL,
    deployment_id TEXT NOT NULL REFERENCES model_deployments(id),
    currency TEXT NOT NULL,
    region TEXT NOT NULL DEFAULT '',
    input_per_million DOUBLE PRECISION NOT NULL DEFAULT 0,
    output_per_million DOUBLE PRECISION NOT NULL DEFAULT 0,
    cache_hit_per_million DOUBLE PRECISION NOT NULL DEFAULT 0,
    cache_miss_per_million DOUBLE PRECISION NOT NULL DEFAULT 0,
    context_bands JSONB NOT NULL DEFAULT '[]'::jsonb,
    batch_factor DOUBLE PRECISION NOT NULL DEFAULT 0,
    async_factor DOUBLE PRECISION NOT NULL DEFAULT 0,
    service_tier_factors JSONB NOT NULL DEFAULT '{}'::jsonb,
    inference_effort_factors JSONB NOT NULL DEFAULT '{}'::jsonb,
    capacity_pricing JSONB NOT NULL DEFAULT '{}'::jsonb,
    self_hosted_allocation JSONB NOT NULL DEFAULT '{}'::jsonb,
    effective_from TIMESTAMPTZ NOT NULL,
    effective_to TIMESTAMPTZ,
    evidence_ref TEXT NOT NULL DEFAULT '',
    digest TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, version)
);

CREATE INDEX IF NOT EXISTS pricing_policies_deployment_effective_idx
    ON pricing_policies(deployment_id, effective_from DESC, created_at DESC);

CREATE INDEX IF NOT EXISTS pricing_policies_effective_window_idx
    ON pricing_policies(deployment_id, effective_from, effective_to);

COMMENT ON TABLE pricing_policies IS
    'Immutable versioned pricing evidence scoped to a concrete Deployment. Historical route and cost evidence must retain the exact policy version used.';
