# S3D PostgreSQL Activity、持久取消与恢复契约

## 1. 目标

用 PostgreSQL/RLS-backed Task Lifecycle Activity 替换 S3C 仅用于测试的业务状态 Seam，并在任何自动 Task Dispatch 被允许启用前，冻结所需的恢复机制。

S3D 继续保持 S3 的事实归属模型：

```text
PostgreSQL  = 权威业务 / 查询状态
TaskEvent   = 追加式业务历史
Outbox      = 事务性投递意图
Temporal    = 持久编排 / 执行历史
```

S3D 增加生产级持久化与恢复 Primitive，但**不会启用生产自动 Outbox -> Temporal Task Execution**。在 S3E 之前，Planning / Routing / Execution / Validation 仍然不是生产业务实现。

## 2. 非目标

S3D 不做以下事情：

- 不让 Temporal 成为 Task 事实源；
- 不因为 Tenant/Project 字段存在于 Temporal Payload 中就直接信任它们作为授权；
- 不给 Worker PostgreSQL `BYPASSRLS`、Superuser、Owner 或任意 Database-admin Capability；
- 不跨租户扫描 Tenant-owned Outbox Payload；
- 不用生产 Stub 替代 Policy、Approval、Router、Model Execution 或 Tool Gateway；
- 不因为 Worker Crash、Temporal Replay Failure 或 Infrastructure Activity 重试耗尽就自动把 Task 标记 `FAILED`；
- 在 Plan/Route/Execute/Validate 仍是 Stub 时不启用生产自动 Dispatch；
- 不在没有同步 API/OpenAPI Contract Review 的情况下暴露公共 Task Cancellation Endpoint；
- 不宣称 Exactly-once Activity 或 Outbox Delivery；
- 不把 Temporal Retry Attempt Number 作为业务幂等身份。

## 3. 安全边界：Temporal 是传输与编排层，不是授权源

S3C Workflow Input 包含 Tenant/Project/Task/Trace 关联身份。S3D 不能仅凭这些字段执行授权。

任何 Activity 在绑定 PostgreSQL Tenant/Project Scope 前，都必须根据 S3 权威身份契约校验当前 Temporal Execution Context。

### 3.1 执行证明

生产 Lifecycle Activity 至少验证：

```text
workflow_type == task-execution-v1
workflow_id   == task/<task_id>
worker namespace == 配置的可信 namespace
activity input task_id == 从 workflow_id 派生的 Task ID
```

Activity Implementation 从 Temporal Activity Context 获取 Workflow Execution 信息，而不是把 Workflow ID/Type 当作可由调用方任意提交的方法参数。

任何不匹配都必须以 Non-retryable Error 失败，且不能执行数据库操作。

### 3.2 Temporal Control Plane 信任

生产激活要求经过身份认证的 Temporal Transport 与 Namespace-level Access Control。未认证的明文 Temporal 连接只允许用于本地开发 / 测试环境。

生产 Activation 至少采用经过部署批准的认证模式之一，例如：

- TLS + mTLS Client Identity；或
- TLS + Temporal 支持的 Namespace/API-key Authentication Mechanism。

只有 AI Cloud Control-plane Starter Identity 可以启动 / 取消 `task-execution-v1`，只有 AI Cloud Worker Identity 可以 Poll 对应 Task Queue。

具体 SDK TLS/Auth 字段在 S3D 实现时基于当前 Temporal Go SDK 再核验。Secret 始终属于 Process Configuration / Secret Reference，不能复制到 Workflow Input / History。

### 3.3 Activity-scoped Workload Principal

Execution Attestation 通过后，Worker 创建显式 Project-scoped Internal Service-account Principal，而不是隐式 SystemPrincipal：

```text
type        = service_account
subject     = aicloud-workflow-worker
tenant_id   = 已证明 tenant_id
project_id  = 已证明 project_id
authn       = internal_workload_identity
issuer      = aicloud
```

之后 Repository Access 继续进入与其他 Scoped Runtime Code 相同的 `identity.RequireProject` 与 PostgreSQL RLS 边界。

Worker DB Role 继续强制 RLS。

## 4. PostgreSQL-backed Activity 边界

S3D 用真实 PostgreSQL Implementation 替换 S3C Runtime `FailClosedLifecycleActivities` 中的 **LoadTask** 与 **TransitionTask** Business-state Method。

Planning / Routing / Execution / Validation 在 S3E 之前仍保持生产 Fail-closed。

### 4.1 LoadTask

`LoadTask` 执行：

```text
Activity execution attestation
 -> 创建 Scoped Service-account Principal
 -> 在 Tenant/Project RLS 下 repository.Get(TaskID)
 -> 校验返回 Task 的 tenant/project/trace 关联
 -> 返回 TaskSnapshot
```

`LoadTask` 只读并且可安全重复。

Cross-tenant、Cross-project、Task/Trace Mismatch 或 Missing Task 必须失败，而且不能泄露其他 Scope 中是否存在该 Task。

### 4.2 TransitionTask

`TransitionTask` 执行：

```text
Activity execution attestation
 -> 创建 Scoped Service-account Principal
 -> 在 RLS 下加载当前 Task
 -> terminal? 返回当前 snapshot
 -> expected_version mismatch? STALE_TASK_VERSION
 -> 校验 Task Aggregate Transition
 -> 构造 TaskEvent
 -> 构造 Activity Idempotency Record
 -> PostgreSQL CommitTransition(
        Task Projection Update,
        TaskEvent Append,
        Optional Outbox Intents,
        Idempotency Completion
    )
 -> 返回已提交 TaskSnapshot
```

业务状态迁移绝不能通过 Transaction Kernel 之外的普通 `UPDATE tasks` 完成。

## 5. Activity Operation 幂等

S3D 复用现有 R6 `idempotency_records` 表，不建立第二套幂等系统。

### 5.1 身份

Transition Activity 使用独立 Operation Namespace：

```text
operation = workflow.activity.task-transition.v1
key       = <task_id>:transition:<target_state>:lifecycle-v1
```

Primary Key 继续按以下 Scope：

```text
tenant_id + project_id + operation + idempotency_key
```

### 5.2 Request Digest

Canonical Request Digest 至少包含：

```text
tenant_id
project_id
task_id
trace_id
expected_version
target_state
cause
operation_key
workflow_type
workflow_id
```

Same Key + Same Digest 返回之前已提交 / 等价的 Result。

Same Key + Different Digest 属于 Invariant Conflict，必须 Non-retryably Fail。

### 5.3 Response Replay

Idempotency Response Payload 保存有界 `TaskSnapshot`，用于处理“DB 已提交但 Activity Response 丢失”的 Retry，且不能追加第二条 TaskEvent。

Idempotency Replay 前仍必须通过 Scope Validation，然后返回已保存 Snapshot。

### 5.4 Retention

当相关 Task 仍为 Nonterminal，或受支持 Workflow Execution 仍可能合法 Retry/Recover 时，Activity Idempotency Record 不能被清理。

初始默认 Retention 至少 30 天并可配置。Cleanup 必须感知 Task / Workflow Retention，而不能仅依据 Wall-clock Expiry 删除。

## 6. Workflow Transition 的 TaskEvent 契约

每次已提交 Lifecycle Transition 必须与 Task Projection Change 在同一 PostgreSQL Transaction 中准确追加一条 Immutable TaskEvent。

推荐 Event Type：

```text
TaskStatusChanged
```

Payload 包含有界业务证据：

```text
from_status
to_status
cause
operation_key
workflow_id
workflow_type
```

已有一级 TaskEvent Column 继续携带：

```text
tenant_id
project_id
task_id
trace_id
sequence
actor principal
occurred_at
schema_version
```

Temporal Activity ID / Attempt 可以作为 Diagnostic Metadata，但 Temporal Attempt Number 不是业务 Idempotency Key。

Replay Transition 不能追加第二条 TaskEvent。

## 7. Error Classification

S3D 冻结 Activity Error Class，避免 Temporal Retry Policy 意外改变业务语义。

### Non-retryable / Workflow-handled

- Invalid Execution Attestation；
- Trusted Scope Binding 后的 Cross-scope / Missing Task；
- Invalid Task Aggregate Transition；
- Idempotency Key/Digest Conflict；
- Malformed Activity Input；
- Unsupported Workflow Type；
- `STALE_TASK_VERSION`：作为 Typed Non-retryable Activity Error，由 Workflow 显式通过 Reload Authoritative Task State 处理。

### Retryable Infrastructure Error

- Transient PostgreSQL Connectivity Failure；
- 能安全重复的 Retryable Transaction / Serialization Failure；
- 尚未产生已提交 Business Effect 的 Temporary Dependency Unavailability。

### Business Failure 与 Orchestration Failure 分离

Infrastructure Retry Exhausted **不能**自动把 Task 转为 `FAILED`。

Task `FAILED` 是业务终态，必须由显式 Business Failure Decision / Command 产生。Worker Crash、Workflow Non-determinism、Temporal Outage 或 Database Outage 仍属于 Orchestration / Recovery Condition。

## 8. 持久业务取消

S3D 在任何公共 Cancellation API 被公开前，先引入内部 Durable Cancellation Command。

强制顺序：

```text
Cancel command
 -> PostgreSQL transaction
      validate Task + expected version
      Task -> CANCELLED
      append TaskStatusChanged / TaskCancelled evidence
      insert workflow.cancel Outbox intent
      complete command idempotency
 -> COMMIT
 -> Outbox delivery
 -> DurableEngine.Cancel(task/<task_id>)
```

Orchestration Cancellation 不能发生在 Business Transaction Commit 之前。

### Cancellation Idempotency

建议 Delivery Identity：

```text
workflow-cancel:<tenant_id>:<task_id>
```

Repeated Cancel Delivery 必须安全。Temporal `NotFound` 属于 Orchestration-level Idempotent Success，因为 PostgreSQL 已拥有 `CANCELLED` 业务事实。

如果 `workflow.cancel` 先于 Pending `workflow.start` 被投递，之后的 Start 仍然安全：PostgreSQL-backed Workflow 加载到已终态 Task 后直接 Short-circuit，不产生第二次业务 Mutation。

### Terminal Race

Completion 与 Cancellation Race 由 OCC 决定。

- Cancellation 先 Commit：后续 Lifecycle Transition 看到 Terminal `CANCELLED` 后停止。
- `COMPLETED` 先 Commit：新的 Cancellation Request 不能重写终态 Task。

在同一个 Change 中完成 API/OpenAPI Contract Review 前，不新增公共 REST Cancellation Route。

## 9. 不绕过 RLS 的 Outbox Dispatch

当前 `ScopedPostgresOutbox` 正确要求显式 Tenant/Project Principal 并应用 RLS。S3D 必须保留这个属性。

因此 Global Dispatcher **不能**通过给 Worker `BYPASSRLS`，或重新引入 Application-controlled System GUC Escape 来解决调度问题。

### 9.1 Global Operational Scope Index

S3D 引入最小化 Global Operational Resource，例如：

```text
outbox_dispatch_scopes
----------------------
tenant_id
project_id
first_seen_at
last_seen_at
PRIMARY KEY (tenant_id, project_id)
```

该 Index：

- 不包含 Task ID；
- 不包含 Outbox ID；
- 不包含 Event Payload；
- 不包含 Model / Tool / User Data；
- 分类为 Global Operational Scheduling Metadata，而不是 Tenant Business Data。

由 Migration-owned Trigger 在已通过 RLS 校验的 `outbox_messages` Insert 时 Transactionally Upsert Scope Row。

Runtime Application Code 没有任意 Direct DML Permission 伪造这个 Index。若使用 `SECURITY DEFINER` Trigger Function，必须固定 `search_path`、撤销 PUBLIC Execution，并由 Migration / Security Test 覆盖。

Worker Scheduler 只获得 Global Scope Index Read-only Access，枚举显式 Tenant/Project Scope 后，再通过 Scoped Workload Principal 与正常 RLS 处理真实 Outbox Row：

```text
Global scope index
 -> (tenant, project)
 -> bind scoped Worker Principal
 -> ScopedPostgresOutbox.Lease
 -> destination adapter
 -> scoped ACK/failure update
```

这是明确的 Two-stage Scheduler。Global Index 永远不返回 Tenant-owned Payload。

### 9.2 Scope-before-delivery Invariant

每个 Outbox Lease / ACK / Failure Operation 都必须重新建立相同 Tenant/Project Scope。某个 Scope 下取得的 Delivery Lease 不能由另一个 Scope ACK。

## 10. Outbox -> Temporal Activation Gate

S3D 可以实现 Dispatcher Wiring，但生产自动 Task Execution 继续默认关闭：

```text
AICLOUD_WORKFLOW_DISPATCH_ENABLED=false
```

S3D Integration Test 可以在 PostgreSQL + Temporal DevServer 环境中打开它。

只有以下条件全部满足，生产 Activation 才被允许：

1. PostgreSQL-backed Load / Transition Activity 通过 RLS / Idempotency Test。
2. Temporal Transport / Namespace Authentication 已配置。
3. Scope Index Scheduling 通过 Cross-tenant Test，且没有 RLS Bypass。
4. Durable Cancellation 与 Duplicate Delivery Test 通过。
5. S3C Replay Compatibility 持续 Green。
6. S3E 提供真实、非 Stub 的 Plan / Route / Execute / Validate Semantic。
7. S3F End-to-end Recovery Proof 通过。

这保证 No-op / Stub Lifecycle 永远不能自动完成 Production Task。

## 11. Workflow Run 恢复策略

Worker Restart 不需要创建新的 Workflow Run；Temporal 会 Replay Open Run。

更困难的情况是 Workflow Execution 自身异常 Closed，但 PostgreSQL 中 Task 仍为 Nonterminal。

S3D 冻结 Engine-neutral 目标语义：

```text
nonterminal Task + active Workflow run     -> resume existing run
nonterminal Task + failed/closed run       -> recovery required; never auto-complete Task
terminal Task + active Workflow run        -> ensure workflow.cancel delivery
terminal Task + closed Workflow run        -> consistent
```

Failed Primary Run 的恢复必须保留确定性 `workflow_id = task/<task_id>`，只有在显式 Review 的 Failed-run Recovery Policy 下才能产生新的 RunID。

实现时需要再次基于当前 Temporal 官方 SDK 核验 `WorkflowIDReusePolicy`。目标语义是：成功完成的 Primary Task Workflow 永远不重复；只有当 PostgreSQL 仍声明 Task Nonterminal，且 Recovery 被显式授权时，失败的 Orchestration Run 才允许 Restart。

现有 S3B `REJECT_DUPLICATE` Policy 不能随意修改；任何修改必须覆盖 Stale Start Delivery、Cancellation-before-start、Completed Task 与 Failed-run Recovery Test。

## 12. Reconciliation

S3D 引入 Reconciliation 作为 Diagnosis / Repair Orchestration，而不是第二事实源。

Reconciler 按显式 Tenant/Project Scope 运行，对比 PostgreSQL / Outbox / Temporal Execution Evidence。

最小 Case：

| PostgreSQL Task | Temporal | 必要行为 |
|---|---|---|
| nonterminal | running/open | healthy |
| terminal | running/open | ensure durable cancel intent |
| nonterminal | missing/failed/closed | recovery anomaly；不能推断 Business Terminal State |
| terminal | missing/closed | healthy 或 cleanup-only |

其他 Outbox Anomaly：

- `workflow.start` dead-letter + nonterminal Task -> Surface / Review Redrive；
- delivered start + missing failed execution + nonterminal Task -> Recovery Review；
- cancel dead-letter + terminal Task + running Workflow -> Redrive / Alert；
- lease expiration -> 继续由现有 Dispatcher Lease Recovery 负责。

### Redrive

如果 S3D 增加 Outbox Redrive，应尽可能复用原 Scoped Outbox Row / Delivery Identity，而不是插入语义重复、可能违反现有 Unique Key 的新 Message。

禁止 Automatic Infinite Dead-letter Redrive。Redrive 必须 Bounded、Auditable、Policy-controlled。

Reconciler 绝不能因为 Temporal 报告 Workflow Completed 就直接把 Task 设置 `COMPLETED`。

## 13. Temporal / Database Failure Matrix

S3D Integration Test 至少覆盖：

- DB Commit Success、Activity Response Lost、Activity Retry -> Idempotency Replay，仅一条 TaskEvent；
- DB Transaction Rollback -> 无 Task Projection Change、无 TaskEvent、无 Idempotency Completion；
- Stale ExpectedVersion -> Typed Stale Error，Workflow Reload；
- Cross-Tenant/Project Activity Input -> 无数据泄露、无 Mutation；
- Forged / Mismatched Workflow ID/Type -> 不绑定 Database Scope；
- Outbox Start 重复投递 -> 单一活动 Primary Workflow Semantic；
- Start Success、ACK Failure -> Lease / Redelivery Safe；
- Cancel Commit、Temporal Unavailable -> Task 保持 CANCELLED，Cancel Outbox Retry；
- Cancel Delivery Earlier Than Start -> 后续 Workflow 加载终态并退出；
- Worker Restart with Open Workflow -> Replay / Resume，不重复 TaskEvent；
- Failed Workflow + Nonterminal Task -> Reconciliation Anomaly，不隐式 Task Failure；
- Dead-letter Delivery -> Bounded Redrive / Alert Semantic；
- 两个 Dispatcher Instance -> SKIP LOCKED / Lease Ownership 防止并发 Delivery Ownership；
- Wrong-scope ACK -> Rejected。

## 14. 可观测性与审计

每个 Activity 与 Dispatcher Operation 按适用情况携带 / 记录：

```text
tenant_id
project_id
task_id
trace_id
workflow_id
workflow_run_id
activity_id
activity_type
operation_key
outbox_id
outbox_attempt
```

Security-sensitive Failure 使用结构化 Reason Code，但不能泄露 Cross-tenant Resource Detail。

Metrics 包括：

```text
workflow_activity_success_total
workflow_activity_retry_total
workflow_activity_idempotency_replay_total
workflow_activity_stale_version_total
outbox_dispatch_pending
outbox_dispatch_dead_letter
outbox_dispatch_retry_total
workflow_reconciliation_anomaly_total
```

Audit / Business History 仍然由 TaskEvent 与 Dedicated Audit Evidence 承担；Temporal Log 不是权威 Audit Ledger。

## 15. Migration Strategy

建议 S3D 交付切片：

```text
D1 Activity execution-attestation + scoped workload identity
 -> D2 PostgreSQL LoadTask
 -> D3 PostgreSQL TransitionTask + TaskEvent + idempotency replay
 -> D4 durable Task cancellation + workflow.cancel adapter
 -> D5 global Outbox dispatch scope index + project-scoped scheduler
 -> D6 Temporal TLS/auth config + dispatcher wiring behind disabled flag
 -> D7 reconciliation/redrive primitives
 -> D8 PostgreSQL + Temporal integration/failure tests
 -> D9 final architecture/security review
```

现有 S3C `FailClosedLifecycleActivities` 在 D1-D3 通过之前仍为默认 Runtime Backend。即使 S3D 进入 Production Build，Automatic Workflow Dispatch 仍然保持关闭，除非后续 S3E / S3F Activation Gate 被显式满足。

## 16. 验收标准

S3D 只有在以下条件全部满足后才通过：

1. Activity Execution Identity 在 Tenant/Project Scope Binding 前被证明。
2. Worker 使用显式 Service-account Workload Identity 与 RLS-enforced DB Role。
3. LoadTask Scoped / Read-only / Repeatable。
4. TransitionTask 使用现有 PostgreSQL Transaction Kernel，而不是 Ad-hoc Task Update。
5. 一次已提交 Transition 准确产生一条 TaskEvent。
6. 相同 Activity Operation 重复执行返回同一 / 等价 TaskSnapshot，不产生第二条 TaskEvent。
7. Same Operation Key + Different Digest 失败。
8. Stale Version 可 Reload，不能覆盖更新后的 Business State。
9. Cancellation 先提交业务状态与 Cancel Outbox，再执行 Temporal Cancellation。
10. Global Outbox Scheduling 只枚举 Non-sensitive Scope Metadata，真实 Message 不绕过 RLS。
11. Wrong-scope Lease / ACK / Activity Access 被拒绝。
12. Worker / Temporal Production Authentication 有显式安全配置路径。
13. S3C Replay Gate 持续 Green。
14. Reconciliation 绝不单凭 Temporal 推导 Task Terminal State。
15. S3E Execution Step 仍不真实时，Automatic Production Dispatch 必须保持 Disabled。
16. Bilingual Docs、Migration、Unit / Integration Test、`go vet` 与 Entrypoint Build 全部 Green。

## 17. 回滚策略

S3D Migration 在 Production Dispatch Activation 前必须 Additive 且 Backward Compatible。

Activation 前 Rollback：

- Disable Workflow Dispatch；
- 保留所有已提交 Task / TaskEvent / Outbox Record；
- Worker 可以退回 Fail-closed Lifecycle Activity；
- Task 不会被静默交给另一个 Executor。

一旦真实 Workflow 已启动，Rollback 必须继续保持 Replay Compatibility 与 Single-executor Ownership。Database Rollback 不能删除解释已提交 Effect 所需的 TaskEvent / Idempotency / Outbox Evidence。

Rollback 绝不能通过直接 Database Edit 伪造 Task State 来“匹配” Temporal History。
