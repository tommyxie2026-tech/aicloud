CREATE TABLE IF NOT EXISTS trace_events (
    id TEXT PRIMARY KEY,
    trace_id TEXT NOT NULL,
    task_id TEXT NOT NULL,
    span_id TEXT NOT NULL,
    parent_span_id TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL,
    kind TEXT NOT NULL,
    status TEXT NOT NULL,
    message TEXT NOT NULL DEFAULT '',
    attributes JSONB NOT NULL DEFAULT '{}'::jsonb,
    input_digest TEXT NOT NULL DEFAULT '',
    output_digest TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ NOT NULL,
    ended_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS trace_events_task_idx ON trace_events(task_id, started_at);
CREATE INDEX IF NOT EXISTS trace_events_trace_idx ON trace_events(trace_id, started_at);

CREATE TABLE IF NOT EXISTS evaluation_runs (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    trace_id TEXT NOT NULL,
    config JSONB NOT NULL,
    config_digest TEXT NOT NULL,
    raw_output_digest TEXT NOT NULL DEFAULT '',
    metrics JSONB NOT NULL,
    thresholds JSONB NOT NULL,
    gate_result JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS evaluation_runs_task_idx ON evaluation_runs(task_id, created_at);
CREATE INDEX IF NOT EXISTS evaluation_runs_config_idx ON evaluation_runs(config_digest, created_at);

CREATE TABLE IF NOT EXISTS model_admission_evidence (
    id TEXT PRIMARY KEY,
    model_id TEXT NOT NULL,
    model_version TEXT NOT NULL,
    status TEXT NOT NULL,
    license_id TEXT NOT NULL,
    license_text_ref TEXT NOT NULL,
    source_ref TEXT NOT NULL,
    upstream_models JSONB NOT NULL DEFAULT '[]'::jsonb,
    dataset_refs JSONB NOT NULL DEFAULT '[]'::jsonb,
    commercial_use_allowed BOOLEAN NOT NULL DEFAULT FALSE,
    hosted_service_allowed BOOLEAN NOT NULL DEFAULT FALSE,
    redistribution_allowed BOOLEAN NOT NULL DEFAULT FALSE,
    notice_required BOOLEAN NOT NULL DEFAULT FALSE,
    notice_ref TEXT NOT NULL DEFAULT '',
    artifact_digest TEXT NOT NULL DEFAULT '',
    artifact_signature TEXT NOT NULL DEFAULT '',
    security_scan_ref TEXT NOT NULL DEFAULT '',
    reviewer TEXT NOT NULL,
    reviewed_at TIMESTAMPTZ,
    evidence_digest TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS model_admission_evidence_digest_uidx ON model_admission_evidence(model_id, model_version, evidence_digest);
CREATE INDEX IF NOT EXISTS model_admission_evidence_model_idx ON model_admission_evidence(model_id, model_version, created_at);
