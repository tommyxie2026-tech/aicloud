# R7D API Contract Convergence

## Status

Implementation slice: R7D executable OpenAPI and runtime API convergence.

R7D follows R7A authentication boundary, R7B OIDC/JWT verification, and R7C RBAC/ABAC authorization. Its purpose is to make the machine-readable v0.1 contract describe the API that actually runs, and to make CI reject drift between the two.

## Contract precedence

For `/api/v1`, the accepted architecture invariants remain authoritative. `docs/implementation/contracts/openapi-v1.yaml` is the executable HTTP contract. Runtime handlers and prose documentation must converge to it in the same PR.

The pre-S1 OpenAPI draft contained aspirational paths and an obsolete Task state enum. R7D intentionally replaces those parts with the post-R6/R7 runtime contract rather than forcing the implementation back to stale assumptions.

## JSON convention

R7D freezes public JSON field names as `camelCase` for v0.1.

This matches the already-running R6 command API, persisted idempotency request digests, and public domain JSON tags. Changing the API to snake_case at this point would create avoidable client and replay incompatibility. Internal SQL and event schemas may continue to use snake_case.

## Public v0.1 route surface

The executable contract documents only routes that are implemented and governed by the R7 security boundary:

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

Cancel, approval, TaskEvent query/streaming, Agent CRUD, Policy CRUD, and Project CRUD remain architectural contracts or later product slices until executable handlers exist. They must not appear as completed v0.1 public OpenAPI paths before implementation.

## Task contract

Task is the aggregate projection returned by the API. The v0.1 HTTP representation contains the immutable security identity and optimistic resource version:

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

The canonical status enum is:

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

`version` is the PostgreSQL/aggregate optimistic revision. It is not a workflow attempt number.

## Task creation

`POST /api/v1/tasks` requires `Idempotency-Key` and consumes the verified Tenant/Project Principal scope. Tenant and Project are not accepted from the body because the caller must not select a security boundary independently from authenticated context.

```json
{
  "input": "scale dev-gpu-cluster gpu-workers from 3 to 6",
  "agentId": "infra-agent"
}
```

Unknown request fields are rejected. Same key + same canonical request returns the original Task and sets `Idempotency-Replayed: true`; same key + different request returns `409`.

## Idempotency boundary

R6 provides durable command idempotency for:

- Task creation;
- Task routing;
- logical model execution.

The OpenAPI contract marks `Idempotency-Key` required on those operations. Other control-plane mutations remain outside the R6 Task command transaction kernel and must not be described as exactly-once or durably idempotent until their own command contracts exist.

## Error contract

Every `/api/v1` error response uses the R7A envelope:

```json
{
  "error": {
    "code": "INVALID_REQUEST",
    "message": "...",
    "request_id": "req-...",
    "trace_id": "trace-...",
    "retryable": false,
    "details": {}
  }
}
```

Request and trace IDs are produced before authentication, so 401/403 responses and handler failures share the same correlation contract.

## Pagination

Collection and evidence-list endpoints use bounded pagination:

- `pageSize`: default 50, minimum 1, maximum 200;
- `pageToken`: opaque continuation token returned by the previous response;
- response: `{ "items": [...], "nextPageToken": "..." }`.

Invalid page tokens or sizes return `INVALID_REQUEST`. Tokens are opaque API values and callers must not construct them.

## Resource version

Task responses expose `version`. Direct Task GET additionally returns:

```text
ETag: "task:<task-id>:v<version>"
X-Resource-Version: <version>
```

R7D does not invent client-side state-transition preconditions that the R6 command kernel does not implement. A future command may add `If-Match` only when it is wired to the aggregate's expected-version transaction semantics.

## Executable contract gate

CI must verify:

1. every OpenAPI path/method maps to a running public API operation;
2. every currently contracted public operation appears in OpenAPI;
3. Task status values in OpenAPI match the domain aggregate;
4. mutating Task command paths declare `Idempotency-Key`;
5. request objects that are closed in OpenAPI reject undocumented fields at runtime;
6. ErrorEnvelope responses contain request/trace correlation fields;
7. pagination and Task resource-version behavior are executable tests.

## Compatibility rule

Before v0.1 is declared stable, R7D is the migration window that removes stale draft contract assumptions. After this convergence merge, incompatible field/path/enum changes require an explicit API migration decision instead of silent drift.
