# R6 Task Transaction Kernel Operational Runbook

> Status: production-shaped rollout and verification contract

## 1. Purpose

This runbook defines how to deploy and verify the R6 Task transaction kernel without allowing legacy Task writers to race with the new TaskEvent, Outbox and Idempotency contracts.

R6 makes the following boundaries operational:

```text
Task Create
  = Task + TaskCreated + workflow.start Outbox + Idempotency

Task Transition
  = Task Projection + canonical TaskEvent + required Outbox + Idempotency

Route Command
  = RouteDecision + Task Routing projection + TaskRoutingStarted + Idempotency

Model Command
  = Begin(EXECUTING + TaskExecutionStarted + in_progress Idempotency)
    -> remote provider attempt(s)
    -> Finalize(Task final projection + canonical events + final Idempotency)
```

## 2. Required preflight

Before deployment:

1. S1/R5 migrations must already be applied.
2. Runtime DB roles must not be SUPERUSER or BYPASSRLS.
3. Runtime API and Worker must use the scoped runtime DB role.
4. Migration credentials must remain separate from runtime credentials.
5. No old writer may continue mutating Task status during the cutover.
6. Existing Task rows must have Tenant, Project, Creator and Version identity.

Recommended checks:

```sql
SELECT count(*) FROM tasks
WHERE tenant_id IS NULL OR project_id IS NULL OR created_by IS NULL OR version < 1;
```

The result must be zero.

## 3. Drain legacy writers

Before applying R6 schema:

```text
Stop accepting new mutating Task requests
        |
        v
Drain old API instances
        |
        v
Drain old Workers
        |
        v
Confirm no legacy Task mutation sessions remain
```

Do not run mixed R5/R6 Task writers against the same database.

## 4. Apply migrations

Using the dedicated migration credential, apply migrations through at least:

```text
007_task_event_outbox_idempotency.sql
008_outbox_dispatch_leases.sql
```

Do not enable application traffic until schema verification succeeds.

## 5. Schema verification

Verify:

- `task_events` exists;
- `outbox_messages` exists;
- `idempotency_records` exists;
- TaskEvent has `UNIQUE(task_id, sequence)`;
- all R6 tables have RLS enabled and forced;
- TaskEvent runtime policies expose no UPDATE/DELETE path;
- Outbox lease columns exist;
- runtime DB roles cannot bypass RLS.

## 6. Deploy order

Recommended deployment order:

```text
1. Migration
2. API R6 build
3. Worker/Dispatcher R6 build
4. Health/readiness verification
5. Enable mutating traffic
```

The API may commit Outbox messages before the dispatcher becomes active. That is safe: pending delivery intent is durable and will be processed after the dispatcher starts.

## 7. Smoke test: Task creation

Send the same request twice with the same `Idempotency-Key`.

Expected:

```text
first request:
  HTTP 202
  new Task

second identical request:
  HTTP 202
  same Task identity
  Idempotency-Replayed: true
```

Verify exactly one:

- Task row;
- `TaskCreated` event with sequence 1;
- `workflow.start` Outbox intent;
- completed public Idempotency record.

Change the business body while reusing the same key. Expected: HTTP 409 `IDEMPOTENCY_CONFLICT`.

## 8. Smoke test: routing

Route a CREATED Task with a stable route Idempotency-Key.

Expected durable facts:

```text
TaskPlanningStarted
TaskRoutingStarted
RouteDecision
Task.status = ROUTING
Task.route_decision_id = RouteDecision.id
```

Exact route-command replay must return the original RouteDecision without recomputing against newer volatile routing signals.

## 9. Smoke test: model execution

Execute the routed Task with a model Idempotency-Key.

Expected successful lifecycle:

```text
ROUTING
  -> EXECUTING
  -> VALIDATING
  -> COMPLETED
```

Expected TaskEvents:

```text
TaskExecutionStarted
TaskValidationStarted
TaskCompleted
```

The public logical model operation ID must stay stable across provider fallback attempts, while each physical attempt has a distinct Attempt ID.

## 10. Retryable provider failure

Inject a retryable provider error such as unavailable, timeout or rate limit.

Expected:

```text
Task.status = EXECUTING
Idempotency.status = failed_retryable
```

Retry with the same public key. The command may perform a new physical provider attempt, but must not append another `TaskExecutionStarted` event.

## 11. Ambiguous provider execution

If the process crashes after the execution-begin transaction commits and before model finalization:

```text
Task.status = EXECUTING
Idempotency.status = in_progress
```

A duplicate public request must return in-progress/conflict semantics and must not blindly call the provider again.

Recovery of ambiguous operations belongs to the later durable Workflow/Reconciler contract.

## 12. Outbox crash recovery

Verify the crash boundary:

```text
lease Outbox
  -> downstream delivery succeeds
  -> dispatcher dies before MarkDelivered
```

After lease expiry another dispatcher must reclaim the message. A second physical delivery is allowed, but the downstream idempotency key must reduce both deliveries to one business effect.

Expected evidence:

```text
physical deliveries >= 2
business effects = 1
outbox final status = delivered
```

## 13. TaskEvent concurrency verification

Concurrent commands targeting the same Task version must produce exactly one successful mutation. Losers must observe `ErrVersionConflict` rather than creating parallel business histories.

After stress testing:

```text
TaskEvent sequence = 1,2,3,...,N
```

with no gaps caused by rolled-back contenders and no duplicate `(task_id, sequence)`.

## 14. Observability checks

Verify that the same logical identifiers can connect:

```text
request_id
trace_id
task_id
logical model operation_id
physical model attempt_id
outbox idempotency_key
```

TaskEvent remains business history; Trace remains execution telemetry.

## 15. Failure handling

### Before migration completion

Abort deployment, fix migration/preflight issue, and keep mutating traffic closed.

### After R6 schema is live but before R6 application rollout

Keep old writers drained. Prefer forward-fixing the R6 application rather than restoring legacy mutation traffic.

### After R6 has written TaskEvent/Idempotency data

Do not roll application writers back to a version that does not understand the R6 transaction contract. Use forward fix unless a separately tested backward-compatible rollback release exists.

## 16. R6 go/no-go gate

R6 is merge/deployment ready only when all are true:

- Task create is atomically idempotent;
- all production Task state mutations use the R6 command kernel;
- route decision persistence is atomic with routing state;
- model lifecycle is protected by begin/finalize command boundaries;
- concurrent event ordering tests pass;
- Outbox crash recovery is proven;
- duplicate physical delivery produces one downstream business effect;
- Tenant/Project RLS integration tests pass;
- bilingual documentation validation passes;
- `go test ./...`, integration tests, `go vet ./...` and entrypoint builds pass.
