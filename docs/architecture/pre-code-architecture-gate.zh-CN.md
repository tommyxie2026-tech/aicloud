# AI Cloud v0.1 Pre-Code Architecture Gate

> 状态：S0 Contract Freeze Review  
> 决策：Contract Gate 通过；S1 当前实现仍需完成下述 P0 Remediation 后才能 Merge。

## 1. Gate 目标

Architecture Gate 用于判断 v0.1 是否已经把关键 Domain/Security Boundary 冻结到足以指导实现。这里的 PASS 代表“契约已经定义清楚”，不代表当前代码已经全部符合契约。

## 2. 20 项 Contract Gate

| # | Gate | 状态 | Contract Source |
|---|---|---|---|
| 1 | Product Boundary | PASS | `S0-Contract-Freeze.md` |
| 2 | Domain Model | PASS | `task-aggregate-contract.md`、Resource Scope Matrix |
| 3 | Tenant Scope | PASS | `resource-scope-matrix.md`、`identity-contract.md` |
| 4 | Security Boundary | PASS | `security-boundary-model.md`、`database-rls-model.md` |
| 5 | Task State Machine | PASS | `task-aggregate-contract.md` |
| 6 | API Contract | PASS / S2 实现 | `docs/implementation/contracts/openapi-v1.yaml` |
| 7 | Event Contract | PASS | `task-event-contract.md` |
| 8 | Idempotency | PASS | `idempotency-contract.md` |
| 9 | Workflow Ownership | PASS | `workflow-source-of-truth.md` |
| 10 | Provider Abstraction | PASS | `provider-model-deployment-contract.md` |
| 11 | Model Registry Boundary | PASS | `provider-model-deployment-contract.md` |
| 12 | Policy Boundary | PASS | `policy-routing-boundary.md` |
| 13 | Tool / Side Effect Boundary | PASS | `security-boundary-model.md` |
| 14 | Audit Contract | PASS | `audit-evidence-contract.md` |
| 15 | Cost Contract | PASS | `cost-accounting-contract.md` |
| 16 | Evaluation Contract | PASS | `evaluation-release-gate-contract.md` |
| 17 | Failure Semantics | PASS | Task / Security / Policy / Workflow Contract |
| 18 | Migration Strategy | PASS / 每 Slice 实现 | RLS Contract + Implementation Milestone |
| 19 | Testing Strategy | PASS | `docs/implementation/deployment-testing*` + 各 Contract Acceptance Criteria |
| 20 | E2E Definition of Done | PASS | Contract-driven Roadmap S8 |

Contract Score：**20/20 已定义**。

## 3. 已冻结的平台不变量

除非通过新的 ADR/Architecture Review 修改，以下原则作为 v0.1 Non-negotiable Invariant：

1. Task 是核心 Business Aggregate；
2. Tenant/Project 是业务资源显式且不可变的 Security Scope；
3. Missing Identity = Unauthenticated，绝不等于 System；
4. System Principal 与 Admin DB Access 必须显式、独立授权；
5. PostgreSQL 管 Business State，TaskEvent 管 Business History，Workflow Runtime 管 Orchestration History；
6. Task State Transition 与对应 Event 原子提交；
7. 外部 Delivery 使用 Outbox，禁止不安全 DB/Event Dual Write；
8. Provider、Model、ModelVersion、Deployment 是不同 Resource；
9. Policy Hard Constraint 永远先于 Router Optimization；
10. Agent 不直接访问 Credential 或 Enterprise Resource；
11. Tool Gateway 是唯一 Controlled Side-effect Boundary；
12. 每层 Retry 都必须有明确 Idempotency Identity 与 Bounded Attempt Budget；
13. Audit 与 Cost Evidence 不可变且可归属于 Task；
14. Admission 与 Evaluation 是不同 Gate；
15. 历史 Pricing/Evaluation/Audit Evidence 不因当前配置变化而被改写。

## 4. S1 Implementation Merge Blocker

PR #12 继续保持 Draft，直到以下 P0 问题按冻结契约修复。

### P0-A：Implicit System Access

当前 Prototype 允许 Unscoped Context 作为 Trusted Internal Work。必须替换为显式 `PrincipalTypeSystem` + Capability Check。Missing Principal 一律 Fail Closed。

### P0-B：RLS Session Variable Privilege Bypass

Prototype 中的 `aicloud.system_access=on` Escape Hatch 不能作为生产 Privilege Model。Runtime App/Worker Role 必须始终 RLS-enforced；Admin Access 使用独立 DB Credential/Role。

### P0-C：Task Ownership Atomicity 与长期 Identity

当前流程先 Create Task，再独立 Bind `task_ownership`，Ownership Binding 失败时可能产生 Orphan Task。

Merge 前至少需要：

1. 将 Task + Ownership Binding 放入同一个 PostgreSQL Transaction，作为 Compatibility Bridge；或者
2. 更推荐直接把 `tenant_id`、`project_id`、`created_by` 迁移进 `tasks`，并让 Task Creation + Initial Event 原子提交。

`task_ownership` Side Table 不作为长期 Task Identity Source of Truth。

## 5. S1/S2 Remediation Order

```text
R1 Explicit Principal Model
  -> R2 Remove No-scope System Behavior
  -> R3 DB Role / RLS Hardening
  -> R4 Atomic Task Scope Persistence
  -> R5 Task Aggregate / State Transition
  -> R6 TaskEvent + Outbox + Idempotency Primitive
  -> R7 OpenAPI / OIDC / RBAC / ABAC Convergence
```

R1-R4 必须在 PR #12 Ready for Merge 前完成。R5-R7 如果为了兼容迁移需要分阶段，可以作为最前面的 S2 PR，但其 Contract 已经冻结。

## 6. 后续 Slice 的统一 Review Template

任何 Future Slice 开始 Coding 前必须回答：

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

英文与 `.zh-CN.md` 必须在同一个 PR 中同步 Review。

## 7. Change Control

如果 Code Change 与 Frozen Invariant 冲突，不能只通过改测试“让它绿”。必须走：

```text
Architecture Issue
  -> Contract/ADR Update
  -> Bilingual Review
  -> Migration/Compatibility Analysis
  -> Implementation
```

## 8. 决策

S0 Architecture & Contract Freeze 已经足够完整，可以恢复 Implementation，但**只能按照 Remediation Order 推进**。

下一步 Coding Objective 不是增加新 Feature，而是先让 S1 实现符合已经冻结的 Identity、RLS、Task Ownership Contract。