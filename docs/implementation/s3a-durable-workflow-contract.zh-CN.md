# S3A 持久工作流契约

## 1. 目标

在替换 `workflow.NoopEngine` 之前，先冻结执行归属与 Temporal 集成契约。

S3 的目标是让 Task 执行具备可重启、可恢复、可重放能力，但不能因此制造第二套业务事实源。

核心不变量为：

```text
PostgreSQL  = 业务状态 / 查询状态
TaskEvent   = 追加式业务历史
Temporal    = 持久编排 / 执行历史
Outbox      = 事务性投递意图
```

Temporal 负责协调执行，但不拥有 Tenant、Project、Task 授权、Task 状态、成本、审计、审批或 Model Registry 的权威业务状态。

## 2. 非目标

S3A 不做以下事情：

- 不实现 Kubernetes 或基础设施真实副作用；
- 不把 Temporal Workflow 变量作为业务状态的权威记录；
- 不引入第二套 Task 状态机；
- 不允许 Workflow 代码直接调用 Provider SDK、数据库、Tool Gateway Adapter 或网络客户端；
- 不宣称分布式 Exactly-Once；
- 不新增公共 REST API；
- 不修改 R7 已冻结的 Task HTTP 契约。

## 3. 领域模型变更

不引入新的业务聚合根。

`Task` 继续作为 Aggregate Root，现有权威状态保持不变：

```text
CREATED
PLANNING
ROUTING
EXECUTING
WAITING_APPROVAL
VALIDATING
COMPLETED
FAILED
CANCELLED
EXPIRED
```

工作流进度必须通过合法的 Task Command / Transition 与追加式 TaskEvent 表达。Temporal Workflow 本地状态只属于执行状态，不是业务事实。

Workflow ID 由 Task 身份确定性生成，例如：

```text
workflow_id = "task/" + task_id
```

v0.1 中，一个 Task 最多只能存在一个活动的主 Workflow Identity。

## 4. API 变更

S3A 不修改公共 API。

现有 Task Create API 继续作为外部命令入口。API 成功意味着 Task 业务记录以及持久化的 Workflow Start Intent 已经提交，而不是意味着整个 Workflow 已同步执行完成。

后续 S3 子切片只有在 OpenAPI 与 Runtime 同步更新时，才可以公开 Workflow Diagnostics。

## 5. 数据模型变更

S3 复用现有 Task、TaskEvent、Outbox 与 Command Idempotency Kernel。

若需要持久化 Workflow 关联关系，优先采用 Task 上的显式 Metadata，或者独立 Execution Reference，至少包含：

```text
tenant_id
project_id
task_id
workflow_id
workflow_run_id（观测字段，可空）
workflow_type
created_at
updated_at
```

`workflow_run_id` 只属于诊断证据，不属于业务身份。

每次业务状态变更时，Task Projection + TaskEvent + Outbox Delivery Intent 必须保持事务一致。

## 6. 安全边界

Workflow Execution 不能根据 Temporal Metadata 推断授权。

每次 Activity Invocation 必须携带或重新加载足够的不可变身份，以恢复可信执行上下文：

```text
tenant_id
project_id
task_id
trace_id
actor / service identity
```

Activity 必须重新进入受 Tenant/Project Ownership 与 RLS 保护的 Repository / Service 边界。

Temporal Payload 中禁止携带长期原始凭证。后续受治理执行阶段中的 Tool Credential 仍必须经 Credential Broker 获取，并具备 Task 绑定、Purpose 绑定、短时有效等约束。

普通 Workflow Worker 使用普通 Worker DB Role，并继续受 RLS 限制。Temporal 的可用性不能形成数据库绕过路径。

## 7. 运行时流程

S3 目标流程为：

```text
API Command
   -> PostgreSQL Transaction
        Task Projection
        TaskEvent
        Outbox Workflow-Start Intent
        Idempotency Result
   -> Commit

Outbox Dispatcher
   -> workflow.Engine.Start(task identity)
   -> Temporal StartWorkflow

Temporal Workflow
   -> Activity: load Task snapshot
   -> Activity: transition to PLANNING
   -> Activity: planning work
   -> Activity: transition to ROUTING
   -> Activity: route decision
   -> Activity: transition to EXECUTING
   -> Activity: model / policy / approval / tool orchestration
   -> Activity: transition to VALIDATING
   -> Activity: validation
   -> Activity: transition to terminal state
```

所有业务写入只能发生在 Activity 或 Activity 调用的 Application Service 中。Workflow 代码本身必须保持 Deterministic，并且不能直接产生外部副作用。

首个实现子切片应先用确定性的 Fake Activities 验证 Start / Restart / Replay，再接真实 Model 或 Tool 副作用。

## 8. 失败模型

系统必须能够容忍：

- API 在数据库 Commit 后崩溃；
- Outbox Dispatcher 在 Temporal Start 前后崩溃；
- Outbox 重复投递；
- Temporal Client Retry；
- Workflow Worker 重启；
- Activity Timeout；
- Activity Retry；
- Task Version 过期；
- Task 已进入终态；
- Cancellation 与执行并发竞争；
- Temporal 暂时不可用；
- PostgreSQL 暂时不可用。

要求语义：

1. 数据库先 Commit、Workflow 后启动是安全的，因为 Outbox 会重试启动投递。
2. Workflow Start Intent 重复投递是安全的，因为 Workflow ID 对同一个 Task 是确定的；同一 Task 已启动时应按幂等成功处理。
3. Activity Retry 必须安全，业务 Mutation 使用稳定 Idempotency Key 和/或 Task Version OCC。
4. Task 终态不可逆。
5. Workflow 观察到 Task 已终态后应直接退出，不得写入第二次终态 Transition。
6. Temporal 故障只能延迟 Task 执行，不能丢失 Task。

## 9. 幂等模型

需要明确区分三层幂等：

### API Command Idempotency

已由 R6 Command Kernel 为受支持命令提供。

### Workflow Start Idempotency

使用确定性：

```text
workflow_id = task/<task_id>
```

Workflow Start 的 Outbox Delivery Key 必须稳定，例如：

```text
workflow-start:<task_id>
```

同一个 Task 的重复投递不能创建多个并行主 Workflow。

### Activity Idempotency

每个 Mutation Activity 的稳定 Operation Key 必须来自业务身份，而不是 Temporal Attempt Number，例如：

```text
<task_id>:transition:<target_state>:<business_step>
<task_id>:route:<routing_revision>
<task_id>:model:<logical_attempt_id>
<task_id>:tool:<tool_invocation_id>
```

Temporal Retry 可以重复执行 Activity Attempt，但不得重复已提交的业务副作用。

## 10. 可观测性

以下 Correlation Fields 必须贯穿 API、Outbox、Workflow、Activity、Model Attempt 与 Tool Invocation：

```text
request_id
trace_id
tenant_id
project_id
task_id
workflow_id
workflow_run_id
activity_id
model_attempt_id
tool_invocation_id
```

`workflow_run_id` 与 `activity_id` 属于执行证据；`task_id` 与 `trace_id` 仍然是业务与证据的主要关联键。

Workflow Log 不得记录凭证，也不得无界记录原始 Model Payload。

## 11. 成本模型

S3 编排本身不改变 Model / Tool 的价格语义。

所有产生可计费资源消耗的 Retry 都必须可归因，生成携带同一 Task / Trace Identity、并区分 Logical / Physical Attempt 的 Cost Evidence。

Temporal 内部 Retry Count 不能作为计费身份。

## 12. 迁移策略

S3 通过现有 `workflow.Engine` Seam 渐进引入。

推荐上线顺序：

```text
NoopEngine
   -> Deterministic In-Process Test Engine
   -> TemporalEngine（配置开关）
   -> Shadow / Dev 启用
   -> 默认启用 Temporal
```

整个过程中公共 API 契约保持不变。

在 Temporal 启用前创建的历史 Task，除非有明确 Reconciliation / Migration Rule，否则继续保留旧执行模式；S3 不允许自动重放历史副作用。

## 13. 测试

S3 必须增加以下可执行测试：

- Workflow ID 确定性生成；
- 同一 Task 重复 Start 幂等；
- API / DB Commit 后 Workflow 延迟启动；
- Outbox 重复投递；
- Worker Restart + Workflow Resume；
- Activity Retry 不重复 Task Transition；
- Stale Task Version Conflict 后的 Retry / Reload；
- Terminal Task Short-Circuit；
- Cancellation / Restart 行为；
- Temporal Replay Determinism；
- Workflow Code 不直接调用 DB / Provider / Tool Network Client；
- 完整 PostgreSQL Integration Path；
- 中英文文档一致性检查；
- `gofmt`、`go test ./...`、`go vet ./...` 与 Entrypoint Build。

## 14. 验收标准

S3A Contract Gate 只有在以下事项全部冻结后才通过：

1. PostgreSQL 是权威业务 / 查询状态源。
2. TaskEvent 是权威追加式业务历史。
3. Temporal 只拥有 Orchestration / Execution History。
4. Durable Task Command 在 Commit 后只能通过 Outbox 投递 Workflow Start。
5. Workflow ID 对每个 Task 确定。
6. Workflow Code Deterministic 且不直接产生外部副作用。
7. 业务写入经幂等 Activity / Application Service 完成。
8. Worker 必须重建 Tenant/Project Scope，并继续受 RLS 限制。
9. Temporal Retry 不等于业务幂等。
10. Restart / Replay Test 属于 Merge Gate。

## 15. 回滚策略

Temporal Engine 引入期间必须保持可通过配置回滚。

只有在确保同一 Task 不会同时被两个 Engine 执行时，才能将新 Task Start 切回旧 Engine。

已经启动的 Temporal Workflow 必须由对应 Worker Version Drain、通过明确业务规则取消，或继续完成。Rollback 绝不能为同一 Task 启动第二个 Executor。

## 建议的 S3 交付切片

```text
S3A Contract + Engine Boundary
    -> S3B Temporal Client/Worker Adapter + Deterministic Start
    -> S3C Task Lifecycle Workflow + Fake Activities
    -> S3D Durable Activity Idempotency + Cancellation/Retry/Recovery
    -> S3E Model/Policy/Approval Orchestration Integration
    -> S3F Replay/Restart/E2E Proof
```

真实 Tool Side Effect 仍属于 S4 的 Tool Gateway / Credential / Sandbox Governance Boundary。S3 可以编排该 Seam，但不能绕过它。
