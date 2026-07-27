# AI Cloud Model Routing, Security, and Evaluation P0 Priorities

> Status date: 2026-07-27  
> Scope: AI Cloud v0.1 and the first production-oriented Developer AI Cloud workflow  
> Repository state: design documents are on `main`; the runnable skeleton remains in Draft PR #1 and is not yet merged.

## 1. Executive decision

AI Cloud must optimize the complete business task rather than the individual model API call.

The immediate platform objective is:

```text
A governed Task
-> classified by workload type
-> routed to a deterministic, efficient, specialist, or flagship path
-> assigned a model, inference-effort profile, and service tier
-> executed through policy-controlled tools and an isolated sandbox
-> measured by quality, total cost, latency, safety, and reliability
-> recorded in a reproducible trace and evaluation evidence chain
```

The following ten capabilities are P0 implementation priorities:

1. operational Model Registry;
2. joint routing of model, inference effort, and service tier;
3. hybrid commercial API and internal-model deployment;
4. total cost per successful task;
5. Tool Gateway, Policy Engine, and Sandbox before broad Agent autonomy;
6. model-license and provenance evidence admission;
7. Agent-level attack, misuse, and privilege-escalation tests;
8. health-, quota-, capacity-, latency-, and failure-aware routing;
9. specialist routing for security, code, document, and future domain tasks;
10. end-to-end traceability and reproducible evaluation configuration.

These priorities refine, and do not replace, the existing implementation plan and v0.1 engineering milestones.

## 2. Priority architecture

```text
Task API
  |
  v
Task Classifier
  |
  +--> Deterministic path: rules, cache, templates, normal code
  |
  +--> Efficient model path: low-cost commercial, local, or private model
  |
  +--> Specialist path: code, security, document, multimodal, or domain model
  |
  +--> Flagship path: high-complexity or high-value work
  |
  v
Route Decision
  - model and immutable model version
  - inference effort or reasoning budget
  - service tier and latency class
  - deployment mode and region
  - estimated total task cost
  - fallback chain
  - evaluation-evidence version
  - policy, license, and residency decision
  |
  v
Agent Runtime
  |
  v
Tool Gateway -> Policy Engine -> Approval -> Credential Broker
  |
  v
Sandbox or Approved Enterprise Resource
  |
  v
Validation -> Evaluation -> Cost Reconciliation -> Trace and Audit
```

## 3. P0-1 Operational Model Registry

The Model Registry is the source of truth for both static governance metadata and changing runtime state.

### Required immutable model-version fields

- model ID and immutable version ID;
- provider and endpoint type;
- commercial API, dedicated endpoint, private cloud, self-hosted, or local deployment mode;
- model family and upstream base model;
- capability labels;
- specialist task labels;
- context and output limits;
- supported tool, structured-output, multimodal, and reasoning controls;
- supported inference-effort profiles;
- supported service tiers;
- pricing and self-hosted allocation rules;
- region and data-residency attributes;
- license and provenance evidence references;
- artifact digest and signature;
- evaluation-suite and evaluation-evidence references;
- risk, owner, approval, and lifecycle state.

### Required runtime fields

- current health state;
- latency percentiles;
- rolling error and timeout rates;
- rate-limit and quota state;
- available and configured concurrency;
- queue depth and estimated wait time;
- current capacity class;
- circuit-breaker state;
- last successful probe;
- runtime-signal timestamp and freshness limit.

### Lifecycle states

```text
draft
-> evidence-collected
-> reviewed
-> approved
-> active / degraded
-> deprecated / retired / revoked
```

Stale health or capacity records must never be interpreted as current available capacity.

## 4. P0-2 Joint model, inference-effort, and service-tier routing

Routing must select an execution profile, not only a model name.

### Route output

Each `RouteDecision` must include:

- route class: deterministic, efficient, specialist, or flagship;
- selected model version;
- inference profile or reasoning-effort level;
- maximum input, output, and reasoning budget;
- maximum tool-call count;
- maximum execution duration;
- service tier or latency class;
- deployment endpoint and region;
- estimated model and total task cost;
- selected fallback chain;
- selection reason;
- rejected candidates and rejection reasons;
- policy, license, residency, budget, and capacity evidence versions.

### Candidate filtering order

```text
Task capability requirements
-> model-version approval and license admission
-> tenant and data-residency policy
-> endpoint health and signal freshness
-> quota, queue, and available capacity
-> evaluation evidence for the task class
-> predicted total task cost
-> latency and service-level requirement
-> final scoring and selection
```

A cheaper model must not be selected when it cannot meet the task-quality or safety threshold. A flagship model must not be selected by default when a deterministic, efficient, or specialist path meets the acceptance threshold.

## 5. P0-3 Hybrid commercial and internal-model deployment

The same Task and Agent APIs must support:

- public commercial model APIs;
- enterprise or dedicated commercial endpoints;
- cloud-hosted third-party and open-weight models;
- private-cloud endpoints;
- self-hosted open-weight models;
- local small models;
- future domain-specific models.

Provider SDK objects must remain inside adapters. Policy, routing, trace, cost, evaluation, and audit behavior must remain consistent across all deployment modes.

### v0.1 minimum proof

- one commercial provider;
- one private, self-hosted, or local provider;
- the deterministic Mock Provider;
- one Task request executable through at least two model routes without changing the public API.

## 6. P0-4 Cost per successful task

Token usage is only one input to the economic model.

```text
Task total cost
= model input, output, cache, and reasoning cost
+ service-tier premium
+ tool-call cost
+ workflow runtime
+ sandbox compute
+ storage and network
+ retry and failed-attempt cost
+ evaluation cost
+ human approval or review cost where measurable
```

### Required immutable `CostEvent` fields

- tenant, project, user or workload identity;
- Agent, Task, workflow run, and trace IDs;
- component type and component ID;
- provider, model, immutable model version, inference profile, and service tier;
- usage quantity, unit, unit price, currency, and calculated amount;
- estimated or final status;
- success, failure, retry, or cancellation attribution;
- timestamp and source event ID.

### Required metrics

- predicted task cost;
- reconciled task cost;
- cost per successful task;
- failed and retried cost ratio;
- cost by route class;
- cost by model, provider, Agent, tenant, and task class;
- quality-adjusted task cost.

## 7. P0-5 Secure Tool Gateway, Policy, and Sandbox before Agent expansion

No Agent may directly access enterprise systems, credentials, networks, repositories, or execution environments.

Mandatory path:

```text
Agent proposal
-> Tool Gateway validation
-> Policy Engine decision
-> human approval when required
-> task-scoped short-lived credential
-> isolated Sandbox or explicitly approved enterprise resource
-> result filtering
-> audit event
```

### Minimum secure-execution controls

- deny by default;
- versioned Tool Registry and risk classification;
- OPA/Rego policy evaluation;
- task-scoped and short-lived credentials;
- Kubernetes Job-based sandbox;
- namespace and service-account isolation;
- CPU, memory, process, storage, and execution-time limits;
- network deny by default with explicit allow lists;
- read-only or scoped workspace inputs;
- controlled artifact collection;
- deterministic cleanup after success, failure, cancellation, or timeout.

Broad Agent autonomy cannot be marked production-ready before this path is enforced.

## 8. P0-6 License and provenance evidence admission

A model-card label is not sufficient evidence for production use.

### Required evidence

- authoritative license text or durable reference;
- weight license;
- upstream base-model license and version;
- adapter, merge, quantization, and fine-tuning lineage;
- dataset and training provenance disclosures where available;
- commercial-use restrictions;
- redistribution restrictions;
- hosted-service restrictions;
- attribution and notice obligations;
- artifact digest and signature;
- security scan result;
- reviewer, approval state, decision reason, and expiry or review date.

### Admission rule

```text
No evidence
or incompatible restriction
or missing approval
or revoked model version
=> excluded from production routing
```

Historical Tasks must remain linked to the exact model version and evidence state used at execution time.

## 9. P0-7 Agent-level attack and privilege-boundary testing

Agent evaluation must include system-level misuse and attack behavior, not only answer quality.

### Mandatory security test categories

- prompt injection through repository files, issues, documents, web pages, and tool output;
- indirect prompt injection and instruction hierarchy conflicts;
- unauthorized tool discovery or invocation;
- credential exfiltration attempts;
- filesystem traversal and workspace escape;
- network egress outside the allow list;
- privilege escalation and service-account misuse;
- sandbox persistence and cleanup failure;
- cross-tenant cache or trace leakage;
- malicious generated code and dependency behavior;
- policy-bypass attempts;
- excessive retry, resource exhaustion, and denial-of-service behavior;
- manipulation of evaluation or audit output.

### Promotion gate

A model, Agent, prompt, workflow, Tool, or Sandbox version cannot be promoted when a critical security test fails. Waivers must be explicit, time-limited, owned, and auditable.

## 10. P0-8 Health, quota, capacity, latency, and failure-aware fallback

The router must ingest operational state from every external and internal endpoint.

### Required signals

- health and last-success timestamp;
- error and timeout windows;
- latency percentiles;
- quota and rate-limit remaining;
- available concurrency;
- queue depth and expected wait;
- regional state;
- current service tier;
- capacity reservation state;
- signal freshness.

### Failure flow

```text
primary route failure, timeout, overload, or quota exhaustion
-> classify the failure
-> update circuit-breaker state
-> filter the fallback chain by capability, policy, residency, license, budget, and capacity
-> select an allowed route
-> continue under the same Task and Trace
-> record additional latency, retries, and cost
```

When no safe route exists, the platform must queue, reject, or enter a declared degraded mode. It must not silently route to a non-compliant model.

## 11. P0-9 Specialist-model routing

The initial task taxonomy must include at least:

- general reasoning;
- code generation;
- code review;
- security analysis;
- structured extraction;
- document understanding;
- multimodal document processing;
- summarization and classification.

A specialist model may be preferred only when the evaluation evidence applies to the current task class and production configuration. Public benchmark ranking alone is not sufficient.

The router should support a composite pattern:

```text
flagship or specialist planner
-> efficient executor
-> deterministic validator
```

This pattern must remain observable as one Task with separate route and cost events.

## 12. P0-10 End-to-end Trace and reproducible evaluation

### Trace hierarchy

```text
API Request
-> Task
-> Classification
-> Route Decision
-> Workflow Run
-> Agent Run
-> Model Call
-> Policy Decision
-> Tool Call
-> Sandbox Execution
-> Validation
-> Evaluation
-> Cost Reconciliation
```

### Required evaluation evidence

- model and immutable version;
- endpoint, deployment mode, region, inference effort, and service tier;
- system and user prompt versions;
- workflow and Agent versions;
- Tool versions, permissions, and policy version;
- input, output, reasoning, and tool-call budgets;
- timeout, retry, fallback, and context-compaction settings;
- Sandbox image, runtime profile, limits, and network policy;
- dataset and case versions;
- evaluator version and scoring thresholds;
- raw outputs, artifacts, scores, failures, and human interventions.

A production comparison is valid only when the evidence is sufficiently complete to reproduce the configuration and explain material differences.

## 13. Module mapping and current implementation status

Percentages below describe code implementation, not design completeness.

| Capability | Primary modules | Current implementation state | Estimated code progress |
|---|---|---|---:|
| Operational Model Registry | Model Registry, Control Plane | basic type and in-memory repository exist only in Draft PR #1; runtime fields are not implemented | 15% |
| Model + effort + service-tier routing | Model Gateway, Router | planned; provider abstraction foundations exist | 5% |
| Hybrid commercial/internal deployment | Provider adapters, Gateway | architecture defined; no production adapter path on `main` | 10% |
| Cost per successful task | FinOps, Task Runtime | simple Task cost field exists in Draft PR #1; immutable ledger is absent | 5% |
| Tool Gateway, Policy, Sandbox | Tool Gateway, Policy, Sandbox | interfaces and fail-closed seams exist in Draft PR #1; runtime enforcement is absent | 10% |
| License/provenance admission | Model Registry, Supply Chain | design only | 5% |
| Agent attack and privilege tests | Evaluation, Security, Sandbox | not implemented | 0% |
| Health/capacity/fallback | Router, Registry, Observability | design only | 5% |
| Specialist routing | Classifier, Evaluation, Router | taxonomy and design only | 5% |
| Trace and reproducible evaluation | Observability, Evaluation | design and telemetry seam only | 10% |

The overall AI Cloud v0.1 code implementation remains approximately 25%. New roadmap detail does not increase completion percentages until acceptance criteria are verified on `main`.

## 14. Implementation sequence

### Stage A: Stabilize the baseline

- review and merge Draft PR #1;
- add CI and branch protection;
- add PostgreSQL migration runner and repositories;
- freeze Model, Task, RouteDecision, CostEvent, and runtime-status schemas.

**Exit:** a reproducible baseline is available on `main`.

### Stage B: Operational Registry and first hybrid route

- implement Model Registry runtime and evidence fields;
- integrate one commercial and one private/self-hosted route;
- implement inference profile and service-tier schemas;
- record RouteDecision and initial model CostEvents;
- implement health probes and deterministic fallback tests.

**Exit:** one persisted Task can execute through a governed real model route and expose its model, effort, tier, reason, trace, and model-call cost.

### Stage C: Secure execution and security tests

- implement Tool Registry, Tool Gateway, OPA, and credential broker;
- implement Kubernetes Sandbox and default-deny controls;
- add GitHub, filesystem, and restricted-shell tools;
- implement prompt-injection, exfiltration, privilege, network, and escape tests.

**Exit:** the Agent can modify and test code without direct credentials or unrestricted network access.

### Stage D: Durable workflow, full cost, and reproducible evaluation

- integrate Temporal;
- implement durable state transitions, retry, cancel, resume, and approval;
- wire OpenTelemetry;
- emit model, tool, workflow, sandbox, storage, network, evaluation, and retry CostEvents;
- implement the complete evaluation-evidence record.

**Exit:** a long-running Task survives restart and can be reconstructed and economically reconciled from one Trace ID.

### Stage E: Specialist routing, reliable fallback, and supply-chain gate

- implement task classification and specialist capability labels;
- add evaluation-driven specialist routing;
- implement capacity-aware fallback and admission control;
- implement license and provenance evidence admission and revocation;
- complete the GitHub Issue-to-Pull Request workflow.

**Exit:** the Developer AI Cloud MVP selects an explainable, safe, available, compliant, and cost-appropriate execution path.

## 15. Mandatory v0.1 gates

### Gate A: Registry and model independence

No provider-specific SDK type may leak into the public or workflow domain. Routing must use governed Model Registry records.

### Gate B: Route economics

A model-backed route cannot be considered complete until it emits usage, route, retry, and cost events that aggregate into task cost.

### Gate C: Secure execution

Broad Agent functionality cannot be promoted before Tool Gateway, policy, short-lived credentials, and Sandbox boundaries are enforced.

### Gate D: Security evaluation

No Agent workflow can be promoted when critical prompt-injection, credential, privilege, network, sandbox, or cross-tenant tests fail.

### Gate E: Reproducibility

Evaluation-driven routing cannot be enabled until configuration, evidence, raw outputs, and thresholds are versioned and retained.

### Gate F: Reliability

The MVP must pass outage, timeout, overload, quota, stale-signal, and fallback tests without violating policy or silently losing cost and trace data.

### Gate G: Supply-chain trust

Only an approved immutable model version with sufficient license, provenance, artifact, security, and ownership evidence may enter production routing.

## 16. Immediate engineering backlog

1. merge and stabilize Draft PR #1;
2. define `ModelRuntimeStatus`, `InferenceProfile`, and `ServiceTier`;
3. define `RouteDecision`, `RouteCandidate`, and `FallbackPolicy`;
4. define immutable `CostEvent` and cost-reconciliation rules;
5. implement PostgreSQL Model and Task repositories;
6. add one commercial and one private/self-hosted adapter;
7. add health, quota, latency, capacity, queue, and signal-freshness ingestion;
8. implement Tool Gateway, OPA, credential, and Kubernetes Sandbox foundations;
9. define the security attack-test suite and promotion thresholds;
10. define the complete evaluation-evidence schema;
11. define specialist task taxonomy and first code, security, and document suites;
12. implement license/provenance evidence and production-admission checks.

## 17. Related documents

- `AI-Cloud-Implementation-Plan.md`;
- `v0.1-engineering-milestones.md`;
- `AI-Cloud-Module-Implementation-Plan-and-Status.md`;
- `2026-07-24-market-driven-priority-adjustment.md`;
- `../design/Model-Registry-Design.md`;
- `../design/Tool-Gateway-Design.md`;
- `../design/Sandbox-Architecture.md`;
- `../design/Evaluation-Platform-Design.md`.
