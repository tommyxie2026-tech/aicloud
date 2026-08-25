# AI Cloud v0.1 Durable Task Workflow Implementation Plan

> Plan date: 2026-08-25  
> Scope: next implementation stage after governed model execution  
> Target: recoverable Developer AI Cloud task runtime

## 1. Objective

Move AI Cloud from a synchronous model-execution path to a durable,
observable and resumable Task workflow while keeping the current modular
monolith deployment model.

The stage is successful when a Task can survive API or Worker interruption,
resume from a persisted state, and expose one traceable history across model,
policy, tool, sandbox and evaluation activities.

```text
Task API
  |
  v
Task State + Event History
  |
  +--> Transactional Outbox -> Workflow Start Dispatcher
  |
  v
Durable Workflow
  |
  +--> Model Activity -> Model Runtime -> Cost Ledger
  +--> Tool Activity  -> Policy -> Credential -> Tool Gateway
  +--> Sandbox Activity -> Kubernetes Job boundary
  +--> Evaluation Activity -> Evidence Store
  |
  v
Task Result + Trace + Cost Report
```

## 2. Baseline and boundaries

### Reuse

- existing `Task` domain type and repository interfaces;
- existing Model Registry and governed Router;
- existing `internal/modelruntime` fallback and circuit-breaker logic;
- existing Trace, Evaluation, Cost Ledger and Tool Gateway packages;
- existing PostgreSQL migration runner and modular-monolith entrypoints;
- existing `cmd/worker` as the worker process boundary.

### Core commitment in this stage

- durable Task state transitions and event history;
- transactional Task creation, workflow-start outbox and reconciliation;
- workflow and activity interfaces;
- Temporal adapter behind the workflow boundary;
- cancellation, timeout, retry and resume semantics;
- task event API;
- unified activity trace attributes;
- fake Sandbox executor and policy-bound execution tests;
- deterministic Developer Agent workflow contract.

### Stretch goals

- first cluster-backed Kubernetes Job Sandbox integration test;
- complete review-ready Developer Agent acceptance fixture.

Stretch goals do not block the core stage exit. They move to the next sprint
if the durable workflow, consistency or replay exit gates are not complete.

### Explicitly defer

- splitting the control plane into microservices;
- a general-purpose Agent marketplace;
- multi-tenant billing and chargeback;
- gVisor/Firecracker as the default runtime;
- a full GitHub production connector and automatic merge;
- autonomous production changes without approval.

## 3. Target state machine and transition contract

The state machine is persisted, not inferred from the latest log line.

```text
CREATED -> PLANNING -> EXECUTING -> VALIDATING -> COMPLETED
   |          |            |             |
   |          |            +--> WAITING_APPROVAL --+
   |          |                                      |
   +----------+---------------------> CANCELLED <-----+
              |            |             |
              +------------+-----------> FAILED
                                           |
                                      retry event
                                           |
                                           v
                                       EXECUTING
```

Allowed transition rules must be centralized and tested. A retry creates a
new attempt and event records while preserving the original Task, Workflow
and Trace identities. `RETRY` is an event, not a persisted Task state.

| Current state | Allowed next states |
|---|---|
| `CREATED` | `PLANNING`, `CANCELLED`, `FAILED` |
| `PLANNING` | `EXECUTING`, `WAITING_APPROVAL`, `CANCELLED`, `FAILED` |
| `EXECUTING` | `VALIDATING`, `WAITING_APPROVAL`, `CANCELLED`, `FAILED` |
| `WAITING_APPROVAL` | `EXECUTING`, `CANCELLED`, `FAILED` |
| `VALIDATING` | `COMPLETED`, `EXECUTING`, `CANCELLED`, `FAILED` |
| `FAILED` | `EXECUTING` only through an explicit retry event and new attempt |
| `COMPLETED`, `CANCELLED` | none; both are terminal |

Cancellation is accepted from every non-terminal state. A cancellation request
is idempotent: repeated requests do not create another state transition, but
may append an audit observation when required by policy.

## 4. Work packages

### WP-1: Task Event History

Add a durable event model containing:

- event ID and sequence number;
- task ID, workflow run ID and trace ID;
- previous state and next state;
- event type and activity name;
- attempt number;
- actor or worker identity;
- error code and retryability;
- timestamp and metadata.

Implementation direction:

- add a PostgreSQL migration for `task_events`;
- add Task `version`, stable Workflow ID, current Workflow Run ID and attempt;
- enforce unique `(task_id, sequence)` event ordering;
- add repository transaction boundaries with memory and PostgreSQL implementations;
- make state transitions append-only;
- keep the materialized `Task.Status` for fast reads;
- reject invalid transitions before persistence;
- update event history and materialized state in one transaction using
  optimistic concurrency.

The required write contract is conceptually:

```go
type TaskTransitionRepository interface {
    TransitionTask(
        context.Context,
        string, // task ID
        int64,  // expected resource version
        TaskEvent,
    ) (Task, error)
}
```

The implementation must atomically validate the transition, allocate the next
sequence, append the event, update materialized Task state and increment the
resource version. A version conflict is retryable only after reloading state.

### WP-2: Task Creation, Outbox and Workflow Identity

Creating a Task and starting its workflow is a distributed dual write. Do not
call Temporal directly as the only action after a database commit.

The creation transaction must:

1. insert the Task with stable `workflow_id = task_id`;
2. append the initial `CREATED` event;
3. insert a `workflow_start_requested` outbox record;
4. commit all three writes atomically.

An outbox dispatcher starts the Temporal workflow idempotently and records the
Temporal Run ID. A reconciler retries pending or stale outbox records and
detects Tasks without a workflow. Workflow-start errors must never be ignored.

Identity and idempotency rules:

- clients may send `Idempotency-Key` on Task creation;
- `(tenant_id, idempotency_key)` is unique when tenant identity is available;
- Task ID is the stable Temporal Workflow ID;
- Temporal Run ID changes after continue-as-new or a new run;
- every Activity has a stable activity ID and explicit attempt number;
- tool and cost writes use immutable source event IDs for deduplication.

### WP-3: Workflow Boundary and Temporal Adapter

Separate the control-plane client, deterministic workflow definition and I/O
activities. The domain must not depend directly on Temporal types.

```go
type WorkflowClient interface {
    StartTask(context.Context, TaskWorkflowInput) (WorkflowRun, error)
    CancelTask(context.Context, string, string) error
    GetTaskRun(context.Context, string) (WorkflowRun, error)
}
```

The production adapter maps that interface to a deterministic Temporal
workflow:

```go
func TaskWorkflow(
    workflow.Context,
    TaskWorkflowInput,
) (TaskWorkflowResult, error)
```

Database, model, tool, sandbox and telemetry I/O must occur in Activities, not
inside workflow code. Workflow and Activity DTOs must be serializable and
versioned. The first workflow should orchestrate these activities:

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

Temporal is the first production adapter. A deterministic in-process adapter
may remain for unit tests and local development, but it must implement the same
client semantics. Task queue names, Workflow IDs and Activity IDs are explicit
configuration, not package constants scattered across the codebase.

### WP-4: Worker and API Responsibilities

API Server responsibilities:

- validate and create Task requests;
- start, query and cancel workflow runs;
- return materialized Task state;
- expose event history and trace references.

Worker responsibilities:

- execute workflow activities;
- enforce activity timeouts and retry policy;
- append state and activity events;
- invoke Model Runtime, Tool Gateway and Sandbox boundaries;
- finalize cost and evaluation evidence.

The API must not execute long-running model/tool/sandbox work inline after this
stage.

### WP-5: Cancellation, Timeout and Retry Policy

Define separate policies for:

| Operation | Retry | Timeout | Cancellation |
|---|---|---|---|
| Model call | bounded, provider-aware | request deadline | cooperative |
| Tool call | only idempotent actions | tool definition limit | terminate activity |
| Sandbox job | bounded infrastructure retry | execution limit | delete Job and collect logs |
| Evaluation | safe to retry | evaluator limit | mark incomplete |

Every retry must record:

- attempt number;
- original error;
- retry decision;
- added cost;
- selected fallback, if any;
- final outcome.

Activity retry does not imply that the external side effect may be repeated.
Each model accounting event, tool action and sandbox request must either be
idempotent by contract or protected by a durable deduplication record. Unknown
tool idempotency disables automatic retry.

### WP-6: Unified Trace and Cost Reconciliation

Use one trace hierarchy:

```text
Task
  └── Workflow Run
       ├── Model Call
       ├── Policy Decision
       ├── Tool Call
       ├── Sandbox Execution
       ├── Evaluation
       └── Cost Reconciliation
```

Every activity must carry `task.id`, `workflow.run.id`, `trace.id`,
`attempt.number` and `activity.name`. The existing Cost Ledger remains the
source of model-call cost events; this stage adds workflow, tool and sandbox
event integration points without claiming complete resource billing.

Cost events must contain a stable `source_event_id` or Activity ID and enforce
a uniqueness constraint. Temporal replay or Activity retry must return the
previously recorded result instead of charging the same operation twice.

### WP-7: Sandbox Boundary and Integration Test

Start with a Kubernetes Job adapter contract and a fake executor for unit
tests. The integration test must verify:

- task-scoped namespace or service-account identity;
- CPU, memory and timeout limits;
- network-deny default;
- workspace input and artifact output boundaries;
- cleanup after success, failure and cancellation.

The test must not require a production cluster for ordinary `go test ./...`.
Cluster-backed tests should be explicitly tagged or run in CI with a dedicated
environment.

The fake executor and its policy/cancellation tests are part of the core
commitment. The cluster-backed Kubernetes Job adapter test is a stretch goal.

### WP-8: Developer Agent Acceptance Workflow

Create a deterministic acceptance fixture before adding a live GitHub write:

```text
Issue fixture
  -> Task
  -> route decision
  -> plan artifact
  -> sandbox code change
  -> test result
  -> evaluation record
  -> cost report
  -> review-ready PR artifact
```

The first acceptance contract uses a fake repository and fake PR publisher.
Real GitHub credentials, repository writes and automatic merge remain outside
this stage. Completing the entire review-ready fixture is a stretch goal after
the durable workflow exit gates pass.

## 5. API changes

Add or stabilize the following endpoints:

```text
POST /api/v1/tasks/{id}/cancel
GET  /api/v1/tasks/{id}/events
GET  /api/v1/tasks/{id}/trace
GET  /api/v1/tasks/{id}/costs
```

`POST /api/v1/tasks` starts a workflow asynchronously and returns the Task
as `202 Accepted` with its stable Workflow ID, current Workflow Run ID and
status URL. It accepts an optional `Idempotency-Key`. `GET /events` returns
ordered events and supports a cursor or sequence number for polling.
Cancellation is idempotent and records the requesting actor.

The existing synchronous model execution endpoint remains available during
this stage for compatibility, is marked deprecated in documentation, and is
not used by new Task workflows. Removal or semantic replacement requires a
separate API decision. Until OIDC is implemented, actor identity is derived
only from a trusted server or workload identity; untrusted request fields must
not be represented as authenticated actors.

## 6. Delivery sequence

### Week 0: Consistency and contract freeze (2-3 days)

- approve the complete transition matrix and terminal-state behavior;
- define Task, Workflow, Run, Activity and source-event identity rules;
- define `TransitionTask` transaction and optimistic-lock contract;
- define outbox, dispatcher and reconciliation behavior;
- separate Workflow Client, Temporal Workflow and Activity interfaces;
- freeze asynchronous Task API and synchronous compatibility behavior;
- decide the local Temporal profile and automated test strategy.

Exit gate: contracts are represented in Go interfaces, schema notes and tests
before production Temporal or PostgreSQL implementation begins.

### Week 1: Persistence and transition safety

- finalize Task event schema;
- add migration and repository interfaces;
- implement transition validator;
- implement transactional `TransitionTask` and optimistic locking;
- implement Task creation outbox and dispatcher reconciliation contract;
- add memory/PostgreSQL tests;
- persist Task creation, routing and execution events.

Exit gate: invalid transitions are rejected and valid transitions are
replayable from event history; event and materialized Task state cannot diverge
under tested failure and concurrency cases.

### Week 2: Workflow and Worker boundary

- define separate client, workflow and activity interfaces;
- implement in-process workflow adapter;
- add Temporal workflow skeleton and worker registration;
- add Temporal as an optional Docker Compose profile with namespace, task queue,
  readiness and configuration documentation;
- move model invocation behind an activity;
- add activity timeout and retry policies;
- add workflow-start outbox dispatcher and orphan reconciliation.

Exit gate: a task runs through the worker boundary without API-side long-running
execution.

### Week 3: Control operations and secure execution

- add cancel and task-events APIs;
- add resume and retry tests;
- connect Tool Gateway and Policy events;
- add fake Sandbox contract, cleanup and cancellation tests;
- add stable source-event deduplication for cost and tool activities.

Exit gate: interrupted or cancelled tasks reach a deterministic terminal state.

### Week 4: Core acceptance and hardening

- add the deterministic Developer Agent workflow contract;
- reconcile model, tool, sandbox and evaluation evidence;
- add trace completeness assertions;
- run race, restart, replay, duplicate-delivery and failure-injection tests;
- update roadmap, API docs and operational runbook.

Exit gate: one core Task workflow survives API and Worker restart and produces
ordered events, trace, deduplicated cost and evaluation evidence. If capacity
remains, complete the cluster-backed Sandbox test and review-ready Developer
Agent fixture as stretch goals.

## 7. Definition of done

This stage is complete only when all conditions hold:

- `go test ./...`, `go vet ./...` and builds pass;
- a Task can be replayed from persisted events;
- Task state and event history update atomically with optimistic concurrency;
- Task creation cannot be orphaned from workflow start without detection and
  reconciliation;
- API Server restart does not lose an active Task;
- Worker restart resumes or safely retries an activity;
- cancellation is idempotent and audited;
- model fallback and retry costs remain visible without duplicate charging;
- tool and sandbox actions cannot bypass policy;
- trace IDs connect Task, workflow, model, tool, sandbox and evaluation events;
- synchronous API compatibility and deprecation behavior are documented;
- automated tests cover concurrent transitions, duplicate Task submission,
  workflow-start failure, Activity retry, Worker restart and cancellation;
- documentation records exactly what remains deferred.

The cluster-backed Sandbox integration and complete review-ready Developer
Agent fixture are stretch completion criteria, not blockers for the durable
workflow core.

## 8. Risks and mitigations

| Risk | Mitigation |
|---|---|
| Temporal adoption expands scope | Keep a workflow-neutral interface and one narrow workflow |
| Event history diverges from Task status | Treat events as append-only source and update materialized state transactionally |
| Task creation succeeds but workflow start fails | Use a transactional outbox, idempotent Workflow ID and reconciliation loop |
| Concurrent workers overwrite state | Add Task resource version and optimistic transition checks |
| Non-idempotent tools retry unsafely | Mark tool idempotency explicitly and deny automatic retry otherwise |
| Sandbox tests become environment-dependent | Keep fake executor tests mandatory; isolate cluster tests by build tag |
| Cost events are duplicated on replay | Use workflow/activity idempotency keys and immutable source event IDs |
| Developer Agent becomes a product expansion | Use a deterministic fixture and fake PR publisher first |

## 9. Runtime and source-of-truth decisions

- PostgreSQL is the source of truth for business Task state, audit events and
  cost facts.
- Temporal history is the source of truth for workflow execution, timers,
  retries, signals and resume behavior.
- A workflow never reconstructs business authorization solely from Temporal
  history; policy-relevant state is loaded through a versioned Activity.
- Local default startup remains lightweight. Temporal is enabled through an
  explicit Docker Compose profile, while unit tests use the in-process adapter
  or Temporal test environment.
- Cluster Sandbox tests are opt-in and must not make ordinary `go test ./...`
  depend on a Kubernetes cluster.

## 10. Related documents

- [AI Cloud Module Implementation Plan and Status](AI-Cloud-Module-Implementation-Plan-and-Status.md)
- [Durable Task Workflow Engineering Contract](../design/durable-task-workflow-engineering-contract.md)
- [Agent State Machine Design](../design/agent-state-machine.md)
- [Durable Task Workflow API v1](../api/task-workflow-api-v1.md)
- [Database Schema Design](../design/database-schema.md)
- [Temporal Development and Test Plan](../development/temporal-development-and-test-plan.md)
- [AI Cloud API Specification v1](../api/api-spec-v1.md)
- [Agent Runtime Design](../design/Agent-Runtime-Design.md)
- [Sandbox Architecture](../design/Sandbox-Architecture.md)
- [Evaluation Platform Design](../design/Evaluation-Platform-Design.md)
