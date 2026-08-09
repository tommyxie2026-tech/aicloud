# AI Cloud Evaluation and Release Gate Contract

> Status: S0 Contract Freeze

## 1. Purpose

Define how model, prompt, agent and workflow quality evidence becomes an enforceable release decision rather than an informational benchmark.

## 2. Distinguish Admission from Evaluation

```text
Admission
  answers: is this ModelVersion legally/security eligible to enter AI Cloud?

Evaluation
  answers: is this version/configuration good enough for this use/release?
```

Admission is based on license, provenance, artifact integrity and security evidence. Evaluation is based on quality, safety, reliability, latency, cost and task outcomes.

## 3. Evaluation Levels

```text
L1 Offline Evaluation
  -> golden datasets / regression

L2 Pre-Production Evaluation
  -> shadow / canary / controlled traffic

L3 Production Evaluation
  -> real task traces / outcomes / human intervention / cost
```

## 4. Evaluation Subject

Evaluation identity is not only a model name. It is a versioned configuration tuple:

```text
ModelVersion
PromptVersion
AgentVersion
WorkflowVersion
PolicyVersion
Tool/Sandbox Profile
DatasetVersion
EvaluatorVersion
```

The exact tuple used must be preserved for reproducibility.

## 5. EvaluationRun

```yaml
evaluation_run:
  evaluation_run_id: string
  tenant_id: string
  project_id: string
  task_id: string?
  subject_ref: object
  dataset_id: string
  dataset_version: string
  evaluator_id: string
  evaluator_version: string
  config_digest: string
  metrics: object
  evidence_refs: []string
  started_at: timestamp
  completed_at: timestamp
```

## 6. Gate Metrics

A gate may use:

```text
quality score
safety score
reliability score
task success rate
hallucination/error rate
latency p50/p95
cost per successful task
human intervention rate
policy violation rate
tool failure rate
```

Thresholds are versioned.

## 7. GateDecision

```yaml
gate_decision:
  gate_decision_id: string
  evaluation_run_id: string
  subject_ref: object
  decision: pass | fail | conditional
  threshold_version: string
  reasons: []string
  approved_for: []string
  expires_at: timestamp?
  created_at: timestamp
```

A failed mandatory gate blocks promotion. A conditional pass must state explicit limits such as tenant, traffic percentage, task class or expiration.

## 8. Promotion

Production eligibility requires both:

```text
Admission Eligible
AND
Required Evaluation Gates Passed
```

Release/promotion is a governed state transition and produces audit evidence.

## 9. Regression

New versions are compared to an approved baseline. A version may fail even when absolute metrics are acceptable if a protected metric regresses beyond allowed tolerance.

## 10. Production Feedback

Production Task traces are sampled into evaluation candidates using data-governance rules. Production outcomes close the loop:

```text
Production Task
  -> Trace/Outcome
  -> Evaluation Dataset Candidate
  -> Regression Run
  -> GateDecision
  -> Router/Release Eligibility Update
```

## 11. Router Integration

Router consumes only the resulting eligibility/performance evidence. It does not decide gate thresholds.

A candidate that failed a mandatory gate is rejected before soft routing score.

## 12. Data Governance

Evaluation datasets inherit sensitivity and tenant rules. Production data cannot automatically become a global benchmark dataset. Dataset lineage and consent/retention policy are required.

## 13. Reproducibility

Every EvaluationRun stores versions/digests of:

- subject configuration;
- dataset;
- evaluator;
- thresholds;
- relevant runtime parameters.

A release decision must be reproducible from stored evidence.

## 14. Rollback

Production monitoring can trigger rollback when protected metrics breach thresholds. Rollback creates a new decision/event; historical GateDecisions are not rewritten.

## 15. Acceptance Criteria

- Admission and Evaluation are separate state machines/records.
- Mandatory failed gate prevents promotion/routing eligibility.
- Evaluation configuration is reproducible by version/digest.
- Baseline regression is tested.
- Production feedback can create governed evaluation candidates.
- Tenant-sensitive evaluation data is not promoted to global scope without policy.
- GateDecision is immutable and auditable.
- Rollback is explainable from evaluation evidence.