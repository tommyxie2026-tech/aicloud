CREATE TABLE IF NOT EXISTS route_pricing_evidence (
    route_decision_id TEXT NOT NULL REFERENCES route_decisions(id) ON DELETE CASCADE,
    deployment_id TEXT NOT NULL REFERENCES model_deployments(id),
    policy_id TEXT NOT NULL,
    policy_version TEXT NOT NULL,
    policy_digest TEXT NOT NULL DEFAULT '',
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

COMMENT ON TABLE route_pricing_evidence IS
    'Immutable route-time pricing evidence used to replay historical task-cost predictions without consulting current pricing.';
