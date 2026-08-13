# R6 Model Execution 幂等与生命周期边界

> 状态：R6 Implementation Contract

## 1. 目标

把一次公共 Model Execution Request 定义为一个持久化的 Logical Command，同时允许 Provider Runtime 在该逻辑命令内部产生一个或多个 Physical Attempt。该设计明确不宣称 Provider Transport 具备全局 Exactly-Once 语义。

```text
Public Model Command
  = Stable Logical Operation Identity
  + Durable Command Idempotency
  + Canonical Task Lifecycle Events
  + One or More Physical Model Attempts
```

## 2. 公共 API 契约

```text
POST /api/v1/tasks/{task_id}/model
Idempotency-Key: <stable logical command key>
```

Canonical Request Digest 必须包含 Task Identity 和所有影响业务意图的 Provider Request 字段。`requestId` 不进入 Digest，因为它属于 Transport/Logical-call Metadata；Control Plane 会将其替换为稳定的 Logical Model Operation ID。

Command Idempotency Scope 继续保持：

```text
tenant_id + project_id + operation + idempotency_key
```

## 3. Logical Operation 与 Physical Attempt 分离

Provider Retry 或 Fallback 不创建新的公共业务命令：

```text
Logical Model Operation
        |
        +-- Physical Attempt 1
        +-- Physical Attempt 2
        +-- Fallback Attempt 3
```

Logical Operation ID 对同一个 Task + Public Idempotency-Key 保持稳定；每次 Physical Provider Attempt 则可以独立 Trace、计费和审计。

## 4. 为什么 Provider Call 不进入 PostgreSQL Transaction

Provider Call 是远程 Side Effect，无法加入 PostgreSQL 本地事务。把网络调用包在数据库 Transaction 内既不会获得 Distributed Exactly-Once，又会在不可预测的 Provider Latency 期间长期占用数据库锁。

因此 R6 在远程调用前后建立两个 Durable Database Boundary。

## 5. Begin Transaction

第一次 Physical Attempt：

```text
BEGIN
  reserve public command idempotency as in_progress
  SELECT Task FOR UPDATE
  validate ROUTING -> EXECUTING
  UPDATE Task projection + version
  INSERT TaskExecutionStarted
COMMIT

Provider Call 在 COMMIT 之后执行
```

如果相同 Logical Command 已经是 `in_progress`，Duplicate Request 必须 Fail Closed，返回 `IDEMPOTENCY_IN_PROGRESS`，不得再次调用 Provider。

## 6. Retryable Provider Failure

Provider Timeout、Unavailable、Rate Limit 或其他被分类为 Retryable 的失败，不立即把 Task 置为 FAILED：

```text
Task remains EXECUTING
Idempotency:
  in_progress -> failed_retryable
```

之后使用相同 Idempotency-Key 的显式 Retry 可以重新获得 Logical Command Ownership：

```text
failed_retryable -> in_progress
```

因为 Task 已经处于 `EXECUTING`，该 Retry 不会再次写入 `TaskExecutionStarted`，但可以产生新的 Physical Provider Attempt。

## 7. Successful Finalization

Provider 成功返回后：

```text
BEGIN
  lock in_progress idempotency
  SELECT Task FOR UPDATE
  validate EXECUTING -> VALIDATING -> COMPLETED
  UPDATE final Task projection
  每个逻辑状态迁移对应一次 Task version increment
  INSERT TaskValidationStarted
  INSERT TaskCompleted
  complete idempotency with replayable model result
COMMIT
```

最终 Task Projection 与两个 Canonical Business Event 必须原子一致。

## 8. Final Non-Retryable Failure

对于最终不可重试失败：

```text
BEGIN
  lock in_progress idempotency
  SELECT Task FOR UPDATE
  validate EXECUTING -> FAILED
  UPDATE Task projection
  INSERT TaskFailed
  mark idempotency failed_final with replayable error evidence
COMMIT
```

之后相同 Key 的 Retry 必须返回原来的 Durable Logical Result，而不是再次调用 Provider。

## 9. Crash Semantics

### Begin COMMIT 之前崩溃

没有 Execution-start Business Fact 被持久化，也不应该已经开始 Provider Call。

### Begin COMMIT 之后、Provider Call 之前崩溃

Task 为 `EXECUTING`，Command 为 `in_progress`。系统不能只根据本地进程状态推断 Provider 是否执行，因此禁止 Blind Replay；后续恢复属于 Workflow/Reconciliation 责任。

### Provider Call 完成之后、Finalization COMMIT 之前崩溃

采用同样的保守规则：Command 保持 `in_progress`，重复公共请求不会盲目重复 Physical Provider Call。

这是刻意设计的 Fail-Closed Semantics。

## 10. Cost 与 Evidence

Provider Attempt 继续负责 Attempt-level Trace 与 Cost Evidence。R6 TaskEvent 只记录业务生命周期事实：

```text
TaskExecutionStarted
TaskValidationStarted
TaskCompleted / TaskFailed
```

不能把每个 Provider Transport Detail 都复制为 TaskEvent。

## 11. Security Boundary

所有 Begin / Finalize / Idempotency 操作都必须具有显式 Project-scoped Principal。System Principal 仍必须具备冻结契约中的 `task:system-access` Capability。PostgreSQL Transaction-local Tenant/Project Scope 继续作为 RLS Boundary。

## 12. Non-Goals

R6 不提供：

- Provider Transport 的全局 Exactly-Once；
- 对 Ambiguous `in_progress` Provider Call 的自动重复执行；
- Temporal Recovery Policy；
- OIDC/RBAC/ABAC 收敛；
- Provider-specific Distributed Transaction。

## 13. Acceptance Criteria

- Public Model Execution 必须要求 `Idempotency-Key`；
- `requestId` 改变不会创建新的 Business Command；
- 已完成的完全相同 Retry 重放原 Logical Result；
- Same Key + Changed Business Request 返回 Conflict；
- Provider Execution 状态不确定时，并发 Duplicate 返回 In-progress 且不能再次调用 Provider；
- 第一次执行只产生一个 `TaskExecutionStarted`；
- Retryable Failure 让 Task 保持 `EXECUTING`，并允许相同 Key 的显式 Reacquisition；
- Success Finalization 原子记录 `VALIDATING` 和 `COMPLETED`；
- Final Failure 原子记录 `FAILED`；
- Task Version 每个 Logical State Transition 增加一次；
- PostgreSQL Integration Test、Unit Test、Vet 与 Build 全部通过。
