# AI Cloud v0.1 Database Schema

> Database: PostgreSQL 16
> Purpose: business state, audit history, idempotency, workflow dispatch and cost facts

## 1. Persistence principles

- PostgreSQL is authoritative for materialized Task state and business events.
- Task events and Task status change in one transaction.
- Every mutable resource has an optimistic `resource_version`.
- External effects use immutable source IDs and unique constraints.
- Temporal stores execution history; PostgreSQL stores Workflow identity and
  business-visible execution metadata.
- JSONB is used for evolving payloads, not for identities, state or indexed
  control fields.

## 2. Existing resources

The existing `models`, `tasks`, `route_decisions`, `cost_events`, trace,
evaluation and admission tables remain in place. The durable workflow migration
extends those resources rather than replacing them.

## 3. Task extensions

Required logical columns on `tasks`:

| Column | Type | Constraint |
|---|---|---|
| `tenant_id` | text | non-null default scope until tenant enforcement |
| `status` | text | valid persisted state |
| `resource_version` | bigint | non-null, starts at 1 |
| `workflow_id` | text | unique, stable, equal to Task ID in v0.1 |
| `workflow_run_id` | text | nullable, latest Temporal Run ID |
| `attempt` | integer | non-null, starts at 1 |
| `idempotency_key` | text | nullable |
| `request_fingerprint` | text | nullable, detects key reuse with another payload |
| `trace_id` | text | non-null for workflow-backed Tasks |
| `cancel_requested_at` | timestamptz | nullable |
| `completed_at` | timestamptz | nullable |
| `updated_at` | timestamptz | non-null |

Required constraints:

```sql
UNIQUE (workflow_id)
UNIQUE NULLS NOT DISTINCT (tenant_id, idempotency_key)
CHECK (resource_version > 0)
CHECK (attempt > 0)
CHECK (status IN (
  'CREATED', 'PLANNING', 'EXECUTING', 'WAITING_APPROVAL',
  'VALIDATING', 'COMPLETED', 'FAILED', 'CANCELLED'
))
```

If the supported PostgreSQL version or migration tooling cannot use
`UNIQUE NULLS NOT DISTINCT`, use a partial unique index where
`idempotency_key IS NOT NULL`.

## 4. Task events

```sql
CREATE TABLE task_events (
  id                  text PRIMARY KEY,
  task_id             text NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  sequence            bigint NOT NULL,
  event_type          text NOT NULL,
  previous_state      text,
  next_state          text,
  workflow_run_id     text,
  trace_id            text NOT NULL,
  attempt             integer NOT NULL,
  activity_name       text,
  source_event_id     text NOT NULL,
  actor_type          text NOT NULL,
  actor_id            text NOT NULL,
  error_code          text,
  error_retryable     boolean,
  error_message       text,
  metadata            jsonb NOT NULL DEFAULT '{}'::jsonb,
  occurred_at         timestamptz NOT NULL,
  created_at          timestamptz NOT NULL DEFAULT now(),
  UNIQUE (task_id, sequence),
  UNIQUE (task_id, source_event_id),
  CHECK (sequence > 0),
  CHECK (attempt > 0)
);

CREATE INDEX task_events_task_time_idx
  ON task_events(task_id, occurred_at, id);
CREATE INDEX task_events_trace_idx
  ON task_events(trace_id);
```

Events are immutable. Corrections are represented by a later event, never an
`UPDATE` or `DELETE`.

## 5. Workflow start outbox

```sql
CREATE TABLE workflow_outbox (
  id                  text PRIMARY KEY,
  task_id             text NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  workflow_id         text NOT NULL,
  event_type          text NOT NULL,
  payload             jsonb NOT NULL,
  status              text NOT NULL DEFAULT 'pending',
  attempts            integer NOT NULL DEFAULT 0,
  available_at        timestamptz NOT NULL DEFAULT now(),
  leased_until        timestamptz,
  lease_owner         text,
  delivered_at        timestamptz,
  last_error_code     text,
  last_error_message  text,
  created_at          timestamptz NOT NULL DEFAULT now(),
  updated_at          timestamptz NOT NULL DEFAULT now(),
  UNIQUE (task_id, event_type),
  UNIQUE (workflow_id, event_type),
  CHECK (status IN ('pending', 'leased', 'delivered', 'dead_letter')),
  CHECK (attempts >= 0)
);

CREATE INDEX workflow_outbox_dispatch_idx
  ON workflow_outbox(status, available_at)
  WHERE status IN ('pending', 'leased');
```

Dispatchers lease rows with `FOR UPDATE SKIP LOCKED`. An expired lease becomes
eligible again. Starting an already-running stable Workflow ID is treated as a
successful delivery.

## 6. Cost idempotency

The existing `cost_events` table must add:

| Column | Purpose |
|---|---|
| `source_event_id` | stable model/tool/sandbox operation identity |
| `workflow_run_id` | execution correlation |
| `activity_id` | logical Temporal Activity identity |
| `attempt` | billed attempt |

Required uniqueness:

```sql
CREATE UNIQUE INDEX cost_events_task_source_idx
  ON cost_events(task_id, source_event_id);
```

On duplicate source ID with an identical payload, the repository returns the
existing Cost Event. A different payload is an idempotency conflict.

## 7. Transaction contracts

### 7.1 Create Task

One transaction:

1. check scoped idempotency key;
2. insert Task in `CREATED` with `resource_version = 1`;
3. insert sequence 1 `CREATED` event;
4. insert `workflow_start_requested` outbox event;
5. commit.

If the idempotency key exists with the same request fingerprint, return the
existing Task. If the fingerprint differs, return `idempotency_conflict`.

### 7.2 Transition Task

Conceptual SQL sequence:

```sql
BEGIN;

SELECT status, resource_version
FROM tasks
WHERE id = $1
FOR UPDATE;

-- Validate current state and expected resource version in repository code.

INSERT INTO task_events (..., sequence, source_event_id, ...)
VALUES (..., next_sequence, ..., ...);

UPDATE tasks
SET status = $next_state,
    resource_version = resource_version + 1,
    attempt = $attempt,
    workflow_run_id = COALESCE($workflow_run_id, workflow_run_id),
    updated_at = now()
WHERE id = $task_id
  AND resource_version = $expected_version;

COMMIT;
```

The update must affect exactly one row. Any error rolls back the event append.

## 8. Repository interfaces

```go
type TaskRepository interface {
    List(context.Context) ([]domain.Task, error)
    Get(context.Context, string) (domain.Task, error)
    CreateWithWorkflowRequest(
        context.Context,
        domain.CreateTaskCommand,
    ) (domain.Task, error)
    TransitionTask(
        context.Context,
        domain.TransitionTaskCommand,
    ) (domain.Task, error)
}

type TaskEventRepository interface {
    ListAfter(
        context.Context,
        string,
        int64,
        int,
    ) ([]domain.TaskEvent, error)
}

type WorkflowOutboxRepository interface {
    Lease(context.Context, string, int, time.Duration) ([]domain.OutboxEvent, error)
    MarkDelivered(context.Context, string, string) error
    Reschedule(context.Context, string, domain.RetryDecision) error
}
```

The memory implementation must preserve transaction, version-conflict and
deduplication semantics; it is not allowed to silently accept behavior that the
PostgreSQL adapter rejects.

## 9. Migration sequence

1. Add nullable workflow/idempotency fields and non-breaking defaults.
2. Backfill existing Tasks from current statuses into the new state model.
3. Add Task events and outbox tables.
4. Backfill one import event for pre-existing Tasks when audit continuity is
   required.
5. Add indexes and uniqueness constraints.
6. Switch writes to transactional repository methods.
7. Make required columns non-null after compatibility verification.

Migrations remain forward-only. Rollback is performed by application
compatibility and a corrective migration, not by deleting durable events.

## 10. Verification

- migration applies to an empty and populated database;
- duplicate scoped idempotency keys are deterministic;
- concurrent transitions produce one winner;
- forced failure after event insert rolls back both writes;
- outbox lease recovery works after dispatcher termination;
- duplicate cost source events do not increase aggregate cost;
- event replay matches materialized Task status, version and attempt.

## 11. Related documents

- [Durable Workflow Engineering Contract](durable-task-workflow-engineering-contract.md)
- [Task State Machine](agent-state-machine.md)
- [Task Workflow API v1](../api/task-workflow-api-v1.md)
