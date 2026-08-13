# R6 Task Transaction Kernel：TaskEvent、Outbox 与 Idempotency

> 状态：Issue #23 的实现设计

## 1. 目的

R5 已经建立 Task Aggregate 状态机和乐观并发控制，但 Task Projection 与相邻的持久化记录仍可能分别提交。R6 的目标就是关闭这个边界。

核心不变量为：

```text
一次 Task 业务变更
    = Task Projection 更新
    + 恰好一个规范 TaskEvent
    + 必需的 Outbox 投递意图
    + Command Idempotency 结果

以上内容必须位于同一个 PostgreSQL 事务中。
```

任何部分提交都视为无效状态。

## 2. 为什么需要 R6

R5 仍存在典型的 Dual Write 风险：

```text
写入 RouteDecision / 外部投递意图
        |
        v
更新 Task Projection
        |
        +-- Version Conflict / 进程崩溃
```

因此 R6 要把“多个相关写操作”收敛为一个真正的业务事务内核。

## 3. 运行时架构

```text
                    Public Command
                         |
                         v
                Idempotency Gate
                         |
                         v
              Task Command Handler
                         |
                         v
+--------------------------------------------------+
|          PostgreSQL Business Transaction         |
|                                                  |
|  lock / validate Task version                    |
|        |                                         |
|        v                                         |
|  apply Task transition                           |
|        |                                         |
|        +--> append TaskEvent(sequence=N+1)       |
|        |                                         |
|        +--> append Outbox message if required    |
|        |                                         |
|        +--> persist Idempotency result           |
|                                                  |
+--------------------------+-----------------------+
                           |
                         COMMIT
                           |
                           v
                    Outbox Dispatcher
                           |
             +-------------+-------------+
             |             |             |
          Temporal      Event Bus      Webhook
```

Dispatcher 故意放在业务事务之外。外部投递采用至少一次（at-least-once）语义；Outbox 的作用是保证即使业务进程在提交后崩溃，外部投递意图仍不会丢失。

## 4. TaskEvent

TaskEvent 是不可变的业务事实历史，不是普通 Telemetry。

Task 生命周期与规范事件映射如下：

```text
CREATED             -> TaskCreated
PLANNING            -> TaskPlanningStarted
ROUTING             -> TaskRoutingStarted
EXECUTING           -> TaskExecutionStarted
WAITING_APPROVAL    -> TaskApprovalRequested
VALIDATING          -> TaskValidationStarted
COMPLETED           -> TaskCompleted
FAILED              -> TaskFailed
CANCELLED           -> TaskCancelled
EXPIRED             -> TaskExpired
```

TaskEvent 按 Task 独立排序：

```text
UNIQUE(task_id, sequence)
sequence >= 1
```

下一个 sequence 必须在 Task 行已经被锁定或版本验证后的同一事务中分配。时间戳只是证据，不作为唯一排序依据。

在运行时 RLS 中，TaskEvent 只开放 SELECT 和 INSERT 策略。普通运行时路径不允许 UPDATE 和 DELETE。

## 5. Transactional Outbox

Outbox 负责把“需要向外部系统发送什么”与 Task 业务状态一起原子提交。

```text
DB commit
  -> outbox pending
  -> dispatcher lease
  -> deliver
  -> delivered
```

允许的生命周期为：

```text
pending -> delivering -> delivered
                    \-> pending       （有界重试）
                    \-> dead_letter   （重试耗尽或终止性错误）
```

每条消息都必须拥有稳定的 delivery idempotency key。由于传输语义仍是至少一次，下游消费者必须具备去重能力。

## 6. Command Idempotency

公共写操作的幂等作用域固定为：

```text
tenant_id + project_id + operation + idempotency_key
```

请求处理规则：

```text
相同 key + 相同 request digest
    -> 返回 / 重放同一个逻辑结果

相同 key + 不同 request digest
    -> IDEMPOTENCY_CONFLICT
```

Idempotency Record 必须与 Task 业务变更处于同一个数据库事务中。系统不能出现“幂等记录已成功，但业务变更没有提交”，也不能出现相反情况。

## 7. Migration 007

Migration `007_task_event_outbox_idempotency.sql` 创建：

```text
task_events
outbox_messages
idempotency_records
```

三个表全部携带 tenant/project scope，并启用且强制执行 PostgreSQL RLS。

关键数据库约束包括：

- `task_events(event_id)` 主键；
- `UNIQUE(task_id, sequence)`；
- event sequence 和 schema version 必须为正数；
- TaskEvent 不提供运行时 UPDATE/DELETE Policy；
- Outbox 的 delivery idempotency key 具有稳定唯一性；
- Outbox 状态只能属于固定集合；
- Command Idempotency 主键由 tenant/project/operation/key 组成；
- `expires_at` 必须晚于 `created_at`。

## 8. 原子 Repository 边界

R6 的 Production Repository 不允许把一次业务命令暴露成以下四个可以独立提交的调用：

```text
UpdateTask()
AppendEvent()
AppendOutbox()
SaveIdempotency()
```

正确的抽象应该是一个事务级操作，例如：

```text
CommitTaskCommand(command)
```

实现内部只拥有一个 SQL Transaction，所有必需记录要么全部提交，要么全部回滚。

## 9. 并发模型

R5 的乐观并发控制继续保留，只是它现在成为 R6 事务中的一个校验步骤。

推荐事务流程：

```text
BEGIN
  resolve / reserve idempotency key
  SELECT Task ... FOR UPDATE
  validate expected Task version
  validate transition
  allocate next TaskEvent sequence
  UPDATE Task projection + version
  INSERT TaskEvent
  INSERT Outbox message(s)
  complete Idempotency record
COMMIT
```

发生 Task Version Conflict 后禁止 Blind Retry。调用方必须重新加载当前状态，并重新判断业务意图是否仍然成立。

## 10. Crash Semantics

R6 必须显式定义进程崩溃边界。

如果进程在 COMMIT 前崩溃，则 Task Mutation、TaskEvent、Outbox 和 Idempotency Result 都不会持久化。

如果进程在 COMMIT 后、外部投递前崩溃，则 Outbox 仍处于 pending 状态，Dispatcher 可以继续处理。

如果 Dispatcher 已经把消息发送给下游，但还未来得及标记 delivered 就崩溃，那么消息可能会再次发送。此时稳定的 delivery idempotency key 应保证合规消费者不会产生重复副作用。

## 11. 安全与租户边界

每一条 TaskEvent、Outbox Message 和 Idempotency Record 都必须携带 `tenant_id` 与 `project_id`。

Runtime Repository 在访问这些表之前必须设置 transaction-local PostgreSQL scope。即使攻击者猜到了其他项目的 ID，也不能跨 Project 读取或修改记录。

System Dispatcher 不能依赖应用层可控制的 RLS bypass。跨项目工作应该通过明确的 scoped processing，或者通过单独评审的后台执行模型完成。

## 12. Evidence Boundary

TaskEvent 与 OpenTelemetry 的职责必须分离：

```text
TaskEvent      = 持久化的业务重要事实
Trace / Span   = 详细执行遥测
Audit          = 身份、授权与安全证据
Outbox         = 持久化的外部投递意图
```

不能把每一个 Trace Span 都复制成 TaskEvent。

## 13. R6 实现分片

R6 分成四个 Slice：

1. Schema + Domain Contract；
2. Atomic PostgreSQL Task Command Repository；
3. Control Plane Idempotency / Transition 收敛；
4. Outbox Dispatcher Lease / Retry / Dead Letter 与 Crash Tests。

第一个 Slice 完成的标准是：Domain Validation、Migration Contract Test 和真实 PostgreSQL RLS/唯一性测试全部通过。

## 14. Definition of Done

只有满足以下条件，R6 才算完成：

- 每一次 Task 状态变更都生成恰好一个规范状态事件；
- Task Projection 与 TaskEvent 不能独立提交；
- 必需的 Outbox 投递意图与业务变更原子提交；
- Command Idempotency 具备持久化能力并与业务事务绑定；
- 相同的重复命令只执行一次；
- 同一个 Key 携带不同业务请求时稳定返回冲突；
- 并发环境下 TaskEvent 顺序确定；
- 外部投递意图在进程崩溃后不会丢失；
- 重复投递不会产生重复副作用；
- Tenant/Project RLS 测试通过；
- Unit、Integration、Vet 和 Build Gate 全部通过。
