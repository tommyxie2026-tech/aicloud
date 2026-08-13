# R6 Task Transaction Kernel: TaskEvent, Outbox and Idempotency

> Status: implementation design for Issue #23

## 1. Purpose

R5 established the Task Aggregate state machine and optimistic concurrency, but Task projection updates and adjacent durable records can still be written independently. R6 closes that boundary.

The invariant is:

```text
Task business mutation
    = Task projection update
    + exactly one canonical TaskEvent
    + required Outbox delivery intent
    + command Idempotency result

all inside one PostgreSQL transaction.
```

A partial commit is invalid.

## 2. Why this exists

The unsafe R5 edge is a classic dual-write boundary:

```text
write RouteDecision / external intent
        |
        v
update Task projection
        |
        +-- version conflict / process crash
```

R6 changes the system from "several related writes" into a business transaction kernel.

## 3. Runtime architecture

```text
                    Public Command
                         |
                         v
                Idempotency Gate
                         |
                         v
              Task Command Handler
                         |
                         v
+--------------------------------------------------+
|          PostgreSQL Business Transaction         |
|                                                  |
|  lock/validate Task version                      |
|        |                                         |
|        v                                         |
|  apply Task transition                           |
|        |                                         |
|        +--> append TaskEvent(sequence=N+1)       |
|        |                                         |
|        +--> append Outbox message if required    |
|        |                                         |
|        +--> persist Idempotency result           |
|                                                  |
+--------------------------+-----------------------+
                           |
                         COMMIT
                           |
                           v
                    Outbox Dispatcher
                           |
             +-------------+-------------+
             |             |             |
          Temporal      Event Bus      Webhook
```

The dispatcher is deliberately outside the business transaction. Delivery is at-least-once; the durable Outbox record makes delivery intent recoverable after process failure.

## 4. TaskEvent

TaskEvent is immutable business history, not telemetry.

Canonical state events are derived from Task lifecycle states:

```text
CREATED             -> TaskCreated
PLANNING            -> TaskPlanningStarted
ROUTING             -> TaskRoutingStarted
EXECUTING           -> TaskExecutionStarted
WAITING_APPROVAL    -> TaskApprovalRequested
VALIDATING          -> TaskValidationStarted
COMPLETED           -> TaskCompleted
FAILED              -> TaskFailed
CANCELLED           -> TaskCancelled
EXPIRED             -> TaskExpired
```

TaskEvent ordering is per Task:

```text
UNIQUE(task_id, sequence)
sequence >= 1
```

The next sequence is allocated while the Task row is locked/validated in the same transaction. Timestamps are evidence, not the ordering primitive.

Runtime RLS exposes SELECT and INSERT policies only. UPDATE and DELETE are intentionally unavailable to ordinary runtime paths.

## 5. Transactional Outbox

Outbox records external delivery intent in the same transaction as the business mutation.

```text
DB commit
  -> outbox pending
  -> dispatcher lease
  -> deliver
  -> delivered
```

Allowed states:

```text
pending -> delivering -> delivered
                    \-> pending       (bounded retry)
                    \-> dead_letter   (retry exhausted / terminal delivery error)
```

Each message has a durable delivery idempotency key. Downstream consumers remain responsible for deduplication because transport semantics are at-least-once.

## 6. Command idempotency

Public mutation scope is:

```text
tenant_id + project_id + operation + idempotency_key
```

Request behavior:

```text
same key + same request digest
    -> replay the same logical result

same key + different request digest
    -> IDEMPOTENCY_CONFLICT
```

The idempotency record is part of the same database transaction as the Task mutation. A request cannot commit an idempotency success record while losing the business mutation, or vice versa.

## 7. Migration 007

Migration `007_task_event_outbox_idempotency.sql` creates:

```text
task_events
outbox_messages
idempotency_records
```

All three carry tenant/project scope and have PostgreSQL RLS enabled and forced.

Important database invariants include:

- `task_events(event_id)` primary key;
- `UNIQUE(task_id, sequence)`;
- positive event sequence and schema version;
- no runtime UPDATE/DELETE TaskEvent policy;
- stable outbox delivery idempotency uniqueness;
- bounded outbox status vocabulary;
- command idempotency primary key on tenant/project/operation/key;
- idempotency expiry must be later than creation time.

## 8. Atomic repository boundary

The production repository API added in R6 must not expose a sequence such as:

```text
UpdateTask()
AppendEvent()
AppendOutbox()
SaveIdempotency()
```

as four independently committable calls for one business command.

Instead it exposes one transaction-level operation conceptually equivalent to:

```text
CommitTaskCommand(command)
```

The implementation owns one SQL transaction and either commits every required record or none.

## 9. Concurrency

R5 optimistic concurrency remains valid and becomes one validation inside the R6 transaction.

Recommended transaction flow:

```text
BEGIN
  resolve/reserve idempotency key
  SELECT Task ... FOR UPDATE
  validate expected Task version
  validate transition
  allocate next TaskEvent sequence
  UPDATE Task projection + version
  INSERT TaskEvent
  INSERT Outbox message(s)
  complete Idempotency record
COMMIT
```

No blind retry is permitted after a Task version conflict. The caller reloads current state and re-evaluates business intent.

## 10. Crash semantics

R6 is designed around explicit crash boundaries.

If the process dies before COMMIT, none of the Task mutation, event, outbox or idempotency result is durable.

If it dies after COMMIT but before external delivery, the Outbox row remains pending and a dispatcher can resume delivery.

If the dispatcher dies after the downstream system accepts the message but before marking delivered, the message may be delivered again; the stable delivery idempotency key prevents duplicate side effects in a compliant consumer.

## 11. Security and tenancy

Every event, outbox message and idempotency record carries `tenant_id` and `project_id`.

Runtime repositories must set transaction-local PostgreSQL scope before accessing these tables. Cross-project references are rejected even if identifiers are guessed.

System dispatchers must not gain an application-controlled RLS bypass. Cross-project work is performed through explicit scoped processing or a separately reviewed administrative/worker design.

## 12. Evidence boundary

TaskEvent and OpenTelemetry serve different purposes:

```text
TaskEvent      = durable business-significant facts
Trace/Span     = detailed execution telemetry
Audit          = security/authorization evidence
Outbox         = durable external delivery intent
```

Do not copy every trace span into TaskEvent.

## 13. Implementation slices

R6 is implemented in four slices:

1. schema + domain contracts;
2. atomic PostgreSQL Task command repository;
3. control-plane idempotency/transition convergence;
4. Outbox dispatcher lease/retry/dead-letter behavior and crash tests.

The first slice is complete when domain validation, migration contract tests and real PostgreSQL RLS/uniqueness tests pass.

## 14. Definition of Done

R6 is complete only when:

- every Task state mutation appends exactly one canonical state event;
- Task projection and TaskEvent cannot commit independently;
- required Outbox intent is atomic with the business mutation;
- command idempotency is durable and transactionally coupled;
- duplicate identical commands execute once;
- changed requests under the same key conflict;
- TaskEvent ordering is deterministic under concurrency;
- external delivery survives process crashes;
- duplicate delivery is safe;
- tenant/project RLS tests pass;
- unit, integration, vet and build gates pass.
