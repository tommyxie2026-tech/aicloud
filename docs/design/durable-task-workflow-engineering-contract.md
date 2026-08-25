# Durable Task Workflow Engineering Contract

> Status: implementation baseline for AI Cloud v0.1
> Deployment model: modular monolith with separate API and Worker processes
> Production workflow adapter: Temporal

## 1. Purpose

This document freezes the engineering contracts required before implementing
the durable Task workflow. It translates the roadmap into package boundaries,
identities, transaction rules and executable acceptance gates.

## 2. Authority boundaries

| Concern | Authority |
|---|---|
| Business Task state, audit events, idempotency and cost facts | PostgreSQL |
| Workflow execution, retries, timers, Signals and resume | Temporal history |
| Model selection and fallback | governed Router and Model Runtime |
| Tool authorization | Policy plus Tool Gateway |
| Untrusted execution | Sandbox boundary |
| Trace transport | OpenTelemetry-compatible telemetry seam |

Temporal history does not replace business audit storage. PostgreSQL does not
attempt to reproduce Temporal timers or Activity scheduling.

## 3. Stable identities

| Identity | Rule |
|---|---|
| Task ID | generated once and never reused |
| Workflow ID | equal to Task ID for v0.1 |
| Workflow Run ID | supplied by Temporal; may change across runs |
| Trace ID | stable across all attempts of one Task |
| Activity ID | deterministic from Task, logical step and occurrence |
| Source Event ID | immutable deduplication identity for cost/tool side effects |
| Idempotency Key | client-provided Task creation identity, scoped by tenant/project |
| Resource Version | monotonically incremented on each Task mutation |

Until tenant enforcement lands, the repository uses the configured default
tenant scope. The schema still reserves tenant identity so uniqueness rules do
not need a breaking migration later.

## 4. Package contracts

The implementation remains a modular monolith:

```text
internal/
  controlplane/   Task commands and API orchestration
  workflow/       neutral client contract and adapters
  taskruntime/    state machine and transition commands
  repository/     memory and PostgreSQL units of work
  modelruntime/   governed model Activity dependency
  toolgateway/    policy-bound Tool Activity dependency
  sandbox/        fake and Kubernetes execution adapters
  cost/           idempotent immutable cost ledger
  trace/          workflow/activity trace conventions
```

Packages are boundaries, not independently deployed services. `cmd/api-server`
and `cmd/worker` compose the same internal modules with different adapters.

### 4.1 Control-plane client

```go
type WorkflowClient interface {
    StartTask(context.Context, TaskWorkflowInput) (WorkflowRun, error)
    CancelTask(context.Context, string, string) error
    GetTaskRun(context.Context, string) (WorkflowRun, error)
}
```

The API uses this interface only after the Task creation transaction has
committed its outbox record. API code never executes model, tool or sandbox
work inline.

### 4.2 Temporal workflow

```go
func TaskWorkflow(
    workflow.Context,
    TaskWorkflowInput,
) (TaskWorkflowResult, error)
```

Workflow code must be deterministic. It may coordinate Activities, timers,
Signals, cancellation and versioned workflow branches. It must not access a
database, network, clock, random source or process environment directly.

### 4.3 Activities

```text
LoadTask
PlanTask
RouteModel
InvokeModel
ExecuteTool
RunSandbox
ValidateResult
RecordEvaluation
ReconcileCost
FinalizeTask
```

Activity inputs and outputs are versioned DTOs. Activities load current policy
and business state rather than trusting stale authorization embedded in
workflow history.

## 5. Task creation and workflow start

```text
POST /tasks
  -> database transaction
       -> insert Task(CREATED)
       -> append CREATED event
       -> insert workflow_start_requested outbox
  -> 202 Accepted

Outbox dispatcher
  -> StartWorkflow(workflowID = taskID)
  -> persist Temporal Run ID
  -> mark outbox delivered
```

The dispatcher treats an already-started Workflow ID as success. A reconciler
periodically scans pending/stale outbox records and Tasks without a known run.
This closes the database/Temporal dual-write gap.

## 6. Transition transaction

`TransitionTask` is one repository unit of work:

1. load and lock the Task or compare its resource version;
2. validate the transition matrix;
3. allocate `MAX(sequence) + 1` under the same Task lock;
4. append the immutable Task event;
5. update status, attempt, Workflow Run ID and resource version;
6. commit or roll back all writes.

The PostgreSQL adapter returns a typed version-conflict error. The caller must
reload before deciding whether retry is safe. The memory adapter implements the
same semantics for unit tests.

## 7. Idempotency and replay

- duplicate Task creation with the same scoped `Idempotency-Key` returns the
  original Task;
- duplicate outbox delivery starts no second workflow;
- duplicate transition source IDs return the existing event when payloads
  match and fail on identity collision when they differ;
- duplicate cost events are rejected by a unique source-event constraint;
- non-idempotent Tool calls set maximum Activity attempts to one unless the
  adapter implements an external idempotency key;
- Temporal replay may schedule no new external side effect solely because code
  was replayed.

## 8. Error taxonomy

| Code | HTTP | Retryable | Meaning |
|---|---:|---:|---|
| `invalid_argument` | 400 | No | malformed command |
| `not_found` | 404 | No | resource does not exist |
| `transition_conflict` | 409 | After reload | state/version conflict |
| `idempotency_conflict` | 409 | No | same key with different payload |
| `policy_denied` | 403 | No | deterministic policy denial |
| `workflow_unavailable` | 503 | Yes | dispatcher/Temporal unavailable |
| `activity_timeout` | 504 | Policy-specific | Activity exceeded timeout |
| `internal` | 500 | Unknown | unexpected failure |

Errors persisted in Task events contain a stable code, retryability, sanitized
message and cause reference. Provider secrets and raw credentials are never
persisted.

## 9. Configuration contract

```text
AICLOUD_WORKFLOW_MODE=inprocess|temporal
AICLOUD_TEMPORAL_ADDRESS=temporal:7233
AICLOUD_TEMPORAL_NAMESPACE=default
AICLOUD_TEMPORAL_TASK_QUEUE=aicloud-task-v1
AICLOUD_OUTBOX_POLL_INTERVAL=1s
AICLOUD_OUTBOX_LEASE_DURATION=30s
AICLOUD_RECONCILE_INTERVAL=30s
```

The default local mode remains `inprocess`. Temporal is enabled explicitly in
Compose or CI. Production must use PostgreSQL and the Temporal adapter.

## 10. Implementation slices

1. Domain states, transition matrix and typed errors.
2. Migration for Task version, events, idempotency and outbox.
3. Memory/PostgreSQL transactional repositories.
4. Outbox dispatcher and orphan reconciler with in-process adapter.
5. Temporal client, workflow, Activities and Worker registration.
6. Asynchronous API, cancellation, events and run metadata.
7. Cost/tool deduplication and trace propagation.
8. Fake Sandbox core acceptance; cluster adapter as stretch.

Each slice must merge with tests and leave the API server startable without
Temporal unless the Temporal profile is explicitly selected.

## 11. Acceptance evidence

- transition table tests cover every allowed and rejected edge;
- PostgreSQL tests prove event/state rollback and optimistic conflicts;
- duplicate create returns the same Task;
- workflow-start failure is repaired by the dispatcher/reconciler;
- Worker restart resumes the Task without duplicated cost;
- cancellation from each non-terminal state reaches `CANCELLED` once;
- `go test ./...`, `go vet ./...` and race tests pass;
- default Compose remains lightweight and Temporal profile smoke tests pass.

## 12. Related documents

- [Implementation Plan](../roadmap/2026-08-25-durable-task-workflow-implementation-plan.md)
- [Task State Machine](agent-state-machine.md)
- [Database Schema](database-schema.md)
- [Task Workflow API](../api/task-workflow-api-v1.md)
- [Temporal Development and Test Plan](../development/temporal-development-and-test-plan.md)
