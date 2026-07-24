# AI Cloud Market-Driven Priority Adjustment

> Status date: 2026-07-24  
> Scope: AI Cloud v0.1 implementation roadmap  
> Reason: recent commercial and open-model developments show that production competition is shifting from isolated model capability toward routing efficiency, task economics, secure execution, evaluation evidence, supply-chain trust, and reliable hybrid deployment.

## 1. Executive decision

AI Cloud should not optimize for the largest possible list of connected models. It should optimize for safe, explainable, cost-effective and recoverable task execution across interchangeable model providers and deployment modes.

The near-term implementation order is therefore adjusted to:

```text
Model Registry operational metadata
-> provider-neutral routing
-> task-level cost ledger
-> Tool Gateway and Sandbox safety path
-> evaluation evidence chain
-> specialist-model routing
-> capacity-aware fallback
-> license-evidence admission
```

The platform success unit remains a completed business task, not a successful model API call.

## 2. Priority changes

### P0-1 Complete routing and task-level cost accounting together

Model routing and cost accounting must be implemented as one workstream. A router that does not understand total task cost will overuse flagship models; a cost ledger without routing decisions cannot improve execution economics.

The router must support three execution classes:

1. deterministic path: rules, cache, templates or non-model code;
2. efficient model path: low-cost commercial, local or specialist model;
3. flagship path: high-capability model for complex or high-value work.

Required routing inputs:

- task type and required capabilities;
- expected quality threshold;
- enterprise evaluation score;
- model and provider health;
- available quota and capacity;
- predicted latency and queue time;
- predicted task-level cost;
- tenant budget and policy;
- data-residency and license restrictions;
- fallback eligibility.

Required routing output:

```text
selected route
+ selection reason
+ rejected alternatives
+ estimated task cost
+ fallback chain
+ policy decision
+ evidence version
```

The task cost ledger must record immutable events for:

- model input, output and cache use;
- tool invocation;
- workflow runtime;
- sandbox CPU, memory and duration;
- storage and network;
- retries and failed attempts;
- human approval where measurable.

Primary metric:

```text
Cost per Successful Task
```

### P0-2 Expand Model Registry from catalog to operational source of truth

The Model Registry must drive routing, admission, evaluation and audit. It must not remain a static list of model names and endpoints.

Each registered model version must include:

#### Identity and lifecycle

- model ID and immutable version ID;
- provider and endpoint;
- owner and responsible team;
- lifecycle state: draft, active, degraded, deprecated, retired or revoked.

#### Capability

- reasoning, coding, multimodal and tool-use capabilities;
- structured-output support;
- context and output limits;
- supported task domains;
- specialist labels such as security, code, finance or healthcare.

#### Deployment

- commercial API, dedicated enterprise API, private endpoint, self-hosted or local;
- region and data-residency properties;
- runtime engine and quantization where applicable;
- availability zone or cluster placement.

#### Runtime state

- health status;
- current latency and error rate;
- current quota and remaining rate limit;
- available concurrency or queue depth;
- capacity timestamp and signal freshness;
- circuit-breaker state.

#### Economics

- input, output and cache price;
- infrastructure allocation rate for self-hosted models;
- historical cost per successful task by task class.

#### Evaluation

- evaluation suite ID and version;
- dataset version;
- prompt, workflow and tool configuration versions;
- quality, latency, safety, stability and human-intervention scores;
- last evaluation time and promotion decision.

#### Trust and compliance

- license evidence and original text reference;
- upstream base-model identity and license;
- dataset and fine-tuning provenance where available;
- commercial-use and hosted-service restrictions;
- artifact digest and signature;
- security scan and approval status.

### P0-3 Move Tool Gateway and Sandbox before broad Agent expansion

Tool Gateway and Sandbox are prerequisites for production Agent work, not later-stage hardening.

The mandatory execution path is:

```text
Agent proposal
-> Tool Gateway
-> Policy Engine
-> Approval when required
-> short-lived credential
-> Sandbox or approved enterprise resource
-> audited result
```

The first secure execution slice must include:

- Tool Registry with versions, permissions and risk levels;
- OPA/Rego policy decision before sensitive execution;
- GitHub, filesystem and restricted-shell tools;
- task-scoped short-lived credentials;
- Kubernetes Job sandbox;
- isolated namespace and service account;
- CPU, memory and execution-time limits;
- network deny by default;
- controlled workspace inputs and artifact outputs;
- cleanup after completion, failure, cancellation or timeout.

No Agent feature should be considered production-ready if it bypasses this path.

### P0-4 Establish an evaluation configuration evidence chain

Benchmark scores are not reproducible unless the complete test configuration is retained.

Every evaluation run must link the following evidence:

```text
Evaluation Run
+ exact model and version
+ provider or deployment endpoint
+ prompt version
+ workflow version
+ tool package versions
+ tool permissions
+ Token and time budgets
+ retry policy
+ context-compaction settings
+ sandbox profile
+ dataset and case versions
+ evaluator version
+ raw outputs and scores
```

Required behavior:

- comparisons must use the same versioned evaluation configuration;
- public benchmark results are reference data only;
- model, prompt, router or tool changes require regression comparison;
- promotion must be blockable on quality, cost, safety, latency or reliability regression;
- production traces may be sampled into controlled evaluation datasets after privacy and policy checks.

### P1-1 Support specialist-model routing

AI Cloud must model task specialization explicitly rather than assuming one general model is optimal for every workload.

Initial task classes should include:

- general reasoning;
- code generation and code review;
- security analysis;
- structured extraction;
- multimodal document analysis;
- low-cost summarization and classification.

Specialist routing rules must use:

- required capability labels;
- enterprise task evaluation results;
- tool compatibility;
- risk and license status;
- cost and latency limits;
- fallback to a general model when specialist capacity or quality is insufficient.

A specialist model may be preferred only when its evidence is relevant to the current task class.

### P0-5 Implement capacity-aware fallback and graceful degradation

Capacity state is part of model correctness because an unavailable model cannot complete a task.

Required reliability controls:

- active and passive health checks;
- quota and rate-limit tracking;
- latency and error-rate windows;
- bounded retries with jitter;
- circuit breaker;
- ordered and policy-filtered fallback chains;
- admission control before task start;
- degraded-mode behavior;
- deterministic non-model fallback where possible;
- no cross-tenant cache reuse.

Fallback sequence:

```text
primary route failure or overload
-> classify failure
-> open or update circuit breaker
-> filter fallback candidates by capability, policy, residency, license and budget
-> select allowed alternative
-> continue under the same Task and Trace
-> record reason, added latency and added cost
```

The router must reject or queue a task when no safe and compliant capacity is available; it must not silently select a non-compliant model.

### P0-6 Replace license labels with evidence-based admission

A text field such as `Apache-2.0` is not sufficient production evidence.

Production admission must verify:

- license text or authoritative reference;
- model-card and repository evidence;
- upstream base-model licenses;
- commercial-use restrictions;
- redistribution and hosted-service restrictions;
- required attribution and notices;
- artifact digest and signature;
- provenance record;
- security scan;
- reviewer and approval decision.

Recommended lifecycle:

```text
discovered
-> evidence collected
-> legal and security review
-> approved / restricted / rejected
-> active
-> revoked when evidence or policy changes
```

Routing must check the immutable model version approval state at execution time.

## 3. Revised module priority

| Priority | Module | Current state | Immediate objective |
|---|---|---|---|
| P0 | Engineering foundation | Skeleton implemented in Draft PR #1 | merge, add CI and create a stable baseline on `main` |
| P0 | Model Registry | design and partial skeleton | add runtime, economics, evaluation and evidence metadata |
| P0 | Router and provider adapters | provider abstraction partially exists | real provider routes, deterministic path and explainable selection |
| P0 | Task cost ledger | only a simple task cost field exists in Draft PR #1 | immutable component-level cost events |
| P0 | Tool Gateway and Policy | fail-closed seams exist in Draft PR #1 | OPA enforcement and first governed tools |
| P0 | Sandbox | architecture complete, implementation not started | Kubernetes Job isolation with network deny and limits |
| P0 | Evaluation evidence | design exists, runtime not started | versioned evaluation configuration and regression gate |
| P0 | Capacity and fallback | design exists, implementation not started | probes, circuit breaker and compliant fallback |
| P0 | License admission | metadata design is partial | evidence records and routing admission check |
| P1 | Specialist routing | planned | task classes and domain-relevant evaluation rules |
| P1 | Multi-tenant governance | design exists | tenant context, identity, RBAC and budget isolation |

## 4. Revised staged implementation

### Stage 0: stabilize the engineering baseline

Deliver:

- review and merge Draft PR #1;
- CI for format, test, vet, build and Helm lint;
- migration runner;
- stable Model and Task API schemas;
- baseline smoke tests.

Exit criterion: the skeleton is reproducible and verified on `main`.

### Stage 1: operational Model Registry, real routes and cost events

Deliver:

- PostgreSQL Model and Task repositories;
- commercial and mock provider adapters, followed by a private or self-hosted route;
- operational Model Registry fields;
- deterministic, efficient and flagship route classes;
- model-call usage and cost events;
- routing reason and fallback-chain persistence.

Exit criterion: a persisted Task calls a real provider through a provider-neutral protocol and records model, version, route reason, usage, latency and estimated cost.

### Stage 2: secure execution before broad Agent autonomy

Deliver:

- Tool Registry and Tool Gateway;
- OPA policy enforcement;
- task-scoped credential path;
- Kubernetes Job sandbox;
- GitHub, filesystem and restricted-shell tools;
- approval and audit events.

Exit criterion: an Agent can perform one governed code task without direct access to enterprise credentials or infrastructure.

### Stage 3: durable workflow, trace and full task economics

Deliver:

- Temporal workflow;
- complete Task state machine;
- retry, timeout, cancellation and resume;
- OpenTelemetry trace hierarchy;
- tool, workflow and sandbox cost events;
- reconciled task-level cost report.

Exit criterion: a task survives component restart and can be reconstructed with complete trace and cost history.

### Stage 4: evaluation evidence, specialist routing and reliable fallback

Deliver:

- versioned evaluation configuration;
- golden datasets and regression gates;
- task classes and specialist-model capability labels;
- health and capacity probes;
- circuit breaker and policy-compliant fallback;
- cost-, quality- and capacity-aware routing.

Exit criterion: the router can select a specialist, efficient or flagship path using reproducible evidence and can safely continue when the primary route fails.

### Stage 5: supply-chain admission and enterprise boundaries

Deliver:

- license evidence records;
- artifact digest and signature verification;
- model approval and revocation workflow;
- tenant context, OIDC and RBAC;
- tenant-scoped budgets, caches, traces, credentials and artifacts.

Exit criterion: only approved model versions and tools can execute within an isolated tenant boundary.

### Stage 6: Developer AI Cloud end-to-end MVP

Target flow:

```text
GitHub Issue
-> Task classification
-> model or deterministic routing
-> Agent planning
-> Policy and approval
-> Tool Gateway and Sandbox
-> code change and tests
-> evaluation
-> Pull Request
-> trace, route and cost report
```

Exit criterion: the system demonstrates a complete, governed and recoverable Issue-to-Pull Request workflow.

## 5. Additional acceptance questions

Every completed task must answer:

- Was a deterministic, efficient, specialist or flagship route selected?
- Which model version and deployment mode were used?
- Which alternatives were rejected and why?
- Did fallback occur, and what failure triggered it?
- What was the original and final estimated cost?
- What was the reconciled cost per successful task?
- Which evaluation configuration and evidence supported the route?
- Which tools, policies, credentials and sandbox profile were used?
- Was the model license and supply-chain evidence approved?
- Can the full execution be reconstructed from one trace ID?

## 6. Current progress impact

This adjustment does not materially increase the estimated overall v0.1 completion above approximately 25%. It changes sequencing and acceptance criteria:

- routing and task cost move earlier and are developed together;
- Tool Gateway and Sandbox become prerequisites for Agent expansion;
- Model Registry gains runtime and evidence responsibilities;
- evaluation becomes a reproducibility system, not only a score table;
- specialist routing and capacity fallback become first-class platform behavior;
- license approval becomes an execution-time admission requirement.

## 7. Immediate backlog additions

1. Define `RouteDecision`, `RouteCandidate`, `FallbackPolicy` and `CostEvent` schemas.
2. Extend Model Registry schema with operational, evaluation and license-evidence fields.
3. Define evaluation-run evidence schema and immutable configuration digest.
4. Define task classes and initial specialist capability taxonomy.
5. Implement provider health, quota and capacity signal interfaces.
6. Define circuit-breaker states and fallback error taxonomy.
7. Define Tool Gateway and Sandbox minimum production path.
8. Define model admission states and routing-time approval check.
9. Add routing, cost, evaluation and admission decisions to the Task trace schema.
10. Add acceptance tests for provider outage, quota exhaustion, non-compliant model rejection and evaluation regression.