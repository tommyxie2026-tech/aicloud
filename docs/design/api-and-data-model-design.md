# AI Cloud API and Data Model Design

## 1. Resource model

AI Cloud v0.1 exposes governed resources while keeping execution asynchronous.

| Resource | Purpose | v0.1 implementation state |
|---|---|---|
| Model | governed provider/version/capability asset | implemented baseline |
| Agent | model, workflow, Tool and Sandbox policy | domain/design baseline |
| Task | durable business execution request | synchronous baseline; durable extension next |
| Task Event | immutable transition/activity audit | next implementation stage |
| Tool | policy-bound external capability | interface baseline |
| Policy | deterministic admission/action decision | interface baseline |
| Workflow Run | execution metadata correlated to Task | next implementation stage |

## 2. Task resource

The durable Task is the primary asynchronous API resource.

```text
Task
  id
  tenant_id
  agent_id
  input
  status
  resource_version
  workflow_id
  workflow_run_id
  attempt
  trace_id
  result / error
  created_at / updated_at / completed_at
```

Task state is materialized for reads and backed by append-only Task events.
Every write uses optimistic concurrency. Task creation may use a scoped
idempotency key.

## 3. Task Event resource

```text
TaskEvent
  id
  task_id
  sequence
  event_type
  previous_state / next_state
  workflow_run_id
  trace_id
  activity_name
  attempt
  source_event_id
  actor
  error
  metadata
  occurred_at
```

The sequence is strictly increasing within one Task. Source Event ID protects
replayed Activities and duplicate delivery.

## 4. API behavior

```text
POST /api/v1/tasks                 -> 202 Accepted
GET  /api/v1/tasks/{id}            -> materialized state
GET  /api/v1/tasks/{id}/events     -> ordered event history
POST /api/v1/tasks/{id}/cancel     -> asynchronous idempotent cancellation
GET  /api/v1/tasks/{id}/trace      -> execution evidence
GET  /api/v1/tasks/{id}/costs      -> immutable cost facts
```

The existing synchronous model endpoint remains temporarily compatible and is
not used by the durable workflow.

## 5. Workflow and persistence model

```text
Task creation transaction
  -> Task(CREATED)
  -> CREATED event
  -> workflow start outbox

Outbox dispatcher
  -> Temporal Workflow ID = Task ID

Activity transition
  -> validate expected resource version
  -> append Task Event
  -> update materialized Task
  -> commit atomically
```

PostgreSQL is authoritative for business state and audit facts. Temporal is
authoritative for execution history, timers, retries and Signals.

## 6. Design principles

- resource-oriented public API;
- modular-monolith implementation;
- workflow engine hidden behind an internal adapter;
- explicit idempotency and optimistic concurrency;
- policy before Tool or Sandbox execution;
- immutable trace, evaluation and cost evidence;
- no authenticated actor identity from untrusted payload fields.

## 7. Authoritative specifications

- [API Specification v1](../api/api-spec-v1.md)
- [Durable Task Workflow API v1](../api/task-workflow-api-v1.md)
- [Database Schema](database-schema.md)
- [Task State Machine](agent-state-machine.md)
- [Durable Workflow Engineering Contract](durable-task-workflow-engineering-contract.md)
