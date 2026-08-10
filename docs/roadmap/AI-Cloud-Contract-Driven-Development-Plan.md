# AI Cloud Contract-Driven Development Plan

> Status date: 2026-08-09  
> Scope: v0.1 production-shaped MVP  
> Rule: architecture documents define intent; implementation contracts define code; tests define completion.

## 1. Delivery Model

AI Cloud v0.1 remains a Go 1.22 modular monolith with API and Worker process roles. We will not split services merely to mirror architecture planes.

Every implementation slice must update, when applicable:

1. domain contract;
2. API/OpenAPI contract;
3. persistence/migration contract;
4. runtime/security flow;
5. failure/idempotency model;
6. observability/cost evidence;
7. tests and acceptance evidence;
8. English and Simplified Chinese documentation in the same PR.

A slice is complete only when `gofmt`, `go test ./...`, `go vet ./...`, entrypoint builds and relevant contract tests pass on the PR head.

## 2. Critical Path

```text
S0 Architecture & Contract Freeze
 -> S1 Tenant / Identity Boundary
 -> S2 API + Domain Contract Convergence
 -> S3 Durable Task Workflow
 -> S4 Governed Execution
 -> S5 Trace + Cost + Audit
 -> S6 Reliable Model Routing
 -> S7 Evaluation Release Gates
 -> S8 End-to-End Developer Scenario
```

Provider count is not a milestone. A complete, recoverable, governed, auditable and economically accountable Task is the milestone.

## 3. S0 — Architecture & Contract Freeze

### Goal

Remove ambiguity before further coding and establish stable invariants for Domain, Identity, Tenant Scope, Task, Event, Workflow, RLS, Provider/Model separation, Idempotency, Trace, Policy/Router, Audit, Cost and Evaluation.

### Frozen contract pack

See `docs/architecture/S0-Contract-Freeze.md` and `pre-code-architecture-gate.md`.

### Exit criteria

- 20/20 architecture contract categories defined;
- bilingual contract pack complete;
- S1 merge blockers explicitly identified;
- implementation can resume only in the approved remediation order.

### Remediation order after S0

```text
R1 Explicit Principal model
 -> R2 remove no-scope System behavior
 -> R3 DB role / RLS hardening
 -> R4 atomic Task scope persistence
 -> R5 Task aggregate/state transitions
 -> R6 TaskEvent + Outbox + Idempotency
 -> R7 OpenAPI / OIDC / RBAC / ABAC convergence
```

R1-R4 are required before the current S1 PR becomes merge-ready.

## 4. S1 — Tenant / Identity Boundary and Task Ownership

### Goal

Make verified Principal, tenant and project context platform invariants and prevent cross-tenant/project Task access before expanding Agent autonomy.

### Implementation

- explicit `Principal` contract: User, ServiceAccount, System;
- authentication boundary resolves verified Principal before protected APIs;
- scoped Task repository and Task subresource guard;
- Task scope persistence becomes atomic;
- long-term Task schema carries `tenant_id`, `project_id`, `created_by` directly;
- PostgreSQL runtime roles use RLS with transaction-local tenant/project context;
- app/worker roles have no administrative bypass;
- health/readiness endpoints remain outside tenant authorization while still following infrastructure access policy.

### Acceptance

- missing identity fails closed;
- missing tenant never implies System access;
- Project APIs require project context;
- Tenant A cannot access Tenant B Tasks;
- same Tenant/different Project isolation follows scope policy;
- cross-scope Task IDs use not-found semantics where required;
- Task creation + scope ownership is atomic;
- app/worker DB roles cannot bypass RLS;
- System access requires explicit System Principal/capability and separate administrative DB path where needed.

### Migration note

The current trusted-header and `task_ownership` implementation is a compatibility bridge. It must converge toward the frozen Identity/RLS/Task contracts before merge.

## 5. S2 — API + Domain Contract Convergence

### Goal

Make the running API and persistence model conform to the frozen Task/Identity/Event contracts and `docs/implementation/contracts/openapi-v1.yaml`.

### Work packages

- OIDC/JWT verifier interface and production implementation;
- RBAC plus policy/ABAC authorization seam;
- stable `ErrorEnvelope` with request ID, trace ID, code and retryability;
- mandatory `Idempotency-Key` for mutating APIs;
- canonical Task schema: tenant, project, created_by, agent, goal/input, constraints, status and version;
- Task transition API/state machine;
- append-only TaskEvent storage;
- transactional Outbox;
- command idempotency records;
- optimistic concurrency/resource version;
- pagination and executable OpenAPI contract tests.

### Exit criteria

- public handlers accept only documented request shapes;
- every documented v0.1 path has executable contract tests;
- Task state cannot be changed by arbitrary field mutation;
- Task mutation + canonical event commits atomically;
- duplicate mutating requests do not duplicate business operations.

## 6. S3 — Durable Task Workflow

### Goal

Replace the no-op workflow seam with restart-safe orchestration without making the workflow runtime the business database.

### Work packages

- Temporal client/worker adapter behind `workflow.Engine`;
- deterministic planning/routing/model/policy/approval/tool/validation workflow;
- Outbox-driven workflow start/signal integration where required;
- durable retry, timeout, cancellation and resume;
- idempotent activities and replay tests;
- reconciliation between Task projection, TaskEvent and workflow runtime.

### Exit criteria

A Task survives API/Worker restart and resumes without duplicate external side effects; business state remains queryable from PostgreSQL without reading workflow history.

## 7. S4 — Governed Execution

### Goal

Turn Tool Gateway and Sandbox foundations into the only controlled path for external side effects.

### Work packages

- first prove the path with deterministic/fake executor;
- then add Kubernetes Job create/watch/collect/destroy executor;
- namespace/service-account isolation;
- default-deny network policy;
- short-lived task/tool credentials;
- OPA policy adapter;
- human approval pause/resume;
- proposal digest binding to approval;
- signed workspace inputs and controlled artifact output.

### Exit criteria

No Agent can reach an enterprise resource except through Tool Gateway -> Policy -> Approval when required -> Credential Broker -> Sandbox/Adapter.

## 8. S5 — Trace, Cost and Audit Completion

### Goal

Make every successful and failed Task reconstructable and economically accountable.

### Work packages

- OpenTelemetry SDK and OTLP export;
- Request -> Task -> Workflow -> Agent -> Model/Tool/Sandbox/Evaluation span hierarchy;
- tenant/project/task correlation on evidence;
- immutable AuditEvents and CostEvents;
- model/tool/workflow/sandbox/storage/network/retry/human-review cost activities;
- pricing version and reconciliation;
- `Cost per Successful Task`.

### Exit criteria

One Task ID or Trace ID reconstructs decisions, actions, failures, retries, approvals and total cost without manual log correlation.

## 9. S6 — Reliable Model Routing

### Goal

Make provider independence operational under outage, quota exhaustion and load while preserving policy hard constraints.

### Work packages

- Provider / Model / ModelVersion / Deployment runtime mapping;
- Redis-backed shared circuit-breaker state;
- provider health/quota/capacity collectors;
- latency, queue, residency, budget and evaluation inputs;
- hard-policy filters before soft scoring;
- bounded retry and fallback;
- explainable eligible/rejected candidates;
- commercial API plus private vLLM/SGLang under the same internal protocol.

### Exit criteria

Primary deployment failure or quota exhaustion causes a policy-compliant fallback without uncontrolled retries, policy bypass or cross-tenant leakage.

## 10. S7 — Evaluation Release Gates

### Goal

Turn recorded evaluation evidence into enforceable promotion, routing eligibility and rollback policy.

### Work packages

- versioned golden datasets;
- model/prompt/agent/workflow regression matrix;
- L1 offline, L2 pre-production and L3 production evaluation;
- quality, safety, reliability, latency, cost and human-intervention thresholds;
- immutable GateDecision;
- production trace sampling under data-governance rules;
- release gate and rollback decision.

### Exit criteria

A version/configuration cannot be promoted or made route-eligible when required gates fail, and every decision is reproducible from stored evidence.

## 11. S8 — End-to-End Product Proof

Use three governed scenarios:

```text
A. read-only repository/cluster inspection        -> ALLOW
B. scale dev-gpu-cluster gpu-workers 3 -> 6       -> REQUIRE APPROVAL
C. destructive production request                 -> DENY
```

For the mutation path, required execution is:

```text
API
 -> authenticated Principal/Tenant/Project
 -> idempotent Task creation
 -> TaskEvent + Outbox
 -> durable Workflow
 -> Policy-eligible candidate set
 -> Router
 -> Provider/Deployment
 -> structured ChangePlan
 -> Validator
 -> Policy
 -> Human Approval when required
 -> Tool Gateway
 -> short-lived Credential
 -> Sandbox/Kubernetes or Fake Adapter
 -> read-back Validation
 -> COMPLETED
```

Required evidence:

- Task events;
- route decision and model attempts;
- policy decision;
- approval record;
- tool invocation;
- audit events;
- cost events;
- OpenTelemetry trace;
- evaluation result.

This three-path ALLOW / APPROVE / DENY proof is the v0.1 Definition of Done.

## 12. Slice Review Template

Every slice must answer before coding:

```text
1. Goal
2. Non-Goals
3. Domain Changes
4. API Changes
5. Data Model Changes
6. Security Boundary
7. Runtime Flow
8. Failure Model
9. Idempotency Model
10. Observability
11. Cost Model
12. Migration
13. Tests
14. Acceptance Criteria
15. Rollback Strategy
```

## 13. Pull Request Policy

Each slice/remediation unit uses a dedicated branch and Draft PR. Do not commit implementation work directly to `main`. A PR becomes ready only after contract review, acceptance criteria and CI gates pass. English and `.zh-CN.md` documents remain synchronized in the same PR.