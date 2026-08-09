# AI Cloud Engineering Blueprint

## Purpose
This directory is the canonical implementation specification for AI Cloud. ADRs explain why architectural decisions exist; files here define what engineers build, how modules interact, which contracts are stable, and how completion is tested.

If roadmap text, diagrams, or research notes conflict with this directory, implementation MUST follow this directory unless an ADR explicitly supersedes it.

## v0.1 architecture rule
AI Cloud v0.1 is a Go 1.22 modular monolith with separately runnable API and worker processes. Do not split control-plane modules into independent microservices until operational evidence justifies the split. External systems such as PostgreSQL, Redis, Temporal, OPA, object storage, model providers, and Kubernetes remain behind ports/adapters.

```text
Clients / SDK
    |
API Server
    |
Application Services
    |
+---------------- Domain Modules ----------------+
| tenant identity model routing task agent tool  |
| policy workflow sandbox cost audit evaluation  |
+-------------------------------------------------+
    |
Ports / Interfaces
    |
Adapters
    +-- PostgreSQL
    +-- Redis
    +-- Temporal
    +-- OPA
    +-- Object Storage
    +-- Provider APIs / vLLM / SGLang
    +-- Kubernetes
    +-- OpenTelemetry
```

## Repository target structure

```text
cmd/
  aicloud-api/
  aicloud-worker/
api/
  http/
  middleware/
  dto/
  errors/
model/
  domain/
  provider/
  registry/
  router/
agent/
  domain/
  runtime/
workflow/
  domain/
  temporal/
tool/
  domain/
  gateway/
policy/
  engine/
identity/
  domain/
  authn/
  authz/
tenant/
  domain/
sandbox/
  runtime/
cost/
  ledger/
eval/
  runner/
audit/
  ledger/
observability/
  telemetry/
storage/
  postgres/
  redis/
  object/
infra/
  kubernetes/
integrations/
migrations/
deploy/
  helm/
  compose/
docs/
```

Package names may evolve, but dependency direction is mandatory: HTTP/worker adapters -> application/domain -> ports; infrastructure adapters implement ports and MUST NOT own business policy.

## Domain invariants

1. Every tenant-owned object carries TenantID; project-owned objects also carry ProjectID.
2. Every Task has TaskID, TraceID, TenantID, ProjectID, SubjectID and immutable creation metadata.
3. Provider-specific SDK types never cross the provider adapter boundary.
4. Models propose; policy decides; humans approve when required; controllers execute.
5. Tool calls and sandbox executions require policy evaluation before side effects.
6. Every model/tool/sandbox execution emits usage, cost, audit, and trace records.
7. Routing filters policy-ineligible candidates before scoring quality/cost/latency.
8. Retries are bounded and idempotent; fallback may not weaken tenant, license, residency, or security policy.
9. Production model versions are immutable references; upgrades create a new version/admission decision.
10. Missing tenant or authorization context fails closed.

## Stable identifiers
Use UUIDv7-compatible opaque IDs where practical. API clients must treat identifiers as opaque strings. Minimum identifiers:

```text
tenant_id, project_id, subject_id, model_id, model_version_id,
agent_id, workflow_id, task_id, tool_id, sandbox_id,
policy_id, approval_id, trace_id, cost_event_id, audit_event_id
```

## Execution state machine

```text
CREATED
  -> PLANNING
  -> EXECUTING
  -> WAITING_APPROVAL
  -> EXECUTING
  -> VALIDATING
  -> COMPLETED

Any non-terminal state -> FAILED / CANCELLED
```

State transitions are persisted as append-only TaskEvents. The current Task row is a materialized projection for efficient reads.

## Source-of-truth documents

- `component-contracts.md`: module responsibilities and Go interfaces.
- `api-data-contracts.md`: HTTP API, persistence schema, idempotency, errors, and event envelope.
- `runtime-security-flow.md`: end-to-end execution, routing, policy, tools, credentials, sandbox, approvals.
- `deployment-testing.md`: local/Kubernetes topology, HA expectations, observability and required test gates.
- `milestone-v0.1.md`: ordered implementation slices and acceptance criteria.

Each file has a synchronized `.zh-CN.md` version.

## Definition of Done
A feature is complete only when domain behavior, persistence, authorization, telemetry, audit/cost accounting where applicable, unit tests, integration tests, API contract tests, and failure-path tests are implemented. Design-only or branch-only work is not complete on `main`.

## Change discipline
Any incompatible change to resource identity, tenant boundary, provider abstraction, task state machine, policy boundary, audit/cost immutability, or public API requires an ADR or explicit ADR amendment. Implementation docs and both language versions must be updated in the same change set.