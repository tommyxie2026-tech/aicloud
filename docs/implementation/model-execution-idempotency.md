# R6 Model Execution Idempotency and Lifecycle Boundary

> Status: R6 implementation contract

## 1. Goal

Treat a public model execution request as one durable logical command while allowing the provider runtime to perform one or more physical attempts. The design must not claim globally exactly-once provider transport.

```text
Public model command
  = stable logical operation identity
  + durable command idempotency
  + canonical Task lifecycle events
  + one or more physical model attempts
```

## 2. Public API contract

```text
POST /api/v1/tasks/{task_id}/model
Idempotency-Key: <stable logical command key>
```

The canonical request digest includes the Task identity and all business-affecting provider request fields. `requestId` is excluded from the digest because it is transport/logical-call metadata and the Control Plane replaces it with a stable logical model operation ID.

The command idempotency scope remains:

```text
tenant_id + project_id + operation + idempotency_key
```

## 3. Logical operation versus physical attempts

A retry or provider fallback does not create a new public business command.

```text
Logical Model Operation
        |
        +-- Physical Attempt 1
        +-- Physical Attempt 2
        +-- Fallback Attempt 3
```

The logical operation ID is stable for the Task and public Idempotency-Key. Physical provider attempts remain independently observable and billable.

## 4. Why the provider call is outside PostgreSQL

The provider call is a remote side effect and cannot participate in the PostgreSQL transaction. Wrapping a network call in a database transaction would not create distributed exactly-once semantics and would hold database locks across unpredictable provider latency.

R6 therefore uses two durable database boundaries around the remote call.

## 5. Begin transaction

For the first physical attempt:

```text
BEGIN
  reserve public command idempotency as in_progress
  SELECT Task FOR UPDATE
  validate ROUTING -> EXECUTING
  UPDATE Task projection + version
  INSERT TaskExecutionStarted
COMMIT

Provider call happens after COMMIT
```

If the same command is already `in_progress`, a duplicate request fails closed with `IDEMPOTENCY_IN_PROGRESS` and does not issue another provider call.

## 6. Retryable provider failure

A provider timeout, unavailable response, rate limit, or other classified retryable failure does not immediately fail the Task.

```text
Task remains EXECUTING
Idempotency:
  in_progress -> failed_retryable
```

A later explicit retry with the same key may reacquire the logical command:

```text
failed_retryable -> in_progress
```

Because the Task is already `EXECUTING`, this retry does not append a second `TaskExecutionStarted` event. It may create a new physical provider attempt.

## 7. Successful finalization

After a successful provider result:

```text
BEGIN
  lock in_progress idempotency
  SELECT Task FOR UPDATE
  validate EXECUTING -> VALIDATING -> COMPLETED
  UPDATE final Task projection
  increment version once per logical state transition
  INSERT TaskValidationStarted
  INSERT TaskCompleted
  complete idempotency with replayable model result
COMMIT
```

The final Task projection and both canonical business events cannot diverge.

## 8. Final non-retryable failure

For a final failure:

```text
BEGIN
  lock in_progress idempotency
  SELECT Task FOR UPDATE
  validate EXECUTING -> FAILED
  UPDATE Task projection
  INSERT TaskFailed
  mark idempotency failed_final with replayable error evidence
COMMIT
```

A retry of the same completed/final key returns the durable logical result rather than calling the provider again.

## 9. Crash semantics

### Crash before begin COMMIT

No execution-start business fact is durable and no provider call should have started.

### Crash after begin COMMIT but before provider call

The Task is `EXECUTING` and the command is `in_progress`. Automatic blind replay is forbidden because the system cannot infer provider-side execution solely from process state. Recovery is a later workflow/reconciliation concern.

### Crash after provider call but before finalization COMMIT

The same conservative rule applies. The command remains `in_progress`; a duplicate public request does not blindly repeat the physical provider call.

This is intentionally fail-closed.

## 10. Cost and evidence

Provider attempts remain responsible for attempt-level Trace and Cost evidence. R6 TaskEvent records only durable business lifecycle facts:

```text
TaskExecutionStarted
TaskValidationStarted
TaskCompleted / TaskFailed
```

TaskEvent must not be expanded into one event per provider transport detail.

## 11. Security boundary

All begin/finalize/idempotency operations require an explicit Project-scoped Principal. System Principal access still requires the frozen `task:system-access` capability. PostgreSQL transaction-local Tenant/Project scope remains the RLS boundary.

## 12. Non-goals

R6 does not provide:

- globally exactly-once provider transport;
- automatic replay of ambiguous in-progress provider calls;
- Temporal recovery policy;
- OIDC/RBAC/ABAC convergence;
- provider-specific distributed transactions.

## 13. Acceptance criteria

- public model execution requires `Idempotency-Key`;
- `requestId` changes do not create a new business command;
- exact completed retry replays the original logical result;
- same key with changed business request returns conflict;
- concurrent duplicate while provider execution is ambiguous returns in-progress and does not call the provider again;
- first execution appends exactly one `TaskExecutionStarted`;
- retryable failure leaves Task `EXECUTING` and allows explicit same-key reacquisition;
- successful finalization atomically records `VALIDATING` and `COMPLETED` facts;
- final failure atomically records `FAILED`;
- Task version increments once per logical state transition;
- PostgreSQL integration tests, unit tests, vet and builds pass.
