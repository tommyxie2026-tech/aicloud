# R6 Task Command API 幂等契约

> 状态：Issue #23 实现分片

## 目的

把公共 Task 创建 API 正式接入 R6 Transaction Kernel，使生产 PostgreSQL Runtime 在 HTTP 重试、客户端超时重发等情况下不会重复创建持久化 Task。

## 公共 API 契约

`POST /api/v1/tasks` 必须携带：

```text
Idempotency-Key: <client-stable-key>
```

服务端在解析业务请求后，对规范化请求结构计算 SHA-256 Digest。`X-Request-ID` 等仅属于传输层或观测层的字段，不进入业务请求 Digest。

处理规则：

```text
相同 tenant/project + operation + key + 相同 digest
  -> 重放原来的逻辑 Task 结果

相同 tenant/project + operation + key + 不同 digest
  -> 409 IDEMPOTENCY_CONFLICT

缺少 Idempotency-Key
  -> 400
```

发生成功重放时，响应可以携带：

```text
Idempotency-Replayed: true
```

## 生产事务边界

当 Runtime 使用 PostgreSQL Repository 时，Control Plane 会发现并使用 R6 `TaskCommandStore`，执行：

```text
BEGIN
  reserve command idempotency
  从 Principal 派生 tenant/project/creator
  INSERT Task(CREATED, version=1)
  INSERT TaskCreated(sequence=1)
  INSERT workflow.start Outbox message
  complete Idempotency record
COMMIT
```

Workflow Start 不再在数据库提交以后直接调用 `engine.Start()`，而是作为 `workflow.start` Outbox 投递意图与 Task 创建一起原子提交。这避免了“数据库已经成功，但 Workflow Start 丢失”的 Dual Write。真正的 Outbox Dispatcher 投递属于 R6 下一分片。

## Repository 暴露方式

原有 `TaskRepository` 契约保持不变。Production Scoped PostgreSQL Task Repository 通过可选的 `repository.TaskCommandStoreProvider` 暴露 R6 Transaction Kernel，从而避免把 Domain Repository 接口直接绑定到 PostgreSQL 实现细节。

开发/内存 Repository 当前没有持久化 Command Idempotency，只保留旧的创建路径用于本地测试。因此 Memory Mode 不能作为 R6 “业务命令 Exactly-Once” 的生产证明。

## 当前边界

本分片只正式关闭公共 Task 创建链路。

Route / Model Transition 等 API 目前还不能宣称端到端幂等，因为 RouteDecision、高层命令结果以及后续状态迁移仍需要进一步收敛到 Transaction Kernel。

## 验收证据

- 公共 Task 创建缺少 `Idempotency-Key` 时拒绝请求；
- 相同规范化业务请求得到稳定 Request Digest；
- PostgreSQL Atomic Create / Replay / Conflict / Rollback Integration Test 继续作为持久化语义证据；
- `workflow.start` Outbox 与 Task 创建位于同一事务；
- 中英文实现文档保持同步。
