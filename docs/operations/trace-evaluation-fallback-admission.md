# Trace, Evaluation, Fallback, and Model Admission

## Scope

This document describes the v0.1 evidence and reliability baseline added after the operational Model Registry, PostgreSQL persistence, governed routing, CostEvent ledger, Tool Gateway, and Sandbox planning stages.

Implemented capabilities:

- immutable Task trace events;
- PostgreSQL persistence for Trace, EvaluationRun, and ModelAdmissionEvidence;
- reproducible evaluation configuration digests;
- quality, safety, reliability, latency, cost, intervention, and task-success release gates;
- model Circuit Breaker with closed, open, and half-open states;
- selected-model and ordered-fallback execution;
- retryable error classification;
- model-attempt Trace evidence;
- successful model-token CostEvent accounting;
- immutable license, provenance, artifact, security-scan, reviewer, and approval evidence;
- routing-time model admission checks.

## Task trace

Each Task has one Trace ID. The current implementation records:

- Task creation;
- route selection or route failure;
- model attempts, including skipped open circuits and fallback attempts;
- Tool Gateway execution results;
- evaluation runs and release-gate results.

Trace events include:

```text
Trace ID
Task ID
Span ID and optional parent Span ID
name and kind
status and message
attributes
input and output digests
start and end time
```

Trace events are append-only. PostgreSQL indexes support retrieval by Task or Trace.

Query a Task trace:

```http
GET /api/v1/tasks/{taskId}/trace
```

The current trace store is compatible with OpenTelemetry concepts but does not yet export OTLP spans. OTLP export remains the next observability integration step.

## Reproducible evaluation

An evaluation configuration records exact execution evidence:

- Model ID, immutable version, Provider, and Endpoint ID;
- Prompt and Workflow versions;
- Tool versions and permission version;
- Token and time budgets;
- Retry and compaction versions;
- Sandbox profile;
- Dataset ID and version;
- Evaluator ID and version;
- additional execution parameters.

Maps are converted to sorted key-value arrays before hashing. This ensures logically identical configurations produce the same SHA-256 digest regardless of map iteration order.

The evaluation run has a unique run ID while retaining the stable configuration digest. Repeated regression runs with the same configuration therefore remain distinct and comparable.

Create an evaluation run:

```http
POST /api/v1/tasks/{taskId}/evaluations
```

Query evaluation runs:

```http
GET /api/v1/tasks/{taskId}/evaluations
```

The release gate can block promotion when any configured threshold fails:

- minimum quality;
- minimum safety;
- minimum reliability;
- maximum P95 latency;
- maximum cost per successful Task;
- maximum human-intervention rate;
- minimum Task success rate.

Raw output is not stored by this baseline API. Its SHA-256 digest is retained so external artifact storage can be verified without placing sensitive model output in the metadata database.

## Capacity fallback

The model runtime consumes a persisted RouteDecision:

```text
Selected candidate
-> ordered fallback chain
```

For each candidate it:

1. checks the Circuit Breaker;
2. resolves the registered Provider for the immutable Model ID;
3. calls the Provider;
4. classifies the result;
5. records Trace evidence;
6. records successful token CostEvents;
7. either returns or advances to the next allowed fallback.

Fallback is allowed for recoverable errors such as:

- Provider unavailable;
- Provider timeout;
- rate limiting;
- another ProviderError explicitly marked retryable;
- context deadline exceeded.

Fallback is not performed for non-retryable errors such as:

- invalid output Schema;
- invalid request configuration;
- deterministic policy or business validation failures;
- missing Provider registration.

This prevents an alternate model from hiding a structural or safety error.

Execute the saved route for a Task:

```http
POST /api/v1/tasks/{taskId}/model
```

The response includes every attempt, the final candidate, whether fallback occurred, latency, error classification, retryability, and Circuit Breaker state.

### Circuit Breaker

The baseline opens a model-version circuit after three consecutive recoverable failures and moves to half-open after a one-minute cooldown. A successful half-open probe closes and resets the circuit.

The current Circuit Breaker store is in memory and is therefore instance-local. Production multi-replica deployment must move this state to Redis or another shared, low-latency coordination store before relying on it for global admission decisions.

## Model admission evidence

A model's `approvalStatus` field is necessary but no longer sufficient for routing. The Router also asks the Model Admission service for an approved evidence record bound to the exact Model ID and Version.

Evidence records are immutable and include:

- evidence ID and digest;
- exact Model ID and Version;
- status;
- license identifier;
- authoritative license text reference;
- model source reference;
- upstream model and dataset references;
- commercial-use, hosted-service, and redistribution permissions;
- attribution or NOTICE requirements;
- artifact digest and signature;
- security-scan reference;
- reviewer and review time.

For commercial or private hosted Endpoints, production admission requires commercial and hosted-service permission.

For self-hosted or local models, production admission additionally requires:

- artifact digest;
- artifact signature;
- security-scan evidence.

A later `revoked` evidence record immediately prevents new routing while historical Tasks remain linked to the exact model version and prior evidence.

Query admission evidence and the current decision:

```http
GET /api/v1/models/{modelId}/admission
```

Append immutable evidence:

```http
POST /api/v1/models/{modelId}/admission
```

The API overwrites Model ID and Version from the stored model resource, so callers cannot attach evidence to a different immutable model version through the request body.

## Provider startup admission

A configured OpenAI-compatible Provider can be marked approved only when the following settings are present:

```text
AICLOUD_PROVIDER_APPROVED=true
AICLOUD_PROVIDER_LICENSE_TEXT_REF=<authoritative reference>
AICLOUD_PROVIDER_EVIDENCE_REVIEWER=<reviewer identity>
```

The model is still subject to health, lifecycle, capability, capacity, quota, budget, residency, and evidence validation during routing.

The deterministic Mock Provider is seeded with internal license, artifact, signature, scan, and reviewer evidence so local development has one complete governed path.

## PostgreSQL tables

Migration `003_trace_evaluation_admission.sql` adds:

- `trace_events`;
- `evaluation_runs`;
- `model_admission_evidence`.

These migrations are applied through the same embedded transactional migration runner as the Model, Task, RouteDecision, and CostEvent tables.

## Current limitations

The v0.1 baseline intentionally leaves several production concerns for follow-up:

1. Circuit Breaker state must move to shared Redis for multi-replica coordination;
2. Provider runtime registration is currently configured at process startup;
3. live capacity, queue, and quota ingestion still requires Provider-specific collectors;
4. fallback execution is synchronous and not yet a Temporal activity;
5. failed Provider attempts without usage data cannot estimate consumed tokens;
6. Trace does not yet export through OpenTelemetry OTLP;
7. evaluation raw artifacts require external object storage;
8. evidence submission requires authentication, RBAC, reviewer separation, and signature verification;
9. artifact signatures and security-scan references are recorded but not yet cryptographically verified by an admission controller;
10. evaluation gates are recorded but release automation does not yet enforce deployment rollback.
