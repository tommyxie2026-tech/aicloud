# S3B Deterministic Temporal Adapter

## 1. Goal

Introduce the first physical Temporal integration behind the S3 durable-workflow boundary without activating a production Task lifecycle workflow yet.

S3B establishes four foundations:

1. a current Go/Temporal SDK baseline;
2. a Temporal-neutral `DurableEngine` boundary with trusted Tenant/Project/Task/Trace identity;
3. deterministic Task workflow identity independent of public API idempotency keys;
4. Outbox-driven workflow-start delivery into a real Temporal client adapter.

The governing ownership contract remains:

```text
PostgreSQL  = business/query state
TaskEvent   = append-only business history
Outbox      = transactional delivery intent
Temporal    = durable orchestration/execution history
```

## 2. Non-Goals

S3B does not:

- register or run the Task lifecycle workflow in `cmd/worker`;
- execute planning, routing, model, policy, approval, validation, or Tool activities;
- expose new public HTTP APIs;
- make Temporal a business source of truth;
- bypass TaskEvent, Outbox, Tenant ownership, or PostgreSQL RLS;
- claim exactly-once execution;
- remove the legacy development-only `workflow.Engine` seam yet;
- introduce real Tool or infrastructure side effects.

Worker registration and deterministic lifecycle execution belong to S3C.

## 3. Domain Changes

No business aggregate or Task state is added.

The S3 execution identity is frozen as:

```text
workflow_type = task-execution-v1
workflow_id   = task/<task_id>
```

The new Temporal-neutral durable boundary is represented by:

```go
type DurableEngine interface {
    Start(context.Context, StartRequest) (StartResult, error)
    Cancel(context.Context, CancelRequest) error
}
```

`StartRequest` explicitly carries trusted `tenant_id`, `project_id`, `task_id`, `trace_id`, and workflow type. `StartResult.RunID` is execution evidence only; Task remains business identity.

The existing `Engine.Start(ctx, taskID)` interface remains temporarily as a deprecated compatibility seam for the legacy non-durable `CreateTask` path. S3C must remove it after all production Task starts are routed through durable Outbox delivery.

## 4. API Changes

None.

The R7 OpenAPI contract is unchanged. Public callers neither choose Temporal Workflow IDs nor send Tenant/Project execution scope in the Task body.

## 5. Data Model Changes

No new table is required in S3B.

The durable Task-create Outbox message is hardened in two ways:

- the TaskCreated payload includes `traceId` so the delivery adapter can propagate correlation without trusting transport headers;
- `workflow.start` delivery identity becomes Task/Tenant-scoped:

```text
workflow-start:<tenant_id>:<task_id>
```

The public API `Idempotency-Key` remains stored only in the command idempotency record and continues to represent API business-command replay semantics.

## 6. Security Boundary

The `workflow.start` delivery adapter constructs `StartRequest` from trusted persisted Outbox fields and the committed TaskCreated payload.

It validates:

- destination is `workflow.start`;
- Tenant, Project, and Task identities are present;
- aggregate type is `Task`;
- aggregate ID equals Task ID;
- payload Task ID equals persisted Outbox Task ID;
- trace ID is propagated through the durable start request.

The adapter does not derive identity from HTTP headers or the Outbox idempotency string.

`DurableEngine` exposes no database, credential, provider, Tool Gateway, shell, Kubernetes, or infrastructure capability.

## 7. Runtime Flow

S3B freezes and implements this start path:

```text
POST /api/v1/tasks
    -> R6 Task command transaction
        -> Task
        -> TaskCreated
        -> workflow.start Outbox
        -> API idempotency result
    -> COMMIT

Outbox Dispatcher
    -> workflow.StartDeliveryAdapter
    -> DurableEngine.Start(StartRequest)
    -> TemporalEngine
    -> client.ExecuteWorkflow(
           ID = task/<task_id>,
           TaskQueue = configured queue,
           WorkflowType = task-execution-v1)
```

No worker is registered in this slice, so merging S3B alone does not activate workflow execution in production.

## 8. Failure Model

S3B must make workflow start safe under at-least-once Outbox delivery.

Required behavior:

- invalid persisted identity fails delivery and remains retry/dead-letter visible;
- Temporal unavailable returns an error so Outbox retry policy remains authoritative for delivery;
- successful Temporal start followed by failed Outbox acknowledgement is safe on redelivery;
- an already-started deterministic Task Workflow is normalized to idempotent success;
- cancellation of an already-absent Temporal execution is normalized as idempotent orchestration success;
- unrelated Temporal errors propagate and are never silently accepted.

S3B does not change Task business state based on Temporal start/cancel responses.

## 9. Idempotency Model

S3B explicitly separates three identities:

```text
API command key      = client supplied Idempotency-Key
Outbox delivery key  = workflow-start:<tenant_id>:<task_id>
Temporal Workflow ID = task/<task_id>
```

The public API key must never become the Temporal Workflow ID.

Temporal start uses a deterministic Workflow ID and rejects duplicate reuse. The SDK adapter normalizes `WorkflowExecutionAlreadyStarted` for the same deterministic identity into:

```text
AlreadyStarted = true
```

This is downstream durable deduplication, not an exactly-once guarantee.

Activity idempotency remains a later S3C/S3D responsibility.

## 10. Observability

S3B propagates:

```text
tenant_id
project_id
task_id
trace_id
workflow_id
workflow_run_id
workflow_type
```

`workflow_run_id` is returned by Temporal and remains diagnostic/execution evidence only.

The public `request_id` remains available in existing TaskEvent/idempotency evidence but is not used as workflow identity.

## 11. Cost Model

No new billable model or Tool operation is introduced.

Temporal orchestration cost accounting is not added in S3B. Future orchestration/platform cost events must remain Task/Trace attributable and separate from model/tool cost evidence.

## 12. Migration

S3B upgrades the project language/runtime baseline from Go 1.22 to Go 1.24 and introduces:

```text
go.temporal.io/sdk v1.44.1
go.temporal.io/api v1.62.12
```

The repository CI and Docker build image move to Go 1.24 in the same slice so local/build/runtime contracts do not drift.

The durable execution seam migrates incrementally:

```text
legacy Engine (development compatibility)
        +
DurableEngine (new S3 boundary)
        -> S3C makes DurableEngine authoritative
        -> legacy Engine removed
```

No historical Outbox message is interpreted as a Temporal ID. Even legacy messages with API-derived idempotency keys are deduplicated downstream by `task/<task_id>`.

## 13. Tests

S3B merge gates include:

- deterministic `task/<task_id>` derivation;
- empty Task identity rejected;
- trusted Tenant/Project/Task/Trace validation;
- Outbox adapter ignores legacy/public idempotency-key text for workflow identity;
- Outbox aggregate/payload identity mismatch rejected;
- Temporal start receives deterministic Workflow ID and configured Task Queue;
- AlreadyStarted normalization;
- deterministic cancellation identity;
- Task-create Outbox key/payload regression coverage;
- `go mod tidy` clean under Go 1.24;
- bilingual documentation validation;
- `gofmt`;
- `go test ./...`;
- PostgreSQL migration/Task/Outbox integration tests;
- `go vet ./...`;
- entrypoint builds.

## 14. Acceptance Criteria

S3B is complete when:

1. Go 1.24 and the selected Temporal SDK build cleanly in CI.
2. `DurableEngine` is Temporal-neutral and carries trusted scope explicitly.
3. Workflow ID is deterministic per Task and independent of API/Outbox keys.
4. Durable Task creation emits a Task-scoped workflow-start delivery key.
5. Outbox delivery can invoke the real Temporal client adapter.
6. duplicate Temporal start is idempotently normalized without hiding unrelated errors.
7. cancellation uses deterministic Task workflow identity but does not mutate Task business state.
8. no public API surface changes.
9. no Task lifecycle workflow or real side effect is activated yet.
10. full CI passes.

## 15. Rollback Strategy

S3B is behaviorally safe to roll back because no Temporal worker/lifecycle workflow is activated by this slice.

Rollback can remove the new adapter and restore the previous build baseline before S3C activation. Once S3C begins executing Task workflows, rollback rules must preserve single-executor ownership and replay compatibility rather than simply reverting binaries.

## Next Slice

S3C will register the Temporal worker and implement a deterministic Task lifecycle workflow using fake/idempotent Activities first. It will prove worker restart, workflow replay, Task terminal short-circuit, and lifecycle orchestration before model/tool side effects are integrated.
