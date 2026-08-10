# AI Cloud 幂等性契约

> 状态：S0 Contract Freeze

## 1. 目标

防止 HTTP Retry、Workflow Retry、Provider Fallback、Tool Retry 相互叠加，造成重复业务操作或重复外部 Side Effect。

## 2. 幂等性分层

```text
API Command Idempotency
Workflow Activity Idempotency
Provider Attempt Identity
Tool/Side-effect Idempotency
Outbox Delivery Idempotency
```

这些层次相互关联，但不能互相替代。

## 3. Public Mutating API

所有 Public Mutation 至少必须要求 `Idempotency-Key`：

```text
POST /tasks
POST /tasks/{id}:cancel
POST /tasks/{id}:approve
POST /tasks/{id}/...mutating-command
```

Idempotency Scope：

```text
tenant_id + project_id + operation + idempotency_key
```

## 4. Idempotency Record

```yaml
idempotency_record:
  tenant_id: string
  project_id: string
  operation: string
  key: string
  request_digest: string
  status: in_progress | completed | failed_retryable | failed_final
  resource_id: string?
  response_code: int?
  response_digest: string?
  response_payload: object?
  created_at: timestamp
  expires_at: timestamp
```

唯一约束：

```text
UNIQUE(tenant_id, project_id, operation, key)
```

## 5. Request Matching

规则：

```text
same key + same canonical request digest
  -> 返回同一个 Logical Result

same key + different request digest
  -> 409 IDEMPOTENCY_CONFLICT
```

Canonical Request Digest 不包含 Request ID 等纯 Transport 字段，但必须包含所有会改变业务意图的字段。

## 6. Transaction Semantics

Task Creation：

```text
BEGIN
  reserve idempotency record
  create Task
  append TaskCreated event
  store resource/result reference
COMMIT
```

Idempotency Record 与 Business Mutation 不能分开 Commit。

## 7. Concurrent Duplicate Request

两个相同 Key 并发到达时，只允许一个 Request 成为 Operation Owner。另一个根据 API Contract 等待/查询或返回 Stable In-progress Response，绝不能再次执行 Business Command。

## 8. Workflow Activity Idempotency

所有产生 Side Effect 的 Activity 必须有稳定 `operation_id`，例如：

```text
hash(task_id + logical_step + proposal_digest + tool_id)
```

Workflow Retry 必须复用相同 Operation Identity，除非 Policy 明确创建新的 Logical Attempt。

## 9. Provider Attempt

Model Generation 可能合法 Retry/Fallback，因此每次物理调用使用不同 `model_attempt_id`，但上层 Logical Model Operation 使用稳定 Operation ID。

Cost/Trace 必须同时区分 Logical Operation 与 Physical Attempt。

## 10. Tool Side Effect

Tool Adapter 至少采用以下一种方式保证 Retry Safety：

1. Target API 原生 Idempotency Key；
2. AI Cloud Durable Side-effect Ledger；
3. Desired-state Reconciliation，使重复执行本身安全。

没有 Deduplication/Compensation 的 Non-idempotent Tool 不得开启自动 Retry。

## 11. Approval

Approval 幂等键：

```text
task_id + proposal_digest + reviewer/action
```

重复相同 Approval 返回已有 Decision；Proposal Digest 变化后必须重新 Approval。

## 12. Cancellation

Cancellation 必须幂等。对已经 Cancelled/Terminal 的 Task 重复 Cancel，只返回稳定当前状态，不能重复触发外部 Compensation。

## 13. Outbox Delivery

每个 Outbox Message 具有稳定 Delivery Idempotency Key。At-least-once Delivery 要求下游 Consumer 可去重。

## 14. Retention

Idempotency Retention 必须覆盖最大 Client Retry Window 与相关 Workflow Retry Horizon。高风险 Side Effect 的 Key 通常需要比普通 API Creation 更长的 Retention。

## 15. 验收条件

- 重复相同 Task Creation 返回同一 Task ID；
- Same Key + Different Body 返回 Conflict；
- Concurrent Duplicate Command 只执行一次；
- Workflow Restart 不重复 Tool Side Effect；
- Provider Retry 产生多个 Attempt，但只有一个 Logical Operation；
- Approval/Cancellation 幂等；
- Outbox Duplicate Delivery 安全；
- 测试覆盖 Request Receipt、DB Commit、External Delivery 之间 Crash 的情况。