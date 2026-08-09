# AI Cloud Workflow Source-of-Truth Contract

> Status: S0 Contract Freeze

## 1. Purpose

Define the ownership boundary between PostgreSQL domain state, TaskEvent business history and the durable workflow runtime. The goal is to make workflow technology replaceable without losing the Task domain model.

## 2. Three Sources with Different Responsibilities

```text
PostgreSQL
  owns current business state / query projection

TaskEvent
  owns immutable business history

Temporal (or another workflow runtime)
  owns durable execution history and orchestration progress
```

No single layer is allowed to silently substitute for another.

## 3. Canonical Ownership

### PostgreSQL

Canonical for:

- current Task projection;
- tenant/project ownership;
- approvals;
- route decisions;
- model attempts;
- tool invocations;
- cost/audit/evaluation records;
- idempotency records;
- outbox records.

### TaskEvent

Canonical for business-significant facts and Task transition history.

### Workflow Runtime

Canonical for execution mechanics such as:

- activity retry history;
- timers;
- durable waits;
- workflow signals;
- orchestration replay history;
- activity scheduling state.

Workflow history is not the authoritative business database.

## 4. Direction of Control

```text
API / Command
  -> Domain Transaction
     -> Task state + TaskEvent + Outbox
        -> Workflow signal/start
           -> Activities
              -> Domain commands/records
```

Activities do not directly rewrite arbitrary Task state. They call domain/application services that enforce Task transition contracts.

## 5. Workflow Identity

A Task may have a `workflow_id`, but the Task ID remains the business aggregate identity. Recommended deterministic workflow identifier:

```text
aicloud/task/{tenant_id}/{project_id}/{task_id}
```

The mapping must be unique and auditable.

## 6. Determinism Rules

Workflow code must be deterministic. Non-deterministic operations such as network calls, wall-clock access, random values and database reads happen in activities or through workflow-native deterministic APIs.

Business policy results that can change over time must be persisted/versioned and passed into workflow decisions rather than recomputed invisibly during replay.

## 7. Retry Ownership

Retry is layered:

```text
HTTP command retry      -> Idempotency layer
Workflow activity retry -> Workflow policy
Provider fallback/retry -> Model runtime policy
Tool retry              -> Tool-specific execution policy
```

A retry at one layer must not accidentally multiply retries at another layer. Each operation defines a maximum attempt budget and stable idempotency key.

## 8. Side Effects

External side effects must not be performed directly inside workflow logic.

```text
Workflow
  -> Activity
    -> Tool Gateway / Provider Runtime
      -> Idempotent side effect
```

Every side-effecting activity has a stable operation key derived from Task + logical operation + attempt policy.

## 9. Restart and Recovery

The system must tolerate:

- API process restart;
- Worker restart;
- workflow worker redeploy;
- temporary PostgreSQL outage;
- temporary provider/tool outage.

Recovery must not create duplicate external actions. The Task projection after recovery must be derivable from committed domain state/events, not from in-memory worker state.

## 10. Reconciliation

A periodic reconciler compares:

```text
Task projection
Workflow runtime status
TaskEvent history
```

and emits an auditable reconciliation finding when inconsistent states are detected. It must not silently rewrite history.

## 11. Workflow Replacement

Domain/Application layers may depend on a small `workflow.Engine` port, but must not import Temporal SDK types. Temporal-specific types remain in an adapter package.

Required capability boundary:

```text
StartTask
SignalTask
CancelTask
QueryExecutionState
```

The contract is intentionally smaller than the Temporal API.

## 12. Completion Semantics

Workflow completion is not equal to Task completion. A workflow may terminate abnormally while the Task is still recoverable, or a Task may reach a terminal business state before background evidence/reconciliation completes.

Task terminal state is decided by the Task domain transition rules.

## 13. Acceptance Criteria

- Domain packages contain no Temporal SDK dependency.
- Task business state is queryable without reading workflow history.
- Workflow replay does not recompute mutable policy/evaluation facts without versioned evidence.
- API/Worker restart produces no duplicate side effects.
- A Task can be reconciled from PostgreSQL + TaskEvent even if workflow runtime metadata is unavailable.
- Activity retries use stable idempotency identities.
- Workflow runtime can be replaced behind the port without changing Task/Policy/Router contracts.

## 14. Implementation Impact

S3 introduces the Temporal adapter and worker. Before S3, S2 must provide TaskEvent, Outbox and command idempotency primitives so workflow coordination does not rely on unsafe dual writes.