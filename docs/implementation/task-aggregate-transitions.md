# R5 Task Aggregate Transitions Implementation

## Status

Implemented on `agent/task-aggregate-transitions`, pending CI and review.

## Goal

Turn Task from a mutable status record into the execution aggregate defined by the S0 contract. R5 establishes validated lifecycle transitions and optimistic concurrency before R6 adds TaskEvent, Transactional Outbox and command idempotency.

## Canonical lifecycle

```text
CREATED
   ↓
PLANNING
   ↓
ROUTING
   ↓
EXECUTING
   ├──────────────┐
   ▼              │
WAITING_APPROVAL  │
   │              │
   └──────► EXECUTING
                    ↓
                VALIDATING
                    ↓
                COMPLETED
```

Any known non-terminal state may terminate as `FAILED`, `CANCELLED` or `EXPIRED`. Terminal states cannot return to non-terminal states.

## Aggregate API

Task lifecycle changes must use:

```go
transition, err := task.Transition(domain.TaskTransitionCommand{
    To:    domain.TaskRouting,
    Actor: "service_account:worker-1",
    Cause: "request model route",
    At:    now,
})
```

The transition validates source/target state and requires actor, cause and time evidence. Direct runtime assignment to `task.Status` is forbidden.

`domain.NewTask` initializes a new aggregate in `CREATED` with `version=1`.

## Optimistic concurrency

Task repositories use the Task `version` as the expected resource version.

```text
read Task(version=N)
      ↓
mutate aggregate
      ↓
Update(expected=N)
      ↓
version=N+1
```

A stale write returns `repository.ErrVersionConflict`. Callers must reload current state and re-evaluate the command; blind retries are forbidden.

The in-memory repository and `ScopedPostgresTasks` implement the same contract.

## PostgreSQL persistence

Migration `006_task_aggregate_state.sql`:

- adds `version BIGINT NOT NULL DEFAULT 1`;
- adds nullable `completed_at`;
- maps prototype `PENDING -> CREATED`;
- maps prototype `RUNNING -> EXECUTING`;
- backfills terminal `completed_at`;
- constrains version to positive values;
- constrains status to the canonical lifecycle;
- adds a scope/status/created index.

PostgreSQL updates use a version predicate and increment:

```sql
UPDATE tasks
SET ..., version = version + 1
WHERE id = $1 AND version = $expected
RETURNING version;
```

## Control-plane convergence

### Create

`CreateTask` uses `domain.NewTask`; persisted Task begins as `CREATED` version 1.

### Route

`DecideRoute` advances:

```text
CREATED -> PLANNING -> ROUTING
```

A retry may remain in `ROUTING`. Other source states are rejected.

### Model execution

`ExecuteModel` advances:

```text
ROUTING -> EXECUTING
```

Successful execution:

```text
EXECUTING -> VALIDATING -> COMPLETED
```

Provider failure:

```text
EXECUTING -> FAILED
```

R5 emits trace evidence for transitions. R6 will make TaskEvent the durable business-history record and persist state/event/outbox atomically.

## Failure semantics

- invalid lifecycle jump -> `domain.ErrInvalidTaskTransition`;
- transition from terminal state -> `domain.ErrTaskTerminal`;
- stale repository write -> `repository.ErrVersionConflict`;
- route/model operations from an incompatible state fail before the corresponding controlled operation proceeds.

## Deliberate R5 boundary

R5 does not claim atomic Task + TaskEvent persistence. It deliberately leaves the following to R6:

```text
Task projection
+
TaskEvent
+
Outbox
+
Idempotency record
```

That next slice will turn the transition result produced here into append-only business evidence within one transaction.

## Tests

Required coverage:

- valid lifecycle path;
- approval loop;
- state skipping rejection;
- terminal state irreversibility;
- fail/cancel/expire from all non-terminal states;
- transition metadata requirements;
- in-memory stale-write rejection;
- PostgreSQL stale-write rejection;
- migration mapping and constraints;
- existing tenant/RLS tests remain green.

## Definition of Done

R5 is complete when:

- runtime control-plane code does not directly assign Task status;
- Task creation is canonical `CREATED/version=1`;
- state transitions are deterministic;
- terminal states cannot reopen;
- memory and PostgreSQL repositories enforce version conflicts;
- migration 006 passes real PostgreSQL integration tests;
- bilingual documentation, `gofmt`, unit tests, integration tests, `go vet` and entrypoint builds pass.
