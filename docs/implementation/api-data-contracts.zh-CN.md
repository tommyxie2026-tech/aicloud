# AI Cloud API 与数据契约

## 1. API 通用规范

Base Path 固定为 `/api/v1`。公共 JSON 使用 `camelCase`；内部 SQL 与 Durable Event Schema 可以继续使用 `snake_case`。时间使用 RFC3339 UTC，ID 使用 Opaque String。

Authentication 产生经过验证的 `identity.Principal`。公共 Handler 不能把客户端提供的 Tenant/Project 当作已认证安全上下文的替代。在 OIDC 模式下，Tenant/Project/Role/Capability 只能来自完成密码学验证的 Claim。Trusted Ingress 只是显式兼容模式，要求独立可信入口已经完成认证并强制替换身份 Header。

Request/Trace Correlation 在 Authentication 之前建立：

```text
X-Request-ID
X-Trace-ID
```

非法或缺失的 Correlation ID 由 API Boundary 重新生成，并在 Response 中返回。

当前 Durable `Idempotency-Key` 语义只适用于 R6 Task Command Kernel：

- Task 创建；
- Task 路由；
- Logical Model Execution。

对于这些 Operation，相同 Key + 相同 Canonical Request 会重放第一次 Business Result；相同 Key + 不同 Request 返回 HTTP 409。其他 Mutation 在建立自己的 Command Transaction 之前不能宣称 Durable Exactly-Once。

## 2. 统一错误 Envelope

所有 `/api/v1` Error 使用：

```json
{
  "error": {
    "code": "INVALID_REQUEST",
    "message": "request is invalid",
    "request_id": "req-...",
    "trace_id": "trace-...",
    "retryable": false,
    "details": {}
  }
}
```

核心 Boundary Error Code 包括 `INVALID_REQUEST`、`UNAUTHENTICATED`、`FORBIDDEN`、`NOT_FOUND`、`CONFLICT`、`METHOD_NOT_ALLOWED`、`RATE_LIMITED`、`SERVICE_UNAVAILABLE`、`INTERNAL_ERROR`、`IDEMPOTENCY_CONFLICT` 与 `IDEMPOTENCY_IN_PROGRESS`。Domain Layer 可以在不改变 Envelope 的前提下增加稳定 Code。

## 3. 可执行 v0.1 公共 REST Surface

机器可读的 Source of Truth 是 `docs/implementation/contracts/openapi-v1.yaml`。R7D 只记录已经由运行时实现、并进入 R7 Security Boundary 的 Operation：

```text
GET/POST  /api/v1/models
GET/PUT   /api/v1/models/{model_id}
GET/POST  /api/v1/models/{model_id}/admission
GET       /api/v1/tools

GET/POST  /api/v1/tasks
GET       /api/v1/tasks/{task_id}
POST      /api/v1/tasks/{task_id}/route
GET       /api/v1/tasks/{task_id}/routes
GET       /api/v1/tasks/{task_id}/costs
GET       /api/v1/tasks/{task_id}/audit
POST      /api/v1/tasks/{task_id}/model
GET       /api/v1/tasks/{task_id}/trace
GET/POST  /api/v1/tasks/{task_id}/evaluations
POST      /api/v1/tasks/{task_id}/tools/{tool_id}
```

Cancel、Approval、TaskEvent Query/Streaming、Agent CRUD、Policy CRUD、Project CRUD 和 Global Audit Query 在对应 Handler 真正存在之前仍属于架构或后续产品契约，不能作为已经完成的 v0.1 HTTP Operation 对外声明。

## 4. Task HTTP 契约

Task 创建使用已验证 Tenant/Project Scope，并要求 `Idempotency-Key`：

```json
{
  "input": "scale dev-gpu-cluster gpu-workers from 3 to 6",
  "agentId": "infra-agent"
}
```

未声明的 Top-level Field 必须拒绝。Tenant/Project 不从 Body 接受。

HTTP 202 返回 Aggregate Projection，例如：

```json
{
  "id": "task-...",
  "tenantId": "tenant-...",
  "projectId": "project-...",
  "createdBy": "user-...",
  "agentId": "infra-agent",
  "input": "scale dev-gpu-cluster gpu-workers from 3 to 6",
  "status": "CREATED",
  "version": 1,
  "traceId": "trace-...",
  "createdAt": "...",
  "updatedAt": "..."
}
```

Canonical Task Status：

```text
CREATED
PLANNING
ROUTING
EXECUTING
WAITING_APPROVAL
VALIDATING
COMPLETED
FAILED
CANCELLED
EXPIRED
```

直接 `GET /tasks/{task_id}` 还返回：

```text
ETag: "task:<task-id>:v<version>"
X-Resource-Version: <version>
```

`version` 是 Aggregate/PostgreSQL 的 Optimistic Revision。R7D 不会提前引入 `If-Match`；只有某个 Public Command 真正接入 Expected-Version Semantics 后才允许增加该 Precondition。

## 5. Pagination

List 与 Evidence List Endpoint 使用有界分页：

- `pageSize`：默认 50，最小 1，最大 200；
- `pageToken`：上一个 Response 返回的 Opaque Continuation Token；
- Response：`{ "items": [...], "nextPageToken": "..." }`。

Client 必须把 `pageToken` 当作 Opaque Value。非法 Size 或 Token 返回 `INVALID_REQUEST`。

## 6. PostgreSQL 契约

已接受 ADR 不变量和机器可读 Migration 是 Storage 的权威来源。以下文字描述 Durable Resource Model；当具体物理字段或 Migration State 不一致时，以 `db/migrations/` 和 Implementation Contract SQL 为准。

### Tenant 与 Project

Tenant 与 Project 是安全边界。Tenant-owned Row 携带 `tenant_id`；Project Resource 在可行时同时携带 `tenant_id` 与 `project_id`。普通 Runtime DB Role 受 RLS 约束，不使用 Application-controlled RLS Bypass Flag。

### Task Aggregate

`tasks` 持有不可变安全身份与当前 Projection，至少包括：

```text
id
tenant_id
project_id
created_by
agent_id
input
status
version
trace_id
created_at
updated_at
completed_at
```

普通 Update 不能修改 Task Identity Field。`version` 通过 Aggregate Command Persistence 递增，用于阻止 Stale Writer。

### TaskEvent

`task_events` 是 Append-only Business History。核心字段包括：

```text
event_id / id
tenant_id
project_id
task_id
sequence
event_type
actor / evidence payload
trace_id（适用时）
occurred_at
schema version（适用时）
```

`(task_id, sequence)` 唯一；已经提交的 Business Event Sequence 必须连续。

### Transactional Outbox 与 Idempotency

当 Command Contract 要求时，R6 在一个 PostgreSQL Transaction 中同时持久化 Task Projection Change、TaskEvent、Outbox Delivery Intent 与 Command Idempotency Result。

Outbox Delivery 使用 At-least-once；Consumer 必须依赖稳定 Delivery Idempotency Key 去重。平台不宣称 Distributed Exactly-Once Transport。

### Routing、Cost、Audit、Evaluation、Tool 与 Admission

Route Decision、Cost Evidence、Audit Evidence、Evaluation Result、Tool Invocation、Model Admission Evidence 和 Model Registry Metadata 都是独立可查询的 Evidence/Control Plane Resource。其 Tenant/Project Attribution 必须遵守 Resource Scope Matrix，不能削弱 Task Ownership 或 RLS。

由 SQL 持有的金额使用 Exact/Numeric Representation；Cost Evidence 记录 Pricing Version，以保证历史成本可解释。

## 7. Row Level Security

Tenant-sensitive Runtime Table 按 Migration 启用并强制 RLS。Application Repository Predicate 仍然必须存在；RLS 是独立 Defense-in-Depth Boundary。

Runtime Application/Worker Role 不能是 Superuser，也不能拥有 `BYPASSRLS`。Migration/Admin Access 使用独立 Credential 与执行路径。

## 8. Durable Event Envelope

Business Event 是 Append-only，外部投递使用 Transactional Outbox。代表性的 Logical Envelope：

```json
{
  "event_id": "evt-...",
  "event_type": "TaskRoutingStarted",
  "schema_version": 1,
  "occurred_at": "...",
  "tenant_id": "tenant-...",
  "project_id": "project-...",
  "task_id": "task-...",
  "trace_id": "trace-...",
  "payload": {}
}
```

Physical Delivery 使用 At-least-once，Consumer 必须幂等。PostgreSQL Business State 与 TaskEvent History 和 Temporal Execution History 保持职责分离。

## 9. Migration 规则

Production SQL Migration 采用 Forward-oriented 策略。破坏性变更使用：

```text
Expand
-> Backfill/Migrate
-> Switch Reader/Writer
-> Contract
```

当旧 Writer 与新 Writer 无法安全并存时，Writer-contract Migration 必须定义明确的 Drain/Cutover Rule。Runtime 与 Migration DB Role 保持分离。

## 10. API 兼容规则

R7D 是 v0.1 清理旧 pre-S1 Draft 假设的 Contract Convergence Window。R7D 合并后，Executable OpenAPI 与 Runtime 必须同步修改。

新增 Optional Response Field 通常属于兼容变更；删除/重命名字段、改变 Required Request Field、改变 Enum 语义、改变 Idempotency Behavior 或改变 Security Scope 属于不兼容变更，必须进入显式 API Migration Decision，不能继续静默漂移。
