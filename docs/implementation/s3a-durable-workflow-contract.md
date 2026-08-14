# S3A Durable Workflow Contract

## 1. Goal

Freeze the execution ownership and Temporal integration contract before replacing `workflow.NoopEngine`.

S3 must make Task execution restart-safe and replay-safe without creating a second business source of truth.

The governing invariant is:

```text
PostgreSQL  = business/query state
TaskEvent   = append-only business history
Temporal    = durable orchestration/execution history
Outbox      = transactional delivery intent
```

Temporal coordinates execution. It does not own canonical Tenant, Project, Task authorization, Task status, cost, audit, approval, or model registry state.

## 2. Non-Goals

S3A does not:

- implement Kubernetes or infrastructure side effects;
- move business state into Temporal workflow variables as the canonical record;
- introduce a second Task state machine;
- allow workflow code to call provider SDKs, databases, Tool Gateway adapters, or network clients directly;
- claim distributed exactly-once delivery;
- expose new public REST endpoints;
- change the frozen R7 Task HTTP contract.

## 3. Domain Changes

No new business aggregate is introduced.

`Task` remains the aggregate root. Existing canonical states remain:

```text
CREATED
PLANNING
ROUTING
EXECUTING
WAITING_APPROVAL
VALIDATING
COMPLETED
FAILED
CANCELLED
EXPIRED
```

Workflow progress must be expressed by validated Task commands/transitions and append-only TaskEvents. Temporal workflow-local state is execution state only.

The workflow identifier is derived deterministically from Task identity, for example:

```text
workflow_id = "task/" + task_id
```

A Task may have at most one active primary workflow identity in v0.1.

## 4. API Changes

None in S3A.

The existing Task creation API remains the external command boundary. API success means the Task business record and durable start intent are committed; it must not depend on synchronous completion of the full workflow.

A later S3 sub-slice may expose workflow diagnostics only if the OpenAPI/runtime contract is updated together.

## 5. Data Model Changes

S3 uses the existing Task, TaskEvent, Outbox, and command-idempotency kernel.

If workflow linkage requires persistence, the preferred model is explicit metadata attached to the Task or a dedicated execution reference that carries at least:

```text
tenant_id
project_id
task_id
workflow_id
workflow_run_id (observational, nullable)
workflow_type
created_at
updated_at
```

`workflow_run_id` is diagnostic evidence, not business identity.

Task projection + TaskEvent + Outbox delivery intent must remain transactionally consistent where a business transition occurs.

## 6. Security Boundary

Workflow execution never infers authorization from Temporal metadata.

Each activity invocation must receive or load enough immutable identifiers to re-establish trusted execution scope:

```text
tenant_id
project_id
task_id
trace_id
actor/service identity
```

Activities must re-enter repository/service boundaries that enforce Tenant/Project ownership and RLS.

Temporal payloads must not contain raw long-lived credentials. Tool credentials remain delegated, purpose-bound, task-bound, and short-lived through Credential Broker in later governed-execution slices.

Normal workflow workers use the normal worker database role and remain subject to RLS. Temporal availability must not create a database bypass path.

## 7. Runtime Flow

The S3 target flow is:

```text
API command
   -> PostgreSQL transaction
        Task projection
        TaskEvent
        Outbox workflow-start intent
        Idempotency result
   -> commit

Outbox dispatcher
   -> workflow.Engine.Start(task identity)
   -> Temporal StartWorkflow

Temporal workflow
   -> Activity: load Task snapshot
   -> Activity: transition to PLANNING
   -> Activity: planning work
   -> Activity: transition to ROUTING
   -> Activity: route decision
   -> Activity: transition to EXECUTING
   -> Activity: model/policy/approval/tool orchestration
   -> Activity: transition to VALIDATING
   -> Activity: validation
   -> Activity: transition to terminal state
```

Business writes occur only inside activities or application services invoked by activities. Workflow code itself must remain deterministic and side-effect free.

The first implementation sub-slice should prove start/restart/replay with deterministic fake activities before adding real model or Tool side effects.

## 8. Failure Model

The system must tolerate:

- API crash after database commit;
- Outbox dispatcher crash before or after Temporal start;
- duplicate Outbox delivery;
- Temporal client retry;
- workflow worker restart;
- activity timeout;
- activity retry;
- stale Task version;
- Task already terminal;
- cancellation racing with execution;
- Temporal temporarily unavailable;
- database temporarily unavailable.

Required semantics:

1. Database commit before workflow start is safe because Outbox retries start delivery.
2. Duplicate workflow-start delivery is safe because workflow ID is deterministic and conflict handling treats already-started as success for the same Task.
3. Activity retries are safe because business mutations use stable idempotency keys and/or optimistic Task version checks.
4. Terminal Task transitions are immutable.
5. A workflow observing a terminal Task exits without creating a second terminal transition.
6. Temporal outage delays execution but must not lose the Task.

## 9. Idempotency Model

Three layers are distinct:

### API command idempotency

Already provided by the R6 command kernel for supported commands.

### Workflow-start idempotency

Use deterministic `workflow_id = task/<task_id>`.

The Outbox delivery key for workflow start must be stable, for example:

```text
workflow-start:<task_id>
```

Repeated delivery for the same Task must not start parallel primary workflows.

### Activity idempotency

Each mutation activity must derive a stable operation key from business identity, not Temporal attempt number, for example:

```text
<task_id>:transition:<target_state>:<business_step>
<task_id>:route:<routing_revision>
<task_id>:model:<logical_attempt_id>
<task_id>:tool:<tool_invocation_id>
```

Temporal retries may repeat an activity attempt; they must not duplicate a committed business effect.

## 10. Observability

The following correlation fields must remain available across API, Outbox, workflow, activity, model attempt, and Tool invocation:

```text
request_id
trace_id
tenant_id
project_id
task_id
workflow_id
workflow_run_id
activity_id
model_attempt_id
tool_invocation_id
```

`workflow_run_id` and `activity_id` are execution evidence. `task_id` and `trace_id` remain the primary business/evidence correlation keys.

Workflow logs must not contain credentials or unbounded raw model payloads.

## 11. Cost Model

S3 orchestration itself does not change model/tool pricing semantics.

Retries must be attributable. Any retry that consumes billable resources must produce cost evidence with the same Task/Trace identity and a distinct logical/physical attempt representation.

Temporal internal retry count alone is not an acceptable billing identity.

## 12. Migration

S3 is introduced behind the existing `workflow.Engine` seam.

Recommended rollout:

```text
NoopEngine
   -> Deterministic in-process test engine
   -> TemporalEngine behind configuration
   -> shadow/dev activation
   -> default Temporal activation
```

The public API contract does not change during this rollout.

Existing Tasks created before Temporal activation may remain on the previous execution mode unless an explicit reconciliation/migration rule is implemented. S3 must not silently replay historical side effects.

## 13. Tests

S3 must add executable tests for:

- deterministic workflow ID derivation;
- duplicate Start treated idempotently for the same Task;
- API/database commit followed by delayed workflow start;
- duplicate Outbox delivery;
- worker restart and workflow resume;
- activity retry without duplicate Task transition;
- stale Task version conflict and retry/reload behavior;
- terminal Task short-circuit;
- cancellation/restart behavior;
- Temporal replay determinism;
- workflow code containing no direct database/provider/tool network calls;
- full PostgreSQL integration path;
- bilingual documentation validation;
- `gofmt`, `go test ./...`, `go vet ./...`, and entrypoint builds.

## 14. Acceptance Criteria

S3A Contract Gate passes when all of the following are frozen:

1. PostgreSQL is the canonical business/query state.
2. TaskEvent is canonical append-only business history.
3. Temporal is orchestration/execution history only.
4. Outbox is the only post-commit workflow-start delivery mechanism from durable Task commands.
5. Workflow ID is deterministic per Task.
6. Workflow code is deterministic and performs no direct external side effects.
7. Business writes occur through idempotent activities/application services.
8. Worker execution re-establishes Tenant/Project scope and remains subject to RLS.
9. Temporal retry does not define business idempotency.
10. Restart/replay tests are part of the merge gate.

## 15. Rollback Strategy

S3 must remain configuration-reversible while the Temporal engine is introduced.

Rollback may switch new Task starts back to the previous engine only if no Task can be executed by both engines concurrently.

Already-started Temporal workflows are drained, cancelled through an explicit business rule, or completed under the Temporal worker version that started them. Rollback must never create a second executor for the same Task.

## Proposed S3 Delivery Slices

```text
S3A Contract + engine boundary
    -> S3B Temporal client/worker adapter + deterministic start
    -> S3C Task lifecycle workflow with fake activities
    -> S3D durable activity idempotency + cancellation/retry/recovery
    -> S3E model/policy/approval orchestration integration
    -> S3F replay/restart/end-to-end proof
```

Real Tool side effects remain governed by S4 Tool Gateway/Credential/Sandbox boundaries; S3 may orchestrate the seam but must not bypass it.
