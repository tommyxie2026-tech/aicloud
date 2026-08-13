# API and Data Contracts

## 1. API conventions

Base path is `/api/v1`. Public JSON uses `camelCase`; internal SQL and durable-event schemas may use `snake_case`. Timestamps are RFC3339 UTC. IDs are opaque strings.

Authentication produces a verified `identity.Principal`. Public handlers never accept Tenant/Project security scope as a substitute for authenticated context. In OIDC mode, Tenant/Project/Role/Capability values originate only from cryptographically verified claims. Trusted-ingress mode is an explicit compatibility mode and requires a separately authenticated ingress that replaces identity headers.

Request and trace correlation are established before authentication:

```text
X-Request-ID
X-Trace-ID
```

Invalid or missing correlation IDs are replaced by the API boundary and echoed in the response.

Durable `Idempotency-Key` semantics currently apply to the R6 Task command kernel only:

- Task creation;
- Task routing;
- logical model execution.

For those operations, same key + same canonical request replays the original business result; same key + different request returns HTTP 409. Other mutations must not be described as durably exactly-once until their own command transaction exists.

## 2. Error envelope

All `/api/v1` errors use:

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

Core boundary codes include `INVALID_REQUEST`, `UNAUTHENTICATED`, `FORBIDDEN`, `NOT_FOUND`, `CONFLICT`, `METHOD_NOT_ALLOWED`, `RATE_LIMITED`, `SERVICE_UNAVAILABLE`, `INTERNAL_ERROR`, `IDEMPOTENCY_CONFLICT`, and `IDEMPOTENCY_IN_PROGRESS`. Domain-specific layers may add stable codes without changing the envelope.

## 3. Executable v0.1 public REST surface

The machine-readable source of truth is `docs/implementation/contracts/openapi-v1.yaml`. R7D documents only operations implemented by the running API and governed by the R7 security boundary:

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

Cancel, approval, TaskEvent query/streaming, Agent CRUD, Policy CRUD, Project CRUD, and global audit query remain architecture or later-product contracts until executable handlers exist. They must not be advertised as completed v0.1 HTTP operations.

## 4. Task HTTP contract

Task creation consumes verified Tenant/Project scope and requires `Idempotency-Key`:

```json
{
  "input": "scale dev-gpu-cluster gpu-workers from 3 to 6",
  "agentId": "infra-agent"
}
```

Unknown top-level fields are rejected. Tenant/Project are not accepted from the body.

Response HTTP 202 returns the aggregate projection, for example:

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

Canonical Task status values are:

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

Direct `GET /tasks/{task_id}` also returns:

```text
ETag: "task:<task-id>:v<version>"
X-Resource-Version: <version>
```

`version` is the aggregate/PostgreSQL optimistic revision. R7D does not introduce `If-Match` until a public command is explicitly wired to expected-version semantics.

## 5. Pagination

List and evidence-list endpoints use bounded pagination:

- `pageSize`: default 50, minimum 1, maximum 200;
- `pageToken`: opaque continuation token returned by the previous response;
- response: `{ "items": [...], "nextPageToken": "..." }`.

Clients must treat `pageToken` as opaque. Invalid sizes or tokens return `INVALID_REQUEST`.

## 6. PostgreSQL contract

Accepted ADR invariants and machine-readable migrations are authoritative for storage. Prose below describes the durable resource model; exact physical columns and migration state must be read from `db/migrations/` and implementation contract SQL when they differ.

### Tenants and projects

Tenant and Project are security boundaries. Tenant-owned rows carry `tenant_id`; Project resources carry both `tenant_id` and `project_id` where practical. Normal runtime database roles are subject to RLS and do not use application-controlled RLS bypass flags.

### Task aggregate

`tasks` owns immutable security identity and the current projection, including at minimum:

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

Task identity fields are not mutable through ordinary updates. `version` increments through aggregate command persistence and protects against stale writers.

### TaskEvent

`task_events` is append-only business history. Core fields include:

```text
event_id / id
tenant_id
project_id
task_id
sequence
event_type
actor / evidence payload
trace_id where applicable
occurred_at
schema version where applicable
```

`(task_id, sequence)` is unique and sequences are contiguous for committed business events.

### Transactional Outbox and idempotency

R6 persists Task projection changes, TaskEvent, Outbox delivery intent, and command idempotency result in one PostgreSQL transaction where the command contract requires them.

Outbox delivery is at-least-once. Consumers must deduplicate using a stable delivery idempotency key. The platform does not claim distributed exactly-once transport.

### Routing, cost, audit, evaluation, tools and admission

Route decisions, cost evidence, audit evidence, evaluation results, Tool invocations, model admission evidence, and model registry metadata remain independently queryable evidence/control-plane resources. Their Tenant/Project attribution must follow the Resource Scope Matrix and cannot weaken Task ownership or RLS.

Monetary persistence uses exact/numeric representation where SQL owns the amount; cost evidence records pricing version so historical cost remains explainable.

## 7. Row Level Security

RLS is enabled/forced on tenant-sensitive runtime tables according to migrations. Application repository predicates remain mandatory; RLS is an independent defense-in-depth boundary.

Runtime application/worker roles must not be superuser or `BYPASSRLS`. Migration/admin access uses separate credentials and execution paths.

## 8. Durable event envelope

Business events are append-only and external delivery uses the transactional Outbox. A representative logical envelope is:

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

Physical delivery is at-least-once; consumers are idempotent. PostgreSQL business state and TaskEvent history remain independent from Temporal execution history.

## 9. Migration rules

Production SQL migrations are forward-oriented. Destructive changes use:

```text
Expand
-> Backfill/Migrate
-> Switch Reader/Writer
-> Contract
```

Writer-contract migrations require explicit drain/cutover rules when old and new writers cannot safely coexist. Runtime and migration database roles remain separate.

## 10. API compatibility

R7D is the v0.1 convergence window that removes stale pre-S1 draft assumptions. After R7D merges, the executable OpenAPI contract and runtime must change together.

Adding optional response fields is normally compatible. Removing or renaming fields, changing required request fields, changing enum semantics, changing idempotency behavior, or changing security scope is incompatible and requires an explicit API migration decision rather than silent drift.
