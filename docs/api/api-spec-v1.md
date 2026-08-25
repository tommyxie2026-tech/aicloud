# AI Cloud API Specification v1

## 1. API principles

- resource-oriented JSON APIs under `/api/v1`;
- asynchronous Task execution;
- stable machine-readable errors;
- idempotent creation and cancellation;
- optimistic concurrency for state-changing operations;
- polling first, with streaming events added without changing event semantics.

The currently implemented endpoints remain the compatibility baseline. Durable
Task additions are specified in [Task Workflow API v1](task-workflow-api-v1.md).

## 2. Common conventions

### Request headers

| Header | Required | Purpose |
|---|---:|---|
| `Content-Type: application/json` | writes | JSON payload |
| `Idempotency-Key` | recommended for Task creation | duplicate submission protection |
| `If-Match` | state-changing commands when exposed | expected resource version |
| `X-Request-ID` | optional | caller correlation |

### Error response

```json
{
  "error": {
    "code": "transition_conflict",
    "message": "task resource version changed",
    "retryable": true,
    "requestId": "req-01",
    "details": {
      "expectedVersion": 3,
      "actualVersion": 4
    }
  }
}
```

Provider responses, stack traces, secrets and raw credentials are never
returned in public errors.

## 3. Health and readiness

```http
GET /healthz
GET /readyz
```

`healthz` reports process liveness. `readyz` verifies required adapters for the
selected runtime mode. Temporal is not required when workflow mode is
`inprocess`.

## 4. Model API

```http
GET  /api/v1/models
POST /api/v1/models
GET  /api/v1/models/{id}
```

Model registration does not imply production approval. Routing applies model
lifecycle, version, capability, policy and health admission independently.

## 5. Agent API

```http
GET  /api/v1/agents
POST /api/v1/agents
GET  /api/v1/agents/{id}
```

Agent CRUD is planned. Agent resources reference governed model requirements,
Tool permissions, workflow definition and Sandbox profile.

## 6. Task API

```http
GET  /api/v1/tasks
POST /api/v1/tasks
GET  /api/v1/tasks/{id}
POST /api/v1/tasks/{id}/route
POST /api/v1/tasks/{id}/cancel
GET  /api/v1/tasks/{id}/events
GET  /api/v1/tasks/{id}/trace
GET  /api/v1/tasks/{id}/costs
```

`POST /api/v1/tasks/{id}/model` and the existing synchronous model execution
path remain compatibility endpoints during the durable workflow stage. They
are deprecated for new Task clients and are not called by the new workflow.

## 7. Workflow API

Workflow execution metadata is exposed through the Task resource. A separate
public workflow-definition CRUD API is deferred until multiple workflow types
are implemented.

## 8. Event ordering

Task events are ordered by monotonically increasing sequence within a Task.
Clients resume polling with `afterSequence` and must tolerate receiving an
already-seen event after reconnecting.

## 9. Security

All APIs will require identity, tenant context and policy evaluation. Until
OIDC is implemented, only trusted server/workload identity is represented as an
authenticated actor. Request payload fields cannot claim approver identity.

## 10. Compatibility policy

- additive fields are permitted within v1;
- state or error code removal requires a versioned migration;
- synchronous execution remains documented until a separate deprecation
  decision removes it;
- clients must ignore unknown JSON fields and handle unknown non-terminal event
  types conservatively.

## 11. Related documents

- [Task Workflow API v1](task-workflow-api-v1.md)
- [API and Data Model Design](../design/api-and-data-model-design.md)
- [Database Schema](../design/database-schema.md)
