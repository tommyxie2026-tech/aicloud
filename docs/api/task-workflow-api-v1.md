# Durable Task Workflow API v1

> Status: implementation contract; endpoints are not complete until code and
> OpenAPI tests land.

## 1. Create a Task

```http
POST /api/v1/tasks
Idempotency-Key: issue-123-fix-v1
Content-Type: application/json
```

```json
{
  "agentId": "developer-agent-v1",
  "input": "Fix issue 123",
  "metadata": {
    "repository": "fixture/example"
  }
}
```

Response:

```http
HTTP/1.1 202 Accepted
Location: /api/v1/tasks/task-01
```

```json
{
  "id": "task-01",
  "agentId": "developer-agent-v1",
  "status": "CREATED",
  "resourceVersion": 1,
  "workflowId": "task-01",
  "workflowRunId": null,
  "attempt": 1,
  "traceId": "trace-01",
  "links": {
    "self": "/api/v1/tasks/task-01",
    "events": "/api/v1/tasks/task-01/events",
    "cancel": "/api/v1/tasks/task-01/cancel"
  }
}
```

The response does not wait for Temporal. The Task, initial event and workflow
outbox request are already committed. Reusing the same scoped idempotency key
and payload returns the original Task. Reusing the key with another payload
returns `409 idempotency_conflict`.

## 2. Get a Task

```http
GET /api/v1/tasks/{taskId}
```

Returns current materialized state, resource version, attempt, run identity,
result/error summary and links. It does not return the complete event history.

## 3. List Task events

```http
GET /api/v1/tasks/{taskId}/events?afterSequence=12&limit=100
```

```json
{
  "items": [
    {
      "id": "event-13",
      "sequence": 13,
      "type": "task.transitioned",
      "previousState": "EXECUTING",
      "nextState": "VALIDATING",
      "workflowRunId": "run-01",
      "traceId": "trace-01",
      "attempt": 1,
      "activity": "ValidateResult",
      "occurredAt": "2026-08-25T10:00:00Z"
    }
  ],
  "nextAfterSequence": 13
}
```

`limit` defaults to 100 and is capped by server configuration. Ordering is
strictly ascending by sequence.

## 4. Cancel a Task

```http
POST /api/v1/tasks/{taskId}/cancel
If-Match: "4"
Content-Type: application/json
```

```json
{
  "reason": "request withdrawn"
}
```

Accepted cancellation returns `202 Accepted` while cleanup is in progress.
Cancellation of an already `CANCELLED` Task returns `200 OK` with the current
Task. Cancellation of `COMPLETED` returns `409 transition_conflict`.

The actor comes from trusted request identity, never from the JSON body.

## 5. Trace and cost views

```http
GET /api/v1/tasks/{taskId}/trace
GET /api/v1/tasks/{taskId}/costs
```

The trace view links workflow, model, policy, tool, sandbox and evaluation
events. Cost results include attempts and fallback while deduplicating repeated
delivery by immutable source event ID.

## 6. Status and error mapping

| Situation | Response |
|---|---|
| accepted asynchronous creation | `202` |
| repeated identical idempotent creation | `200` or original `202` representation |
| malformed input | `400 invalid_argument` |
| unauthenticated/unauthorized | `401` / `403` |
| Task not found | `404 not_found` |
| resource-version conflict | `409 transition_conflict` |
| idempotency payload mismatch | `409 idempotency_conflict` |
| workflow backend unavailable after Task commit | Task remains `CREATED`; dispatcher retries |

## 7. OpenAPI implementation requirement

The implementation must add an OpenAPI document and contract tests that verify:

- status codes and `Location` header;
- idempotency and conflict responses;
- all persisted Task states and error codes;
- event pagination/order;
- cancellation terminal-state behavior;
- backward compatibility of existing model execution endpoints.

## 8. Related documents

- [API Specification v1](api-spec-v1.md)
- [Task State Machine](../design/agent-state-machine.md)
- [Database Schema](../design/database-schema.md)
- [Engineering Contract](../design/durable-task-workflow-engineering-contract.md)
