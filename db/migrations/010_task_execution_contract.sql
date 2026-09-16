-- R1 Task Execution Contract projections and append-only events.
CREATE TABLE IF NOT EXISTS execution_plans (
  id VARCHAR(64) PRIMARY KEY,
  task_id VARCHAR(64) NOT NULL,
  revision INT NOT NULL,
  plan_json JSON NOT NULL,
  policy_decision_json JSON NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_execution_plan_revision (task_id, revision)
);

CREATE TABLE IF NOT EXISTS execution_runs (
  id VARCHAR(64) PRIMARY KEY,
  task_id VARCHAR(64) NOT NULL,
  plan_id VARCHAR(64) NOT NULL,
  attempt INT NOT NULL DEFAULT 1,
  phase VARCHAR(32) NOT NULL,
  cost_usd DECIMAL(18,8) NOT NULL DEFAULT 0,
  duration_ms BIGINT NOT NULL DEFAULT 0,
  outcome_success BOOLEAN NULL,
  started_at TIMESTAMP NULL,
  finished_at TIMESTAMP NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_execution_run_attempt (plan_id, attempt)
);

CREATE TABLE IF NOT EXISTS execution_events (
  id VARCHAR(64) PRIMARY KEY,
  event_type VARCHAR(64) NOT NULL,
  task_id VARCHAR(64) NOT NULL,
  plan_id VARCHAR(64) NULL,
  run_id VARCHAR(64) NULL,
  trace_id VARCHAR(64) NULL,
  tenant_id VARCHAR(128) NOT NULL,
  actor VARCHAR(256) NOT NULL,
  payload_json JSON NULL,
  occurred_at TIMESTAMP(6) NOT NULL,
  INDEX idx_execution_events_task_time (task_id, occurred_at),
  INDEX idx_execution_events_run_time (run_id, occurred_at)
);
