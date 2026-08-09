# AI Cloud Task Aggregate 契约

> 状态：S0 Contract Freeze

## 1. 目标

Task 是 AI Cloud 中最核心的业务执行单位，也是所有受治理 AI Work 的 Aggregate Root。Routing、Approval、Tool Invocation、Cost、Audit、Evaluation 等执行证据最终都必须归属于某个 Task。

## 2. Aggregate 结构

```text
Task
  ├─ Identity
  │   ├─ task_id
  │   ├─ tenant_id
  │   └─ project_id
  ├─ Request
  │   ├─ goal
  │   ├─ input
  │   ├─ agent_id/version
  │   └─ requested constraints
  ├─ Execution
  │   ├─ workflow_id
  │   ├─ status
  │   ├─ version
  │   └─ current step
  ├─ Result
  └─ Evidence references
```

Tenant 与 Project 是 Task 创建后的不可变身份字段；`created_by` 记录创建 Task 的 Principal。

## 3. 标准 Task 字段

```yaml
task:
  task_id: string
  tenant_id: string
  project_id: string
  created_by: string
  agent_id: string
  agent_version: string
  goal: string
  input: object
  constraints: object
  status: TaskStatus
  version: int64
  workflow_id: string?
  result: object?
  failure: object?
  created_at: timestamp
  updated_at: timestamp
  completed_at: timestamp?
```

Provider、Model Attempt、Tool Invocation、Cost 不应该作为 Task 内部不断覆盖的可变字段，而是作为独立的 Task-owned Record/Event 保存。

## 4. 状态机

```text
CREATED
   ↓
PLANNING
   ↓
ROUTING
   ↓
EXECUTING
   ├───────────────┐
   ▼               │
WAITING_APPROVAL   │
   │               │
   └──────► EXECUTING
                    │
                    ▼
                VALIDATING
                    │
          ┌─────────┼─────────┐
          ▼         ▼         ▼
      COMPLETED   FAILED   CANCELLED
```

任意非终态在符合 Transition Rule 时，也可以进入：

```text
FAILED
CANCELLED
EXPIRED
```

终态为：

```text
COMPLETED
FAILED
CANCELLED
EXPIRED
```

## 5. Transition Rule

Task Status 只能通过 Aggregate Transition API 修改。

禁止：

```go
task.Status = TaskCompleted
repo.Update(task)
```

要求：

```text
Task.Transition(command)
  -> validate current state
  -> validate actor/cause
  -> produce new Task state
  -> produce TaskEvent
  -> persist atomically
```

每一个 Transition 必须定义：

- Allowed Source State；
- Target State；
- Command/Cause；
- Required Principal/Capability；
- Required Evidence；
- Side Effect 可以发生在 Transition 前还是后。

## 6. Version 与并发控制

Task 使用 Integer `version` 做 Optimistic Concurrency Control：

```text
UPDATE tasks
SET ..., version = version + 1
WHERE task_id = ? AND version = expected_version
```

Version Conflict 不能 Blind Retry。必须重新加载当前 Task 状态并重新判断 Command 是否仍然成立。

## 7. 创建不变量

Task Creation 必须满足：

- 已验证 Principal；
- Tenant 与 Project Scope；
- 合法 Agent Reference 或显式 System Workflow Type；
- Goal/Input 满足 API Schema；
- Public Mutating API 具有 Idempotency Key；
- 生成 Request ID 与 Trace ID。

Task 与首个 `TaskCreated` Event 必须原子提交。

## 8. Ownership

长期 Schema 必须直接在 `tasks` 表中包含：

```text
tenant_id
project_id
created_by
```

当前 S1 的 `task_ownership` 表只作为 Migration Bridge，不作为 Task Identity 的长期 Source of Truth。

Cross-Project 或 Cross-Tenant Task Move 不是普通 Update。如果未来支持，必须采用显式 Migration Operation 并留下完整 Audit Trail。

## 9. Failure Contract

Task Failure 使用结构化模型：

```yaml
failure:
  code: string
  category: validation | policy | provider | tool | sandbox | workflow | internal
  message: string
  retryable: bool
  source_ref: string?
  occurred_at: timestamp
```

某一次 Provider/Tool Attempt 失败不等于 Task 必然 FAILED；是否可以 Retry/Fallback 由 Workflow Policy 判断。

## 10. Cancellation 与 Expiration

Cancellation 是 Command，不是字段赋值。必须具备 Idempotency，并传播到 Durable Workflow。

Expiration 基于显式 Deadline/TTL Policy，并产生 Event。

已经完成的外部 Side Effect 不得因为 Task Cancel 就被隐式回滚；如需补偿，必须使用显式 Compensating Workflow。

## 11. Evidence Ownership

以下记录必须关联 `task_id`：

```text
TaskEvent
RouteDecision
ModelAttempt
PolicyDecision
Approval
ToolInvocation
CredentialGrant
SandboxExecution
CostEvent
AuditEvent（与 Task 相关时）
EvaluationRun（生产 Task 评估）
```

## 12. Persistence Transaction

Business State Change 应遵循：

```text
BEGIN
  validate expected task version
  update Task projection
  append TaskEvent
  append Outbox record when external delivery is required
COMMIT
```

Task State 不允许在没有对应 TaskEvent 的情况下单独 Commit。

## 13. 验收条件

- Public Domain API 无法直接任意修改 Task Status；
- Invalid Transition 确定性失败；
- 每次成功 Transition 恰好产生一个标准 State Event；
- Task 与 Initial Event 原子创建；
- Stale Write 被 Optimistic Concurrency 拒绝；
- Tenant/Project Identity 不能通过普通 Update 改变；
- Cancellation 幂等；
- Terminal Task 不允许重新进入 Non-terminal State，除非未来通过明确的新 Recovery/Retry Task Contract 实现。

## 14. 对实现的影响

S2 必须把运行 API/Schema 收敛到本契约；S3 的 Durable Workflow 只能编排 Task，不拥有 Task 的 Business State。