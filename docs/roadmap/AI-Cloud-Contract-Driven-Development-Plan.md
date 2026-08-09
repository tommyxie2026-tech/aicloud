# AI Cloud Contract-Driven Development Plan

> Status date: 2026-08-09  
> Scope: v0.1 production-shaped MVP  
> Rule: architecture documents define intent; implementation contracts define code; tests define completion.

## 1. Delivery Model

AI Cloud v0.1 remains a Go 1.22 modular monolith with two process roles: API and Worker. We will not split services merely to mirror architecture planes.

Every implementation slice must update, when applicable:

1. domain contract;
2. API/OpenAPI contract;
3. persistence/migration contract;
4. runtime/security flow;
5. tests and acceptance evidence;
6. English and Simplified Chinese documentation in the same PR.

A slice is complete only when `gofmt`, `go test ./...`, `go vet ./...`, entrypoint builds and relevant contract tests pass on the PR head.

## 2. Critical Path

```text
S1 Tenant Boundary
 -> S2 API Contract Convergence
 -> S3 Durable Task Workflow
 -> S4 Governed Execution
 -> S5 Trace + Cost + Audit
 -> S6 Reliable Model Routing
 -> S7 Evaluation Release Gates
 -> S8 End-to-End Developer Scenario
```

Provider count is not a milestone. A complete, recoverable and governed Task is the milestone.

## 3. Slice 1 — Tenant Boundary and Task Ownership

### Goal

Make tenant/project context a platform invariant at the external API boundary and prevent cross-tenant Task access before expanding Agent autonomy.

### Implementation

- `internal/tenant`: request scope contract.
- `internal/httpapi`: trusted-ingress tenant middleware.
- `internal/tenantrepo`: scoped Task repository decorator.
- PostgreSQL `task_ownership` table with tenant/project/subject binding.
- PostgreSQL RLS on the ownership table using transaction-local tenant settings.
- Task subresource guard so route, cost, audit, trace, evaluation, model execution and tool execution inherit Task ownership.
- health/readiness endpoints remain outside the tenant gate.

### Acceptance

- API requests without authenticated tenant/subject context fail closed.
- Task APIs require project context.
- Tenant A cannot get or list Tenant B Tasks.
- Cross-tenant Task IDs return not-found rather than authorization detail.
- PostgreSQL ownership reads/writes execute with transaction-local RLS context.
- trusted internal/system context remains available for bootstrap and maintenance paths.

### Explicit boundary

v0.1 uses trusted identity headers supplied by an authenticating ingress. Direct client headers are not a production authentication mechanism. OIDC/JWT verification and RBAC are Slice 2 work.

## 4. Slice 2 — API Contract Convergence

### Goal

Make the running API conform to `docs/implementation/contracts/openapi-v1.yaml` while preserving a short compatibility window.

### Work packages

- OIDC/JWT verifier interface and production implementation.
- RBAC: tenant admin, project admin, developer, operator, reviewer and service account.
- stable `ErrorEnvelope` with request ID, trace ID, code and retryability.
- mandatory `Idempotency-Key` for mutating APIs.
- Task schema convergence: tenant, project, agent, goal/input, status and version.
- Task state machine: CREATED -> PLANNING -> EXECUTING -> WAITING_APPROVAL -> VALIDATING -> terminal state.
- pagination and resource-version semantics.
- OpenAPI contract tests.

### Exit criteria

No public handler accepts an undocumented request shape, and all documented v0.1 paths have executable contract tests.

## 5. Slice 3 — Durable Task Workflow

### Goal

Replace the no-op workflow seam with restart-safe execution.

### Work packages

- Temporal client/worker adapter behind `workflow.Engine`.
- deterministic workflow for planning, routing, model call, policy, approval, tool execution and validation.
- durable retry, timeout, cancellation and resume.
- append-only Task events for every state transition.
- idempotent activities and replay tests.

### Exit criteria

A Task survives API/Worker restart and resumes without duplicate external side effects.

## 6. Slice 4 — Governed Execution

### Goal

Turn the existing Tool Gateway and Sandbox planning foundation into a controlled execution path.

### Work packages

- Kubernetes Job create/watch/collect/destroy executor.
- namespace/service-account isolation.
- network deny by default.
- short-lived task/tool credentials.
- OPA policy adapter in the production path.
- human approval pause/resume through the durable workflow.
- signed workspace input and controlled artifact output.

### Exit criteria

No Agent can reach an enterprise resource except through Tool Gateway -> Policy -> Credential -> Sandbox/Adapter.

## 7. Slice 5 — Trace, Cost and Audit Completion

### Goal

Make every successful and failed Task reconstructable and economically accountable.

### Work packages

- OpenTelemetry SDK and OTLP export.
- Request -> Task -> Workflow -> Agent -> Model/Tool/Sandbox/Evaluation span hierarchy.
- tenant/project dimensions on trace, audit and cost evidence.
- immutable cost events for model, tool, workflow, sandbox, storage/network, retry and human review.
- cost reconciliation and `Cost per Successful Task`.

### Exit criteria

One Task ID or Trace ID reconstructs decisions, actions, failures, retries and total cost without reading application logs manually.

## 8. Slice 6 — Reliable Model Routing

### Goal

Make provider independence operational under failure and load.

### Work packages

- Redis-backed shared circuit-breaker state.
- provider health/quota/capacity collectors.
- latency, queue, residency, budget and evaluation inputs.
- bounded retry and fallback.
- explainable selection/rejection evidence.
- commercial API plus private vLLM/SGLang route under the same internal protocol.

### Exit criteria

Primary-provider failure or quota exhaustion causes a policy-compliant fallback without uncontrolled retries or cross-tenant leakage.

## 9. Slice 7 — Evaluation Release Gates

### Goal

Turn recorded evaluation evidence into enforceable promotion policy.

### Work packages

- versioned golden datasets.
- model/prompt/workflow regression matrix.
- quality, safety, reliability, latency, cost and human-intervention thresholds.
- release gate and rollback decision.
- production trace sampling into evaluation candidates.

### Exit criteria

A model, prompt or workflow version cannot be promoted when required regression gates fail.

## 10. Slice 8 — End-to-End Developer Scenario

Reference scenario:

```text
scale dev-gpu-cluster gpu-workers from 3 to 6
```

Required path:

```text
API
 -> authenticated Tenant/Project
 -> Task persistence
 -> durable Workflow
 -> Router
 -> Provider
 -> structured ChangePlan
 -> Validator
 -> Policy
 -> Human Approval when required
 -> Tool Gateway
 -> short-lived Credential
 -> Kubernetes/Fake Adapter
 -> read-back Validation
 -> COMPLETED
```

Required evidence:

- Task events;
- route decision;
- model attempts;
- policy decision;
- approval record;
- tool invocation;
- audit events;
- cost events;
- OpenTelemetry trace;
- evaluation result.

This scenario is the v0.1 Definition of Done.

## 11. Pull Request Policy

Each slice is developed on a dedicated branch and opened as a Draft PR. Do not commit implementation work directly to `main`. A PR becomes ready only after its acceptance criteria and CI gates pass. English and `.zh-CN.md` documents must remain synchronized in the same PR.
