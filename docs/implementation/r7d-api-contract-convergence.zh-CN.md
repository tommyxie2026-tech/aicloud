# R7D API 契约收敛

## 状态

实施阶段：R7D 可执行 OpenAPI 与运行时 API 收敛。

R7D 承接 R7A Authentication Boundary、R7B OIDC/JWT Verification 和 R7C RBAC/ABAC Authorization。目标是让机器可读的 v0.1 契约准确描述真正运行的 API，并让 CI 阻止两者再次漂移。

## 契约优先级

对于 `/api/v1`，已经接受的架构不变量继续保持最高约束。`docs/implementation/contracts/openapi-v1.yaml` 是可执行 HTTP 契约；运行时 Handler 与文字文档必须在同一个 PR 中与它同步收敛。

早期 pre-S1 OpenAPI Draft 中包含尚未实现的愿景路径和已经过时的 Task 状态枚举。R7D 会用 R6/R7 之后的实际运行契约替换这些内容，而不是强迫实现退回旧假设。

## JSON 约定

R7D 冻结 v0.1 公共 JSON 字段为 `camelCase`。这与已经运行的 R6 Command API、已持久化的 Idempotency Request Digest 以及公共 Domain JSON Tag 保持一致。内部 SQL 和 Event Schema 仍可继续使用 snake_case。

## v0.1 公共 Route Surface

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

Cancel、Approval、TaskEvent Query/Streaming、Agent/Policy/Project CRUD 在 Handler 真正实现之前仍属于后续产品 Slice，不能提前作为已经完成的 v0.1 公共 OpenAPI Path。

## Task 契约

Task 是 API 返回的 Aggregate Projection，并包含不可变的安全身份与乐观资源版本：

```json
{
  "id": "task-...",
  "tenantId": "tenant-...",
  "projectId": "project-...",
  "createdBy": "user-...",
  "agentId": "agent-...",
  "input": "...",
  "status": "CREATED",
  "version": 1,
  "traceId": "trace-...",
  "createdAt": "...",
  "updatedAt": "..."
}
```

Canonical Status Enum：`CREATED`、`PLANNING`、`ROUTING`、`EXECUTING`、`WAITING_APPROVAL`、`VALIDATING`、`COMPLETED`、`FAILED`、`CANCELLED`、`EXPIRED`。

`version` 是 PostgreSQL/Aggregate 的 Optimistic Revision，不是 Workflow Attempt Number。

## Task 创建与 Idempotency

`POST /api/v1/tasks` 使用已经验证的 Tenant/Project Principal Scope，不从 Body 接受 Tenant/Project。请求示例：

```json
{
  "input": "scale dev-gpu-cluster gpu-workers from 3 to 6",
  "agentId": "infra-agent"
}
```

未声明字段必须拒绝。R6 已经为 Task 创建、Task 路由和 Logical Model Execution 提供 Durable Command Idempotency，因此这些 Operation 必须携带 `Idempotency-Key`。相同 Key + 相同 Request 重放原结果；相同 Key + 不同 Request 返回 409。

其他 Control Plane Mutation 尚未进入 R6 Task Command Transaction Kernel，在独立 Command Contract 建立之前不能宣称 Durable Exactly-Once。

## Error 契约

所有 `/api/v1` Error Response 使用 R7A Envelope，包含稳定 `code`、`message`、`request_id`、`trace_id` 与 `retryable`。Request/Trace ID 在 Authentication 前产生，所以 401/403 与业务 Handler Failure 使用同一 Correlation Contract。

## Pagination

Collection 与 Evidence List Endpoint 使用有界分页：

- `pageSize`：默认 50，最小 1，最大 200；
- `pageToken`：上一个 Response 返回的 Opaque Continuation Token；
- Response：`{ "items": [...], "nextPageToken": "..." }`。

非法 Page Token 或 Page Size 返回 `INVALID_REQUEST`。

## Resource Version

Task Response 暴露 `version`。直接 GET Task 还返回：

```text
ETag: "task:<task-id>:v<version>"
X-Resource-Version: <version>
```

R7D 不会虚构 R6 Command Kernel 尚未实现的客户端状态迁移 Precondition；未来只有真正接入 Aggregate Expected-Version 时才引入 `If-Match`。

## 可执行契约门禁

CI 必须验证：

1. 每个 OpenAPI Path/Method 都映射到实际公共 API Operation；
2. 当前所有 Contracted Public Operation 都出现在 OpenAPI；
3. OpenAPI Task Status 与 Domain Aggregate 一致；
4. Mutating Task Command Path 声明 `Idempotency-Key`；
5. Closed Request Object 在 Runtime 拒绝未声明字段；
6. ErrorEnvelope 包含 Request/Trace Correlation Field；
7. Pagination 与 Task Resource-Version 行为具有可执行测试。

## 兼容性规则

在 v0.1 正式稳定前，R7D 是清理旧 Draft Contract 假设的迁移窗口。该收敛 PR 合并后，不兼容的 Field/Path/Enum 修改必须通过显式 API Migration Decision，不能继续静默漂移。
