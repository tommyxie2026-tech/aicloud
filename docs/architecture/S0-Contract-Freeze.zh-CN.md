# AI Cloud S0 契约冻结

## 目的

在继续编码之前冻结关键契约，避免架构漂移和后期大规模重构，并定义所有 v0.1 实现必须遵守的平台不变量。

## 核心原则

1. Task 是业务执行聚合根。
2. Tenant 和 Project 是安全边界。
3. Workflow 负责编排，不拥有业务事实。
4. Provider 是 Adapter，不是 Model Catalog。
5. Router 只在 Policy 允许的候选中选择。
6. Policy 决定是否允许，Router 决定如何优化。
7. Tool Gateway 是唯一副作用入口。
8. 所有 Side Effect 必须产生 Evidence。
9. 所有成本必须归属于 Task。
10. 所有生产 ModelVersion 必须具备 Admission 与 Evaluation Evidence。
11. Missing Identity = Unauthenticated，绝不等于 System Access。
12. Task State 与 Canonical TaskEvent 必须原子提交。
13. DB/Event Dual Write 必须使用 Transactional Outbox 消除。
14. 各层 Retry 必须拥有 Stable Idempotency Identity 与 Bounded Attempt Budget。
15. 历史 Evidence 必须不可变并且 Versioned。
16. 运行时演进遵循 `Task -> ExecutionPlan -> model/deployment/tool/subagent graph`；Task 继续作为聚合根，ExecutionPlan 是版本化执行契约。
17. 每个可路由 Graph Node 都必须保持 Model Identity 与 Deployment Identity 分离。
18. 为事后重建所需的 Route-time Policy、Evaluation、Signal 与 Pricing Context 必须固化为 Decision Evidence。
19. Subagent 只能获得 Bounded Delegation，不能获得 Ambient Authority。
20. 第一阶段 Execution Graph 必须是有边界 DAG；无边界 Autonomous Cycle 需要单独冻结契约。

## 运行时责任边界

PostgreSQL：
- Business State；
- Query Projection；
- Resource Ownership；
- Durable Evidence Record。

Workflow Engine：
- Execution Orchestration；
- Retry、Timer、Recovery；
- Durable Wait/Signal。

Task Events：
- Immutable Business History。

Observability：
- Operational Reconstruction 与 Telemetry。

## 已冻结 Contract Pack

S0 契约包目前包括：

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

以上文档均维护同步的 `.zh-CN.md` 中文版本。

## Review Gate

任何 Slice 开始 Coding 前必须回答：

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

平台级 Pre-Code Gate 记录在 `pre-code-architecture-gate.md`，当前 20 项 Contract Category 已全部定义。

## 当前实现决策

S0 Contract Freeze 已完成。Implementation 只能按照以下 Remediation Order 恢复：

```text
R1 Explicit Principal Model
  -> R2 Remove No-scope System Behavior
  -> R3 DB Role / RLS Hardening
  -> R4 Atomic Task Scope Persistence
  -> R5 Task Aggregate / State Transition
  -> R6 TaskEvent + Outbox + Idempotency
  -> R7 OpenAPI / OIDC / RBAC / ABAC Convergence
```

ExecutionPlan / Graph 实现属于 R7 之后的演进，必须继承以上 Frozen Invariant，不能绕过现有安全、证据、成本和身份边界。

PR #12 在 R1-R4 完成前继续保持 Draft。

## 非目标

v0.1 不做：

- 过早 Microservice 拆分；
- 无限制 Autonomous Agent；
- Domain Code 中的 Provider-specific Business Logic；
- Agent 直接访问 Enterprise Resource；
- 把 Workflow History 当作 Business Database；
- 把 Missing Scope 当成 Administrative Privilege；
- 无边界循环式 Agent Execution。

## Change Control

未来任何与 Frozen Invariant 冲突的实现，都必须先创建 Architecture Issue，更新中英文 Contract/ADR，完成 Migration/Compatibility Analysis 并通过 Review 后，才能 Merge 代码。