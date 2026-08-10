# R5 Task Aggregate 状态迁移实现

## 状态

已在 `agent/task-aggregate-transitions` 实现，等待 CI 与评审。

## 目标

把 Task 从“可以任意修改 status 的数据记录”收敛为 S0 契约定义的执行聚合根。在 R6 引入 TaskEvent、Transactional Outbox 与命令幂等之前，R5 先建立受验证的生命周期迁移和乐观并发控制。

## 标准生命周期

```text
CREATED
   ↓
PLANNING
   ↓
ROUTING
   ↓
EXECUTING
   ├──────────────┐
   ▼              │
WAITING_APPROVAL  │
   │              │
   └──────► EXECUTING
                    ↓
                VALIDATING
                    ↓
                COMPLETED
```

任意已知非终态都可以根据明确命令进入 `FAILED`、`CANCELLED` 或 `EXPIRED`。终态不能重新进入非终态。

## Aggregate API

Task 生命周期变化必须通过：

```go
transition, err := task.Transition(domain.TaskTransitionCommand{
    To:    domain.TaskRouting,
    Actor: "service_account:worker-1",
    Cause: "request model route",
    At:    now,
})
```

Transition 会校验 Source/Target State，并强制要求 Actor、Cause 与时间证据。Runtime 代码禁止直接执行：

```go
task.Status = ...
```

`domain.NewTask` 创建的新 Task 固定从 `CREATED`、`version=1` 开始。

## 乐观并发控制

Task Repository 使用 Task 的 `version` 作为期望资源版本：

```text
读取 Task(version=N)
      ↓
执行 Aggregate Mutation
      ↓
Update(expected=N)
      ↓
version=N+1
```

如果写入基于旧版本，则返回：

```text
repository.ErrVersionConflict
```

调用方必须重新加载当前 Task，并重新判断原命令是否仍成立，禁止 Blind Retry。

Memory Repository 与 `ScopedPostgresTasks` 使用同一并发契约。

## PostgreSQL 持久化

Migration `006_task_aggregate_state.sql`：

- 新增 `version BIGINT NOT NULL DEFAULT 1`；
- 新增 nullable `completed_at`；
- 显式迁移 Prototype 状态 `PENDING -> CREATED`；
- 显式迁移 Prototype 状态 `RUNNING -> EXECUTING`；
- 为历史终态补齐 `completed_at`；
- 约束 version 必须大于等于 1；
- 约束 status 只能属于标准生命周期；
- 新增 Tenant/Project/Status/Created 组合索引。

PostgreSQL Update 使用版本条件：

```sql
UPDATE tasks
SET ..., version = version + 1
WHERE id = $1 AND version = $expected
RETURNING version;
```

因此两个并发 Writer 同时读取 `version=N` 时，最多只有一个 Writer 能成功提交为 `N+1`。

## Control Plane 收敛

### Create

`CreateTask` 使用 `domain.NewTask`，持久化后的 Task 初始状态为：

```text
CREATED
version=1
```

### Route

`DecideRoute` 必须按照：

```text
CREATED -> PLANNING -> ROUTING
```

推进 Task。

Routing Retry 可以继续停留在 `ROUTING`。其他不兼容状态会在 Router 操作之前被拒绝。

### Model Execution

`ExecuteModel` 首先推进：

```text
ROUTING -> EXECUTING
```

模型执行成功后：

```text
EXECUTING -> VALIDATING -> COMPLETED
```

Provider 执行失败时：

```text
EXECUTING -> FAILED
```

R5 会为这些 Transition 写 Trace Evidence。R6 将进一步把 TaskEvent 作为持久化 Business History，并实现 Task Projection、TaskEvent 与 Outbox 的原子事务。

## Failure Semantics

- 非法生命周期跳转：`domain.ErrInvalidTaskTransition`；
- 从终态再次迁移：`domain.ErrTaskTerminal`；
- Repository 旧版本写入：`repository.ErrVersionConflict`；
- Route/Model Operation 在 Task 状态不兼容时必须在受控操作发生前失败。

## R5 有意保留的边界

R5 不宣称已经实现 Task 与 TaskEvent 原子持久化。

以下事务模型明确留给 R6：

```text
Task Projection
+
TaskEvent
+
Outbox
+
Idempotency Record
```

R6 会把本 Slice 已经产生的 Transition Result 转换成 Append-only TaskEvent，并在单一数据库事务中提交。

## 测试要求

必须覆盖：

- 标准生命周期完整路径；
- Approval Loop；
- 跳过状态被拒绝；
- 终态不可重新打开；
- 所有非终态进入 Fail/Cancel/Expire；
- Transition Actor/Cause/Time 强制要求；
- Memory Repository Stale Write 被拒绝；
- PostgreSQL Stale Write 被拒绝；
- Migration 状态映射与数据库约束；
- 原有 Tenant/RLS 测试继续通过。

## Definition of Done

R5 完成必须满足：

- Runtime Control Plane 不再直接修改 Task Status；
- Task 创建固定为 `CREATED/version=1`；
- 状态迁移结果确定且可测试；
- 终态无法重新进入执行状态；
- Memory 与 PostgreSQL Repository 都执行 Version Conflict 检查；
- Migration 006 通过真实 PostgreSQL Integration Test；
- 中英文文档同步；
- `gofmt`、Unit Test、PostgreSQL Integration Test、`go vet` 与所有入口 Build 全部通过。
