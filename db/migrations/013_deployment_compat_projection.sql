CREATE OR REPLACE FUNCTION fill_deployment_compat_projection()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    legacy_pricing JSONB;
BEGIN
    IF NEW.model_version_id IS NULL OR NEW.model_version_id = '' THEN
        NEW.model_version_id := NEW.model_id;
    END IF;

    IF NEW.pricing IS NULL OR NEW.pricing = '{}'::jsonb THEN
        SELECT pricing INTO legacy_pricing
        FROM models
        WHERE id = NEW.model_id;
        IF legacy_pricing IS NOT NULL THEN
            NEW.pricing := legacy_pricing;
        END IF;
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS model_deployments_fill_compat_projection ON model_deployments;
CREATE TRIGGER model_deployments_fill_compat_projection
BEFORE INSERT OR UPDATE ON model_deployments
FOR EACH ROW
EXECUTE FUNCTION fill_deployment_compat_projection();
