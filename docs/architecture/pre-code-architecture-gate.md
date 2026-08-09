# AI Cloud v0.1 Pre-Code Architecture Gate

> Status: S0 Contract Freeze Review  
> Decision: Contract gate PASS; S1 implementation merge remains blocked by explicit P0 remediation listed below.

## 1. Gate Purpose

The architecture gate determines whether the v0.1 implementation can proceed without leaving critical domain/security boundaries undefined. A PASS here means the contracts are sufficiently frozen to guide implementation. It does not mean current code already conforms to every contract.

## 2. 20-Point Contract Gate

| # | Gate | Status | Contract Source |
|---|---|---|---|
| 1 | Product Boundary | PASS | `S0-Contract-Freeze.md` |
| 2 | Domain Model | PASS | `task-aggregate-contract.md`, resource scope matrix |
| 3 | Tenant Scope | PASS | `resource-scope-matrix.md`, `identity-contract.md` |
| 4 | Security Boundary | PASS | `security-boundary-model.md`, `database-rls-model.md` |
| 5 | Task State Machine | PASS | `task-aggregate-contract.md` |
| 6 | API Contract | PASS/IMPLEMENT IN S2 | `docs/implementation/contracts/openapi-v1.yaml` |
| 7 | Event Contract | PASS | `task-event-contract.md` |
| 8 | Idempotency | PASS | `idempotency-contract.md` |
| 9 | Workflow Ownership | PASS | `workflow-source-of-truth.md` |
| 10 | Provider Abstraction | PASS | `provider-model-deployment-contract.md` |
| 11 | Model Registry Boundary | PASS | `provider-model-deployment-contract.md` |
| 12 | Policy Boundary | PASS | `policy-routing-boundary.md` |
| 13 | Tool/Side-Effect Boundary | PASS | `security-boundary-model.md` |
| 14 | Audit Contract | PASS | `audit-evidence-contract.md` |
| 15 | Cost Contract | PASS | `cost-accounting-contract.md` |
| 16 | Evaluation Contract | PASS | `evaluation-release-gate-contract.md` |
| 17 | Failure Semantics | PASS | Task, security, policy and workflow contracts |
| 18 | Migration Strategy | PASS/IMPLEMENT PER SLICE | RLS contract + implementation milestone documents |
| 19 | Testing Strategy | PASS | `docs/implementation/deployment-testing*`, per-contract acceptance criteria |
| 20 | E2E Definition of Done | PASS | Contract-driven development roadmap S8 |

Contract score: **20/20 defined**.

## 3. Frozen Platform Invariants

The following are now non-negotiable v0.1 invariants unless changed by a new ADR/review:

1. Task is the core business aggregate.
2. Tenant/Project are explicit immutable security scopes on business resources.
3. Missing identity is unauthenticated, never System.
4. System Principal and administrative DB access are explicit and separately authorized.
5. PostgreSQL owns business state; TaskEvent owns business history; workflow runtime owns orchestration history.
6. Task state transitions produce events atomically.
7. External delivery uses an Outbox rather than unsafe DB/event dual writes.
8. Provider, Model, ModelVersion and Deployment are separate resources.
9. Policy hard constraints run before Router optimization.
10. Agent cannot directly access credentials or enterprise resources.
11. Tool Gateway is the only controlled side-effect boundary.
12. Retry layers have explicit idempotency identities and bounded attempt budgets.
13. Audit and Cost evidence are immutable and Task-attributable.
14. Admission and Evaluation are separate gates.
15. Historical pricing/evaluation/audit evidence is never rewritten to match current configuration.

## 4. S1 Implementation Merge Blockers

PR #12 remains Draft until these P0 issues are remediated against the frozen contracts.

### P0-A: implicit system access

Current prototype behavior allows unscoped context for trusted internal work. This must be replaced by explicit `PrincipalTypeSystem` + capability checks. Missing principal must fail closed.

### P0-B: RLS session-variable privilege bypass

The prototype `aicloud.system_access=on` escape hatch must not be the production privilege model. Runtime app/worker roles must remain RLS-enforced; administrative access uses separate DB credentials/roles.

### P0-C: Task ownership atomicity and long-term identity

Current Task creation followed by separate `task_ownership` binding can create an orphan Task if ownership binding fails. Before merge, either:

1. make Task + ownership binding one PostgreSQL transaction as an explicit compatibility bridge; or
2. preferably migrate `tenant_id`, `project_id`, `created_by` into `tasks` and make Task creation + initial event atomic.

The side table is not the long-term Task identity source.

## 5. S1/S2 Required Remediation Order

```text
R1 Explicit Principal model
  -> R2 remove no-scope system behavior
  -> R3 DB role/RLS hardening
  -> R4 atomic Task scope persistence
  -> R5 Task aggregate/state transition implementation
  -> R6 TaskEvent + Outbox + Idempotency primitives
  -> R7 OpenAPI/OIDC/RBAC/ABAC convergence
```

R1-R4 are required before PR #12 is merge-ready. R5-R7 may be delivered as the first S2 implementation PR(s) if repository compatibility requires staged migration, but their contracts are already frozen.

## 6. Slice Review Template

Before coding any future slice, its design must answer:

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

The English and `.zh-CN.md` versions are reviewed in the same PR.

## 7. Change Control

A code change that contradicts a frozen invariant must not be merged by merely changing tests. It requires:

```text
Architecture issue
  -> contract/ADR update
  -> bilingual review
  -> migration/compatibility analysis
  -> implementation
```

## 8. Decision

S0 Architecture & Contract Freeze is complete enough to resume implementation **only in the remediation order above**. The next coding objective is not a new feature; it is making S1 conform to the frozen identity, RLS and Task ownership contracts.