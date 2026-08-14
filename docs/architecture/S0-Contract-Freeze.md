# AI Cloud S0 Contract Freeze

## Purpose

Before additional implementation, freeze the contracts that prevent architectural drift and define the invariants that all v0.1 code must satisfy.

## Core Principles

1. Task is the business execution aggregate.
2. Tenant and Project are security boundaries.
3. Workflow orchestrates execution but does not own business truth.
4. Provider is an adapter, not the model catalog.
5. Router selects among policy-approved candidates.
6. Policy decides permission; Router decides optimization.
7. Tool Gateway is the only side-effect boundary.
8. Every side effect requires evidence.
9. Every cost belongs to a Task.
10. Every production model requires admission and evaluation evidence.
11. Missing identity is unauthenticated, never System access.
12. Task state and canonical TaskEvent commit atomically.
13. Unsafe DB/event dual writes are replaced by a transactional Outbox.
14. Retry layers require stable idempotency identities and bounded attempt budgets.
15. Historical evidence is immutable and versioned.
16. Runtime evolution follows `Task -> ExecutionPlan -> model/deployment/tool/subagent graph`; Task remains the aggregate root and ExecutionPlan is a versioned execution contract.
17. Model and Deployment identity remain separate at every routable graph node.
18. Route-time policy, evaluation, signal and pricing context required for reconstruction is persisted as decision evidence.
19. Subagents receive bounded delegation rather than ambient authority.
20. Initial execution graphs are bounded DAGs; unbounded autonomous cycles require a separate frozen contract.

## Runtime Ownership

PostgreSQL:
- business state;
- query projection;
- resource ownership;
- durable evidence records.

Workflow Engine:
- execution orchestration;
- retry, timers and recovery;
- durable waits/signals.

Task Events:
- immutable business history.

Observability:
- operational reconstruction and telemetry.

## Frozen Contract Pack

The S0 contract pack is now:

```text
S0-Contract-Freeze.md
resource-scope-matrix.md
identity-contract.md
task-aggregate-contract.md
task-event-contract.md
workflow-source-of-truth.md
security-boundary-model.md
database-rls-model.md
provider-model-deployment-contract.md
idempotency-contract.md
trace-context-contract.md
policy-routing-boundary.md
audit-evidence-contract.md
cost-accounting-contract.md
evaluation-release-gate-contract.md
pre-code-architecture-gate.md
execution-plan-graph-contract.md
```

Every document has a synchronized `.zh-CN.md` version.

## Review Gate

No Slice may start implementation unless its design answers:

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

The platform-wide pre-code gate is documented in `pre-code-architecture-gate.md` and currently has all 20 contract categories defined.

## Current Implementation Decision

S0 contract freeze is complete. Implementation may resume only in this remediation order:

```text
R1 Explicit Principal model
  -> R2 remove no-scope System behavior
  -> R3 DB role / RLS hardening
  -> R4 atomic Task scope persistence
  -> R5 Task aggregate/state transition
  -> R6 TaskEvent + Outbox + Idempotency
  -> R7 OpenAPI / OIDC / RBAC / ABAC convergence
```

ExecutionPlan/graph implementation is a post-R7 evolution and must preserve the frozen invariants above rather than bypass them.

PR #12 remains Draft until R1-R4 are complete.

## Non-Goals

v0.1 does not attempt:

- premature microservice decomposition;
- autonomous unrestricted agents;
- provider-specific business logic in domain code;
- direct Agent access to enterprise resources;
- treating workflow history as the business database;
- treating missing scope as administrative privilege;
- unbounded cyclic agent execution.

## Change Control

A future implementation that conflicts with a frozen invariant requires an architecture issue, bilingual contract/ADR review, migration analysis and explicit approval before code changes are merged.