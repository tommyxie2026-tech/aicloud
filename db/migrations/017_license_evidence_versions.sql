CREATE TABLE IF NOT EXISTS license_evidence_versions (
    id TEXT NOT NULL,
    version TEXT NOT NULL,
    model_version_id TEXT NOT NULL REFERENCES model_versions(id) ON DELETE CASCADE,
    license_id TEXT NOT NULL,
    weight_availability TEXT NOT NULL,
    commercial_use TEXT NOT NULL,
    hosted_service TEXT NOT NULL,
    redistribution TEXT NOT NULL,
    derivative_works TEXT NOT NULL,
    attribution_required BOOLEAN NOT NULL DEFAULT FALSE,
    notice_required BOOLEAN NOT NULL DEFAULT FALSE,
    thresholds JSONB NOT NULL DEFAULT '[]'::jsonb,
    revenue_share_ref TEXT NOT NULL DEFAULT '',
    additional_fee_ref TEXT NOT NULL DEFAULT '',
    allowed_geographies JSONB NOT NULL DEFAULT '[]'::jsonb,
    blocked_geographies JSONB NOT NULL DEFAULT '[]'::jsonb,
    allowed_customer_tags JSONB NOT NULL DEFAULT '[]'::jsonb,
    blocked_customer_tags JSONB NOT NULL DEFAULT '[]'::jsonb,
    effective_from TIMESTAMPTZ NOT NULL,
    effective_to TIMESTAMPTZ,
    review_after TIMESTAMPTZ,
    evidence_ref TEXT NOT NULL,
    evidence_digest TEXT NOT NULL,
    reviewer TEXT NOT NULL DEFAULT '',
    approval_state TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, version),
    CONSTRAINT license_evidence_permission_check CHECK (
        weight_availability IN ('allowed','conditional','forbidden') AND
        commercial_use IN ('allowed','conditional','forbidden') AND
        hosted_service IN ('allowed','conditional','forbidden') AND
        redistribution IN ('allowed','conditional','forbidden') AND
        derivative_works IN ('allowed','conditional','forbidden')
    ),
    CONSTRAINT license_evidence_approval_check CHECK (
        approval_state IN ('pending','approved','rejected','revoked')
    ),
    CONSTRAINT license_evidence_effective_window_check CHECK (
        effective_to IS NULL OR effective_to > effective_from
    )
);

CREATE INDEX IF NOT EXISTS license_evidence_model_effective_idx
    ON license_evidence_versions(model_version_id, effective_from DESC, created_at DESC);
CREATE INDEX IF NOT EXISTS license_evidence_approval_idx
    ON license_evidence_versions(model_version_id, approval_state, effective_from DESC);

COMMENT ON TABLE license_evidence_versions IS
    'Immutable authoritative commercial-license evidence. New upstream terms create a new version; historical routing evidence must never be rewritten.';
