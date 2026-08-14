CREATE OR REPLACE FUNCTION sync_model_version_from_legacy_model()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    INSERT INTO model_versions (
        id, name, version, lifecycle_state, capabilities, evaluation_version,
        license, license_evidence, provenance, artifact_digest, approval_status,
        risk_level, created_at, updated_at
    ) VALUES (
        NEW.id, NEW.name, NEW.version, NEW.lifecycle_state, NEW.capabilities,
        NEW.evaluation_version, NEW.license, NEW.license_evidence, NEW.provenance,
        NEW.artifact_digest, NEW.approval_status, NEW.risk_level,
        NEW.created_at, NEW.updated_at
    )
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
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS models_sync_model_versions ON models;
CREATE TRIGGER models_sync_model_versions
AFTER INSERT OR UPDATE ON models
FOR EACH ROW
EXECUTE FUNCTION sync_model_version_from_legacy_model();
