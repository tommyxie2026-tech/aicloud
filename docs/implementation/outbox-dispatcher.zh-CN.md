# R6 Outbox Dispatcher Lease 与 Retry

> 状态：Issue #23 实现分片

## 目的

实现 R6 Transactional Outbox 的持久化投递侧，同时保持 Tenant/Project RLS 边界，并明确外部传输仍然是 At-Least-Once，而不是伪装成 Exactly-Once。

## 投递模型

业务命令提交时只把外部投递意图写成 `pending` Outbox。Dispatcher 在事务提交之后处理：

```text
pending
  -> delivering（持有 lease）
       -> delivered
       -> pending（退避后重试）
       -> dead_letter
```

如果 Dispatcher 已经让下游产生副作用，但在 `MarkDelivered` 前崩溃，该消息可能再次被投递。因此稳定的 Outbox `idempotency_key` 仍然是下游去重身份。

## Lease 契约

Migration 008 新增：

```text
lease_owner
lease_expires_at
last_error
```

只有 `delivering` 状态允许持有 Lease。`pending`、`delivered` 与 `dead_letter` 不允许保留活动 Lease。

`Lease()` 使用 `FOR UPDATE SKIP LOCKED`，因此同一个 Project Scope 内多个 Worker 可以并行领取不同消息，而不会重复领取同一条仍处于有效 Lease 的记录。

如果 `delivering` Lease 已经过期，新的 Worker 可以重新领取，这就是 Dispatcher 进程崩溃后的恢复机制。

每次成功领取都会增加 `attempts`。记录成功或失败时必须校验 Lease Owner，因此旧 Worker 在 Lease 被其他 Worker 接管以后不能再把消息错误地标记为完成。

## Retry 与 Dead Letter

`FailDelivery()` 保存最近一次错误，并根据 Attempt Budget 执行二选一：

- 回到 `pending`，使用调用方给出的 `available_at` 作为下次退避时间；
- 达到最大尝试次数后进入 `dead_letter`。

Retry Backoff 算法不固化在 Repository 中，从而使指数退避、抖动和不同 Destination 的策略可以独立演化。

## Tenant / Project 边界

Dispatcher Repository 故意保持 Project Scoped。调用者必须拥有显式 Project Principal；Repository 在 Lease 或更新之前设置 Transaction-Local Tenant/Project Context。

这里没有新增 Runtime RLS Bypass，也不允许通过应用可控 Session Flag 扫描所有 Tenant。未来跨 Project 调度必须由单独评审的 Runtime Orchestration 机制提供。

## 当前实现证据

- `db/migrations/008_outbox_dispatch_leases.sql`
- `db/migrations/r6_outbox_contract_test.go`
- `internal/repository/postgres_outbox.go`
- `internal/repository/postgres_outbox_integration_test.go`

Integration Test 已覆盖：有效 Lease 不可被抢占、过期 Lease 恢复、旧 Owner 拒绝、Retry、Dead Letter 和成功 Delivered。

## 当前剩余边界

本分片建立的是 Durable Dispatcher Persistence Primitive。Worker 进程还需要真正的 Delivery Adapter，把 `workflow.start` 等 Destination 映射到具体消费者，并把稳定 Delivery Idempotency Key 传递给下游。
