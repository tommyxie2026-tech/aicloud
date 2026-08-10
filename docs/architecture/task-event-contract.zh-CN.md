# AI Cloud Task Event 与 Outbox 契约

> 状态：S0 Contract Freeze

## 1. 目标

定义 Task Execution 的不可变 Business Event History，以及用于可靠投递外部信号的 Transactional Outbox，避免数据库和 Workflow/Event Bus 之间发生危险的 Dual Write。

## 2. TaskEvent 契约

```yaml
task_event:
  event_id: string
  tenant_id: string
  project_id: string
  task_id: string
  sequence: int64
  event_type: string
  actor:
    principal_type: string
    subject_id: string
  payload: object
  request_id: string?
  trace_id: string
  schema_version: int
  occurred_at: timestamp
  created_at: timestamp
```

必须保证：

```text
PRIMARY KEY(event_id)
UNIQUE(task_id, sequence)
```

## 3. Event 语义

TaskEvent 是 Append-only Business History。

禁止：

```text
UPDATE task_events
DELETE task_events
为了适配新代码重写历史 payload
```

Schema Evolution 通过新的 `schema_version` 实现，必要时使用兼容 Reader/Upcaster。

## 4. 标准 Event Family

状态事件：

```text
TaskCreated
TaskPlanningStarted
TaskRoutingStarted
TaskExecutionStarted
TaskApprovalRequested
TaskApprovalGranted
TaskApprovalRejected
TaskValidationStarted
TaskCompleted
TaskFailed
TaskCancelled
TaskExpired
```

执行证据事件可以包括：

```text
RouteDecisionRecorded
ModelAttemptStarted
ModelAttemptCompleted
ModelAttemptFailed
PolicyDecisionRecorded
ToolInvocationRequested
ToolInvocationCompleted
ToolInvocationFailed
EvaluationCompleted
CostReconciled
```

并不是每一个低层 Telemetry Span 都属于 TaskEvent。TaskEvent 记录业务上有意义的事实；OpenTelemetry 记录更细粒度的 Execution Telemetry。

## 5. 顺序模型

`sequence` 在每个 Task 内单调递增，由修改 Task Business State 的数据库事务分配。

不要求不同 Task 之间存在 Global Ordering。

Consumer 不得仅依赖 Timestamp 推断事件顺序。

## 6. 与 Task Projection 的原子性

每次 Task State Mutation 和对应 Canonical Event 必须原子提交：

```text
BEGIN
  SELECT task FOR UPDATE / validate version
  UPDATE task projection
  INSERT task_event(sequence = previous + 1)
  INSERT outbox message if delivery required
COMMIT
```

没有对应 Event 的 Task State Change 属于无效状态。

## 7. Outbox 契约

Outbox 用于解决以下不安全模式：

```text
commit PostgreSQL
then signal Temporal
```

或者：

```text
publish event
then fail DB commit
```

标准 Outbox Record：

```yaml
outbox:
  outbox_id: string
  tenant_id: string
  project_id: string
  task_id: string?
  aggregate_type: string
  aggregate_id: string
  event_type: string
  payload: object
  destination: string
  idempotency_key: string
  status: pending | delivering | delivered | dead_letter
  attempts: int
  available_at: timestamp
  created_at: timestamp
  delivered_at: timestamp?
```

## 8. Dispatcher 语义

Outbox Delivery 采用 At-least-once，因此下游 Consumer 必须具备 Idempotency。

```text
DB Commit
  -> Outbox Pending
  -> Dispatcher
  -> Temporal Signal / Event Bus / Webhook
  -> mark Delivered
```

Dispatcher 使用 Bounded Retry + Backoff，并支持 Dead Letter。Outbox Delivery Status 的变化不能修改 TaskEvent History。

## 9. Idempotency

每个外部 Message 必须有稳定 Idempotency Key。Consumer 需要使用 Durable Processed-message Store 或等价 Native Idempotency Mechanism 去重。

Task Command Idempotency 与 Outbox Delivery Idempotency 是两个不同问题，二者都必须实现。

## 10. Consumer Rule

Consumer 必须：

- 校验 Schema Version；
- 校验 Tenant/Project/Task Identity；
- 拒绝不可能的 Cross-scope Reference；
- 对 Duplicate Delivery 安全；
- 不修改历史 TaskEvent；
- 在确认 Delivery 前先持久化自己的 Side Effect。

## 11. Retention

TaskEvent 属于长期 Business/Audit Evidence。Retention 遵循 Tenant/Legal Policy；如果法规要求删除，必须走受治理的 Evidence Retention Process，而不是普通 Application DELETE API。

Delivered Outbox Row 可以在定义的 Retention Period 后压缩或清理，但必须确保 Delivery Evidence 已在其他地方保留。

## 12. 验收条件

- 每个 Task State Transition 恰好产生一个 Canonical State Event；
- `(task_id, sequence)` 唯一；
- Application Repository 无法 Update Historical Event；
- Task Projection、TaskEvent、Outbox 原子提交；
- Dispatcher Retry 不产生重复下游 Side Effect；
- Duplicate Delivery Test 通过；
- Schema Version Compatibility Test 通过；
- 所有 Event/Message 保留 Tenant/Project Identity。

## 13. 对实现的影响

S2 定义 TaskEvent 与 Command Idempotency 的 DB/API Contract；S3 使用 Outbox 与 Durable Workflow Adapter 协调 Temporal，但 Temporal 不成为 Business Event Store。