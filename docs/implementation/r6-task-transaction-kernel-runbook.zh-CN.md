# R6 Task Transaction Kernel 运维 Runbook

> 状态：Production-shaped Rollout 与 Verification Contract

## 1. 目的

本文规定如何部署并验证 R6 Task Transaction Kernel，避免 Legacy Task Writer 与新的 TaskEvent、Outbox 和 Idempotency Contract 混写。

R6 将以下边界正式运行化：

```text
Task Create
  = Task + TaskCreated + workflow.start Outbox + Idempotency

Task Transition
  = Task Projection + Canonical TaskEvent + Required Outbox + Idempotency

Route Command
  = RouteDecision + Task Routing Projection + TaskRoutingStarted + Idempotency

Model Command
  = Begin(EXECUTING + TaskExecutionStarted + in_progress Idempotency)
    -> Remote Provider Attempt(s)
    -> Finalize(Task Final Projection + Canonical Events + Final Idempotency)
```

## 2. Preflight

部署前必须确认：

1. S1/R5 Migration 已完成；
2. Runtime DB Role 不是 SUPERUSER/BYPASSRLS；
3. API 与 Worker 使用 Scoped Runtime DB Role；
4. Migration Credential 与 Runtime Credential 分离；
5. Cutover 期间没有旧版本 Writer 继续修改 Task Status；
6. 现有 Task 都具有 Tenant、Project、Creator、Version Identity。

建议检查：

```sql
SELECT count(*) FROM tasks
WHERE tenant_id IS NULL OR project_id IS NULL OR created_by IS NULL OR version < 1;
```

结果必须为 0。

## 3. Drain Legacy Writer

执行 R6 Schema 前：

```text
停止接受新的 Mutating Task Request
        |
        v
Drain 旧 API Instance
        |
        v
Drain 旧 Worker
        |
        v
确认没有 Legacy Task Mutation Session
```

禁止 R5/R6 Task Writer 同时写入同一数据库。

## 4. 执行 Migration

使用独立 Migration Credential，至少执行到：

```text
007_task_event_outbox_idempotency.sql
008_outbox_dispatch_leases.sql
```

Schema Verification 通过之前不得恢复业务写流量。

## 5. Schema Verification

必须验证：

- `task_events` 存在；
- `outbox_messages` 存在；
- `idempotency_records` 存在；
- TaskEvent 具有 `UNIQUE(task_id, sequence)`；
- 三个 R6 Table 都 ENABLE + FORCE RLS；
- TaskEvent Runtime Policy 不提供 UPDATE/DELETE；
- Outbox Lease Column 已存在；
- Runtime DB Role 无法绕过 RLS。

## 6. 推荐部署顺序

```text
1. Migration
2. API R6 Build
3. Worker / Dispatcher R6 Build
4. Health / Readiness Verification
5. 打开 Mutating Traffic
```

API 可以在 Dispatcher 尚未启动时提交 Outbox。因为 Delivery Intent 已持久化为 Pending，Dispatcher 启动后可以恢复处理，不会丢失。

## 7. Smoke Test：Task Creation

使用同一个 `Idempotency-Key` 连续发送两次完全相同的请求。

期望：

```text
第一次：
  HTTP 202
  创建新 Task

第二次完全相同：
  HTTP 202
  返回同一个 Task
  Idempotency-Replayed: true
```

数据库中必须只有一个：

- Task Row；
- sequence=1 的 `TaskCreated`；
- `workflow.start` Outbox Intent；
- Completed Public Idempotency Record。

保持 Key 不变但修改业务 Body，必须返回 HTTP 409 `IDEMPOTENCY_CONFLICT`。

## 8. Smoke Test：Routing

使用稳定 Route Idempotency-Key 对 CREATED Task 执行路由。

预期 Durable Fact：

```text
TaskPlanningStarted
TaskRoutingStarted
RouteDecision
Task.status = ROUTING
Task.route_decision_id = RouteDecision.id
```

完全相同的 Route Retry 必须重放原 RouteDecision，而不能因为 Health/Capacity/Pricing 等 Runtime Signal 已变化就重新规划出另一个结果。

## 9. Smoke Test：Model Execution

对已路由 Task 使用 Model Idempotency-Key 执行 Model Command。

成功状态机：

```text
ROUTING
  -> EXECUTING
  -> VALIDATING
  -> COMPLETED
```

对应 TaskEvent：

```text
TaskExecutionStarted
TaskValidationStarted
TaskCompleted
```

同一个公共 Logical Model Operation 的 `operation_id` 必须稳定；Provider Fallback/Retry 产生的每个 Physical Attempt 必须具有不同 `attempt_id`。

## 10. Retryable Provider Failure

注入 Unavailable、Timeout、Rate Limit 等可重试错误。

期望：

```text
Task.status = EXECUTING
Idempotency.status = failed_retryable
```

使用同一个 Public Key 重试时允许产生新的 Physical Provider Attempt，但不能再次追加 `TaskExecutionStarted`。

## 11. Ambiguous Provider Execution

如果进程在 Execution Begin Transaction 已 COMMIT、Model Finalization 尚未 COMMIT 时崩溃：

```text
Task.status = EXECUTING
Idempotency.status = in_progress
```

重复公共请求必须返回 In-progress/Conflict 语义，禁止 Blind Provider Replay。

这类 Ambiguous Operation 的恢复策略属于后续 Durable Workflow/Reconciler Contract。

## 12. Outbox Crash Recovery

必须验证以下边界：

```text
Lease Outbox
  -> Downstream Delivery Success
  -> Dispatcher 在 MarkDelivered 前崩溃
```

Lease Expire 后，新 Dispatcher 必须重新 Claim Message。允许出现第二次 Physical Delivery，但 Downstream 必须依据稳定 Idempotency Key 把两次 Delivery 收敛为一次 Business Effect。

验证结果应为：

```text
Physical Deliveries >= 2
Business Effects = 1
Outbox Final Status = delivered
```

## 13. TaskEvent Concurrency Verification

多个 Concurrent Command 同时针对同一个 Task Version 时，只允许一个 Mutation 成功；其他 Writer 必须得到 `ErrVersionConflict`，不能形成 Parallel Business History。

Stress Test 后：

```text
TaskEvent sequence = 1,2,3,...,N
```

必须连续，不允许 Rolled-back Contender 造成 Sequence Gap，也不允许重复 `(task_id, sequence)`。

## 14. Observability Verification

必须可以关联：

```text
request_id
trace_id
task_id
logical model operation_id
physical model attempt_id
outbox idempotency_key
```

TaskEvent 继续表示 Business History；Trace 表示 Execution Telemetry，二者不能混淆。

## 15. Failure Handling

### Migration 完成前失败

停止部署，修复 Preflight/Migration，保持 Mutating Traffic 关闭。

### R6 Schema 已生效、R6 Application 尚未完成发布

继续保持旧 Writer Drain。优先 Forward Fix R6 Application，不要恢复 Legacy Writer。

### R6 已写入 TaskEvent/Idempotency 后

不要把 Writer 回滚到不理解 R6 Transaction Contract 的版本。除非存在经过测试的 Backward-compatible Rollback Release，否则采用 Forward Fix。

## 16. R6 Go/No-Go Gate

只有以下全部满足，R6 才允许 Merge/Deploy：

- Task Create 原子幂等；
- 所有 Production Task State Mutation 进入 R6 Command Kernel；
- RouteDecision 与 Routing State 原子提交；
- Model Lifecycle 使用 Begin/Finalize Command Boundary；
- Concurrent TaskEvent Ordering Test 通过；
- Outbox Crash Recovery 已证明；
- Duplicate Physical Delivery 只产生一次 Downstream Business Effect；
- Tenant/Project RLS Integration Test 通过；
- Bilingual Documentation Validation 通过；
- `go test ./...`、Integration Test、`go vet ./...`、Entrypoint Build 全部通过。
