# Execution Control Plane 收敛分析（S1 → ECP）

[English](execution-control-plane-convergence.md) | **简体中文**

> 状态：DRAFT / Architecture Gate Input  
> 日期：2026-09-16  
> 目标：在不破坏 S0 Frozen Contracts 的前提下，将现有 AI Cloud 收敛为 Task-centric AI Execution Control Plane。

## 1. 结论

本轮代码级 Gap Analysis 的核心结论不是“需要重新设计一套 Execution OS”，而是：**仓库已经具备 Execution Control Plane 的大部分关键骨架，但这些能力分散在 Task、execution、workflow、router、policy/evaluation 和 repository 层，缺少一条明确的 Task → Execution 纵向主线。**

因此后续应以 **Convergence（收敛）** 为主，而不是 parallel rewrite（平行重写）。

冻结原则保持不变：

```text
Task = 业务真相 / business execution aggregate
TaskEvent = 不可变业务历史
Temporal Workflow = durable orchestration
execution/ = 执行域与执行安全语义
Router = policy-approved candidates 中的优化选择
Policy = permission authority
PostgreSQL = durable business/execution evidence
Evaluation = 结果质量与发布/任务 Gate
```

## 2. 已存在的关键能力

### 2.1 Task Domain：KEEP

`internal/domain.Task` 已经拥有规范状态机：

```text
CREATED
 → PLANNING
 → ROUTING
 → EXECUTING
 → WAITING_APPROVAL (conditional)
 → VALIDATING
 → COMPLETED
```

并拥有 FAILED / CANCELLED / EXPIRED 终态、版本号、RouteDecision、Cost、Trace 等字段。

决定：**不创建第二套 TaskDefinition/TaskPhase。** 未来结构化 Goal、constraints、acceptance criteria 必须以兼容方式扩展现有 Task contract，或转换成 execution Goal snapshot，不能形成第二个 Task aggregate。

### 2.2 TaskEvent / Outbox / Idempotency：KEEP

现有 `task_events` 已提供：

- append-only business history；
- tenant/project scope；
- monotonic sequence；
- transactional outbox；
- durable command idempotency；
- PostgreSQL RLS。

决定：**Execution 不再创建独立 event store。** ExecutionPlan/Execution/Attempt 的关键生命周期事件通过现有 `domain.TaskEvent` 写入，payload 携带 `executionId / planId / planRevision / nodeId / attemptId`。

### 2.3 Temporal Workflow：KEEP + EXTEND

现有 Task lifecycle 已经实现：

```text
LoadTask
 → PlanStub
 → RouteStub
 → ExecuteStub
 → ValidateStub
```

并使用 Task version + durable transition 防止 stale mutation。

决定：**Temporal 是 durable orchestration substrate，不再开发第二套 workflow engine。** ECP 的主要工作是逐步用真实 activity 替换 Stub。

### 2.4 `execution/`：KEEP，升级为 canonical Execution Domain

现有 `execution/` 已具备：

- Goal；
- versioned ExecutionPlan；
- bounded DAG validation；
- Execution / ExecutionPhase；
- Node / Target / CapabilityRequirement；
- PolicyDecisionRecord；
- EffectClass；
- Budget；
- Approval；
- Attempt；
- PersistentWorker；
- lease + fence；
- mutation effect uncertainty handling。

决定：**`execution/` 是唯一 canonical Execution Domain。** 不建立 `internal/taskexec` 等平行包。

### 2.5 Router：KEEP + EXTEND

现有 Router 已支持纯 Plan 边界、Deployment/ModelVersion/Pricing evidence 和 RouteDecision 原子提交。

决定：短期保留“Model/Deployment Route”能力，逐步演进为 Node Target Resolver，而不是创建第二个 Intelligent Router。

### 2.6 Evaluation：KEEP + EXTEND

现有 Evaluation 已包含：

```text
TaskSuccessRate
CostPerSuccessfulTask
HumanInterventionRate
Quality / Safety / Reliability
P95 Latency
threshold gate
```

这已经与 ECP 目标高度一致。

决定：将 evaluation 从 release/model gate 延伸到 execution outcome gate；不另造 Evaluator Framework。

## 3. 当前真正的 Gap

### G1：Task 与 Execution 缺少正式绑定

当前 Task 是业务聚合，`execution.Execution` 是执行聚合，但二者之间缺少冻结契约。

建议关系：

```text
Task 1 ────── N Execution
               │
               ├── 1 Goal Snapshot
               ├── 1 active PlanRevision
               └── N Node Attempts
```

一个 Task 可以因为 replan / recovery / alternate strategy 产生多个 Execution；每个 Execution 必须永久保留 `taskRef`。

**建议新增：**

```go
type Execution struct {
    ID      ExecutionID
    TaskRef string
    GoalRef GoalID
    ...
}
```

Goal 是 Task intent 的不可变 execution snapshot，而不是新的业务 Task。

### G2：ExecutionPlan / Execution 缺少一级持久化

数据库已有：

```text
execution_node_runtime
execution_attempts
```

但缺少 canonical：

```text
execution_goals
execution_plans
executions
```

因此 runtime row 当前可以持有 `execution_id / plan_revision`，但 control-plane 还没有完整的 durable parent records。

建议新 migration 使用下一个空闲序号（当前为 019），遵循 PostgreSQL + JSONB + TIMESTAMPTZ + tenant/project RLS；不得重复已有 migration number。

### G3：Workflow Stub 尚未接入 Execution Domain

目标映射：

```text
PlanStub
  → BuildGoalSnapshot
  → Build/Validate ExecutionPlan
  → persist immutable plan revision

RouteStub
  → resolve Node target candidates
  → Policy check
  → Router optimize
  → freeze ExecutionTarget snapshot

ExecuteStub
  → materialize node runtime rows
  → PersistentWorker
  → lease/fence
  → ExecutionAttempt

ValidateStub
  → aggregate outcome
  → evaluation gate
  → Task COMPLETED / FAILED / REPLAN
```

### G4：Agent Runtime 边界重复/过薄

`internal/agentruntime` 目前只是 Skeleton `Run(context.Context,string)`，而 `execution.TargetInvoker` + PersistentWorker 已经定义更完整的执行边界。

决定：`internal/agentruntime` 不再发展成第二套 runtime contract。两种可选收敛方式：

1. 将其作为 `TargetInvoker` 的 Agent adapter facade；或
2. 在确认无外部兼容需求后逐步 deprecated/remove。

默认采用 1，避免破坏现有引用。

### G5：Policy 能力存在，但缺 Execution Authorization 聚合入口

现有 authorization/admission/approval 都应保留。新增的是一个 orchestration facade，而不是重写 policy engine：

```text
ExecutionPolicyEvaluator
  ├── identity / scope
  ├── admission
  ├── RBAC / ABAC
  ├── data sensitivity
  ├── tool/action effect
  ├── budget
  └── human approval
```

输出统一 `execution.PolicyDecisionRecord`。

### G6：API 需要对齐 canonical domain

现有 API prefix 是 `/api/v1`。ECP 扩展应保持兼容：

```text
POST /api/v1/tasks
GET  /api/v1/tasks/{id}

POST /api/v1/tasks/{id}/plan
GET  /api/v1/tasks/{id}/plans
POST /api/v1/tasks/{id}/executions
GET  /api/v1/tasks/{id}/executions
GET  /api/v1/executions/{execution_id}
POST /api/v1/executions/{execution_id}/cancel
```

外部 API 使用 Task / Execution；Node Attempt 默认作为 observability/evidence 资源，不应成为普通业务调用方必须理解的一级概念。

## 4. Package Ownership 收敛

| Package | Decision | ECP 责任 |
|---|---|---|
| `internal/domain` | KEEP | Task / TaskEvent / business truth |
| `execution/` | KEEP + EXTEND | Goal / Plan / Execution / Node / Attempt / effect safety |
| `internal/controlplane` | KEEP + EXTEND | application command + transaction coordination |
| `internal/workflow` | KEEP + EXTEND | Temporal durable orchestration |
| `internal/router` | KEEP + EXTEND | target selection / optimization |
| `internal/admission` | KEEP | admission evidence |
| `internal/authorization` | KEEP | identity/RBAC/scope authorization |
| `internal/approval` | KEEP + EXTEND | human approval persistence |
| `internal/evaluation` | KEEP + EXTEND | execution outcome gate |
| `internal/repository` | KEEP + EXTEND | PostgreSQL/RLS/event/outbox/execution persistence |
| `internal/agentruntime` | REFACTOR | Agent TargetInvoker adapter |
| `agent/*` | KEEP | concrete agent/application logic |
| `cmd/worker` | KEEP + EXTEND | Temporal + execution worker bootstrap |
| `internal/taskexec` | REMOVE | duplicate domain; do not reintroduce |

## 5. Task 与 Execution 双层状态模型

不要合并两个状态机。

**TaskStatus** 表示用户/业务看到的生命周期；**ExecutionPhase** 表示内部一次执行的生命周期。

推荐投影：

```text
Task PLANNING
    ↔ Goal snapshot + Plan build

Task ROUTING
    ↔ target resolution + policy decision

Task EXECUTING
    ↔ Execution READY/RUNNING/REPLANNING

Task WAITING_APPROVAL
    ↔ ExecutionCondition ApprovalRequired=true

Task VALIDATING
    ↔ Execution VERIFYING + Evaluation

Task COMPLETED
    ↔ accepted outcome + evaluation gate passed
```

Execution 失败不必立刻等价于 Task FAILED；如果预算/策略允许，Control Plane 可以产生新的 PlanRevision 或新的 Execution。

## 6. Source of Truth

```text
Business state          → tasks
Business history        → task_events
Execution definition    → execution_goals / execution_plans / executions
Node mutable authority  → execution_node_runtime
Attempt evidence        → execution_attempts
Orchestration history   → Temporal
Operational telemetry   → trace / metrics / logs
Evaluation evidence     → evaluation_runs
```

严禁用 Temporal history 替代 PostgreSQL business truth，也严禁用 trace/log 代替审计证据。

## 7. ECP Convergence Roadmap

为避免与仓库已有 R1–R7 remediation 编号冲突，新路线统一使用 `ECP-Cx`。

### ECP-C1 — Task ↔ Execution Binding

- 在 `execution.Execution` 增加 immutable `TaskRef`；
- 定义 Goal snapshot conversion；
- 新增 execution goals/plans/executions durable schema；
- repository interface + PostgreSQL implementation；
- TaskEvent 增加 plan/execution lifecycle payload conventions。

**验收：** 给定一个 Task，可以稳定查询其 Goal、所有 PlanRevision、所有 Execution 和 Attempt lineage。

### ECP-C2 — Replace Plan/Route Stubs

- `PlanStub` → real ExecutionPlan builder + validation；
- `RouteStub` → existing Router + PolicyDecisionRecord；
- 将 Model Route 扩展为 Node Target resolution。

**验收：** Temporal workflow 不再以 stub 假装 planning/routing 成功。

### ECP-C3 — Execute through PersistentWorker

- `ExecuteStub` → materialize node runtime；
- ready-node scheduler；
- persistent worker + target invoker；
- Agent/model/tool adapters；
- cancellation/timeout/retry/effect recovery。

**验收：** 至少 Model + Tool 两类节点可以 durable execute/recover。

### ECP-C4 — Policy / Approval / Budget Convergence

-统一 execution policy facade；
- high-risk mutation approval；
- budget reservation durable backend；
- credential/tool permission binding。

### ECP-C5 — Evaluation / Outcome

- `ValidateStub` → task-level evaluation；
- result artifact + evidence；
- Task Success Rate / Cost per Successful Task；
- failure → replan / alternate execution 策略。

### ECP-C6 — Organization Memory

- 从成功 Execution trajectory 生成 Procedure Candidate；
- evaluation + policy gate 后进入 Organization Memory；
- Router/Planner 可以读取，但不能自主修改安全策略。

## 8. 当前已执行的纠偏

本轮 Gap Analysis 已执行以下 cleanup：

1. 删除新建但与 `execution/` 重复的 `internal/taskexec`；
2. 删除错误重复编号且与现有 TaskEvent/Execution Attempt 体系冲突的 `010_task_execution_contract.sql`；
3. 保留并修正 Task Execution OpenAPI 草案，使其使用既有 `/api/v1` namespace；
4. 明确后续数据库变更从 migration `019` 开始；
5. 明确 canonical event stream 继续使用 `task_events`。

## 9. ECP-C1 Architecture Gate

进入下一批代码前必须满足：

- Goal：Task 可产生 durable Execution lineage；
- Non-goal：不替换 Task aggregate / Temporal / Router；
- Domain：只增加 TaskRef/binding，不引入第二 Task；
- API：向后兼容 `/api/v1/tasks`；
- Data：019 migration + RLS；
- Security：tenant/project scope 全链路保持；
- Runtime：Temporal → execution worker；
- Failure：replan 与 retry 分离；
- Idempotency：reuse existing command/outbox model；
- Observability：task/trace/execution/plan/node/attempt IDs 可关联；
- Cost：所有 Execution cost 最终归属 Task；
- Migration：不复用既有编号；
- Tests：domain + migration contract + PostgreSQL integration + Temporal replay；
- Acceptance：完整 lineage 可重建；
- Rollback：019 只新增，不破坏旧 Task API。

## 10. 下一实现切片

下一步不再继续抽象讨论，直接实现 **ECP-C1**：

```text
execution.Execution.TaskRef
        +
019_execution_control_plane.sql
        +
ExecutionStore interface
        +
PostgresExecutionStore
        +
Task -> GoalSnapshot adapter
        +
contract/integration tests
```

完成后再替换 Temporal `PlanStub`，避免一次改动 Task、Workflow、Router、Runtime 四个层面而失去可验证边界。
