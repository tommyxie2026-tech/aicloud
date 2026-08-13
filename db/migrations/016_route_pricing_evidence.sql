CREATE TABLE IF NOT EXISTS route_pricing_evidence (
    route_decision_id TEXT NOT NULL REFERENCES route_decisions(id) ON DELETE CASCADE,
    deployment_id TEXT NOT NULL REFERENCES model_deployments(id),
    policy_id TEXT NOT NULL,
    policy_version TEXT NOT NULL,
    policy_digest TEXT NOT NULL DEFAULT '',
    policy_snapshot JSONB NOT NULL,
    quote JSONB NOT NULL,
    selected BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (route_decision_id, deployment_id),
    CONSTRAINT route_pricing_evidence_policy_fk
        FOREIGN KEY (policy_id, policy_version)
        REFERENCES pricing_policies(id, version)
);

CREATE INDEX IF NOT EXISTS route_pricing_evidence_policy_idx
    ON route_pricing_evidence(policy_id, policy_version);
CREATE INDEX IF NOT EXISTS route_pricing_evidence_selected_idx
    ON route_pricing_evidence(route_decision_id, selected);

CREATE OR REPLACE FUNCTION capture_route_pricing_evidence()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    deployment_id_value TEXT;
    estimated_cost_value DOUBLE PRECISION;
    policy_record pricing_policies%ROWTYPE;
BEGIN
    deployment_id_value := COALESCE(NEW.selected->>'deploymentId', '');
    IF deployment_id_value = '' THEN
        RETURN NEW;
    END IF;

    SELECT * INTO policy_record
    FROM pricing_policies
    WHERE deployment_id = deployment_id_value
      AND effective_from <= NEW.created_at
      AND (effective_to IS NULL OR effective_to > NEW.created_at)
    ORDER BY effective_from DESC, created_at DESC, version DESC
    LIMIT 1;

    IF NOT FOUND THEN
        RETURN NEW;
    END IF;

    estimated_cost_value := COALESCE((NEW.selected->>'estimatedCost')::DOUBLE PRECISION, 0);

    INSERT INTO route_pricing_evidence (
        route_decision_id, deployment_id, policy_id, policy_version,
        policy_digest, policy_snapshot, quote, selected, created_at
    ) VALUES (
        NEW.id,
        deployment_id_value,
        policy_record.id,
        policy_record.version,
        policy_record.digest,
        to_jsonb(policy_record),
        jsonb_build_object(
            'policyId', policy_record.id,
            'policyVersion', policy_record.version,
            'policyDigest', policy_record.digest,
            'deploymentId', deployment_id_value,
            'currency', policy_record.currency,
            'components', '[]'::jsonb,
            'total', estimated_cost_value,
            'quotedAt', NEW.created_at
        ),
        TRUE,
        NEW.created_at
    );
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS route_pricing_evidence_capture ON route_decisions;
CREATE TRIGGER route_pricing_evidence_capture
AFTER INSERT ON route_decisions
FOR EACH ROW
EXECUTE FUNCTION capture_route_pricing_evidence();

COMMENT ON TABLE route_pricing_evidence IS
    'Immutable route-time pricing evidence captured atomically with RouteDecision. The policy snapshot and exact version remain replayable after later pricing changes.';
