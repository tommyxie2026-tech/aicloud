# AI Cloud Task Event and Outbox Contract

> Status: S0 Contract Freeze

## 1. Purpose

Define the immutable business-event history for Task execution and the transactional outbox used to deliver external signals without unsafe dual writes.

## 2. TaskEvent Contract

```yaml
task_event:
  event_id: string
  tenant_id: string
  project_id: string
  task_id: string
  sequence: int64
  event_type: string
  actor:
    principal_type: string
    subject_id: string
  payload: object
  request_id: string?
  trace_id: string
  schema_version: int
  occurred_at: timestamp
  created_at: timestamp
```

Required uniqueness:

```text
PRIMARY KEY(event_id)
UNIQUE(task_id, sequence)
```

## 3. Event Semantics

TaskEvent is append-only business history.

Forbidden:

```text
UPDATE task_events
DELETE task_events
rewrite past payload to match current code
```

Schema evolution uses new `schema_version` and compatible readers/upcasters where required.

## 4. Canonical Event Families

State events:

```text
TaskCreated
TaskPlanningStarted
TaskRoutingStarted
TaskExecutionStarted
TaskApprovalRequested
TaskApprovalGranted
TaskApprovalRejected
TaskValidationStarted
TaskCompleted
TaskFailed
TaskCancelled
TaskExpired
```

Evidence/linkage events may include:

```text
RouteDecisionRecorded
ModelAttemptStarted
ModelAttemptCompleted
ModelAttemptFailed
PolicyDecisionRecorded
ToolInvocationRequested
ToolInvocationCompleted
ToolInvocationFailed
EvaluationCompleted
CostReconciled
```

Not every low-level telemetry span is a TaskEvent. TaskEvent records business-significant facts; OpenTelemetry records detailed execution telemetry.

## 5. Ordering

`sequence` is monotonically increasing per Task. The database transaction that changes Task business state allocates the next sequence.

Cross-task global ordering is not required.

Consumers must not use timestamps as the sole ordering mechanism.

## 6. Atomicity with Task Projection

Every Task state mutation and its canonical event commit atomically:

```text
BEGIN
  SELECT task FOR UPDATE / validate version
  UPDATE task projection
  INSERT task_event(sequence = previous + 1)
  INSERT outbox message if delivery required
COMMIT
```

A Task state change without its corresponding event is invalid.

## 7. Outbox Contract

The outbox prevents unsafe dual writes such as:

```text
commit PostgreSQL
then signal Temporal
```

or:

```text
publish event
then fail DB commit
```

Canonical outbox record:

```yaml
outbox:
  outbox_id: string
  tenant_id: string
  project_id: string
  task_id: string?
  aggregate_type: string
  aggregate_id: string
  event_type: string
  payload: object
  destination: string
  idempotency_key: string
  status: pending | delivering | delivered | dead_letter
  attempts: int
  available_at: timestamp
  created_at: timestamp
  delivered_at: timestamp?
```

## 8. Dispatcher Semantics

Outbox delivery is at-least-once. Therefore downstream consumers must be idempotent.

```text
DB Commit
  -> Outbox Pending
  -> Dispatcher
  -> Temporal Signal / Event Bus / Webhook
  -> mark Delivered
```

The dispatcher uses bounded retry with backoff and a dead-letter state. Delivery status changes do not mutate TaskEvent history.

## 9. Idempotency

Each external message has a stable idempotency key. Consumers deduplicate using a durable processed-message store or an equivalent native idempotency mechanism.

Task command idempotency and outbox delivery idempotency are distinct concerns and both are required.

## 10. Consumer Rules

Consumers must:

- validate schema version;
- validate tenant/project/task identity;
- reject impossible cross-scope references;
- handle duplicate delivery safely;
- not mutate historical TaskEvent records;
- persist side effects before acknowledging delivery.

## 11. Retention

TaskEvent is long-lived audit/business evidence. Retention follows tenant/legal policy but deletion, if legally required, must use a governed evidence-retention process rather than ordinary application DELETE APIs.

Outbox delivered rows may be compacted after a defined retention period, provided delivery evidence is retained elsewhere.

## 12. Acceptance Criteria

- Every Task state transition creates exactly one canonical state event.
- `(task_id, sequence)` is unique and gap behavior is understood under rollback.
- Event history cannot be updated through application repositories.
- Task projection, TaskEvent and Outbox record commit atomically.
- Dispatcher retries do not duplicate downstream side effects.
- Duplicate delivery tests pass.
- Schema-version compatibility tests pass.
- Tenant/project identity is preserved in all events and messages.

## 13. Implementation Impact

S2 defines DB/API contracts for TaskEvent and command idempotency. S3 uses Outbox and durable workflow adapters to coordinate Temporal without making Temporal the business event store.