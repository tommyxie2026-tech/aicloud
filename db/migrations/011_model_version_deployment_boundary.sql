CREATE TABLE IF NOT EXISTS model_versions (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    version TEXT NOT NULL,
    lifecycle_state TEXT NOT NULL DEFAULT 'draft',
    capabilities JSONB NOT NULL DEFAULT '[]'::jsonb,
    evaluation_version TEXT NOT NULL DEFAULT '',
    license TEXT NOT NULL DEFAULT '',
    license_evidence JSONB NOT NULL DEFAULT '{}'::jsonb,
    provenance JSONB NOT NULL DEFAULT '{}'::jsonb,
    artifact_digest TEXT NOT NULL DEFAULT '',
    approval_status TEXT NOT NULL DEFAULT 'discovered',
    risk_level TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO model_versions (
    id, name, version, lifecycle_state, capabilities, evaluation_version,
    license, license_evidence, provenance, artifact_digest, approval_status,
    risk_level, created_at, updated_at
)
SELECT
    id, name, version, lifecycle_state, capabilities, evaluation_version,
    license, license_evidence, provenance, artifact_digest, approval_status,
    risk_level, created_at, updated_at
FROM models
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    version = EXCLUDED.version,
    lifecycle_state = EXCLUDED.lifecycle_state,
    capabilities = EXCLUDED.capabilities,
    evaluation_version = EXCLUDED.evaluation_version,
    license = EXCLUDED.license,
    license_evidence = EXCLUDED.license_evidence,
    provenance = EXCLUDED.provenance,
    artifact_digest = EXCLUDED.artifact_digest,
    approval_status = EXCLUDED.approval_status,
    risk_level = EXCLUDED.risk_level,
    updated_at = EXCLUDED.updated_at;

ALTER TABLE model_deployments
    ADD COLUMN IF NOT EXISTS model_version_id TEXT,
    ADD COLUMN IF NOT EXISTS pricing JSONB NOT NULL DEFAULT '{}'::jsonb;

UPDATE model_deployments d
SET model_version_id = COALESCE(NULLIF(d.model_version_id, ''), d.model_id),
    pricing = CASE
        WHEN d.pricing = '{}'::jsonb THEN m.pricing
        ELSE d.pricing
    END
FROM models m
WHERE m.id = d.model_id;

ALTER TABLE model_deployments
    ALTER COLUMN model_version_id SET NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'model_deployments_model_version_fk'
    ) THEN
        ALTER TABLE model_deployments
            ADD CONSTRAINT model_deployments_model_version_fk
            FOREIGN KEY (model_version_id) REFERENCES model_versions(id);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS model_versions_lifecycle_idx
    ON model_versions(lifecycle_state, approval_status);
CREATE INDEX IF NOT EXISTS model_deployments_model_version_id_idx
    ON model_deployments(model_version_id);

COMMENT ON TABLE model_versions IS
    'Immutable/catalog model-version evidence. Endpoint, pricing, health, quota, capacity, region and runtime state belong to model_deployments.';
COMMENT ON COLUMN model_deployments.model_version_id IS
    'Authoritative R5 model-version identity. Legacy model_id/model_version remain temporarily for API compatibility.';
