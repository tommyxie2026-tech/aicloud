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

### Build in this stage

- durable Task state transitions and event history;
- workflow and activity interfaces;
- Temporal adapter behind the workflow boundary;
- cancellation, timeout, retry and resume semantics;
- task event API;
- unified activity trace attributes;
- first Kubernetes Job Sandbox integration test;
- Developer Agent workflow contract and acceptance test.

### Explicitly defer

- splitting the control plane into microservices;
- a general-purpose Agent marketplace;
- multi-tenant billing and chargeback;
- gVisor/Firecracker as the default runtime;
- a full GitHub production connector and automatic merge;
- autonomous production changes without approval.

## 3. Target state machine

The state machine is persisted, not inferred from the latest log line.

```text
CREATED
  |
  v
PLANNING -----> CANCELLED
  |
  v
EXECUTING ----> WAITING_APPROVAL
  |                    |
  |                    v
  +-------------- VALIDATING
                       |
             +---------+---------+
             v                   v
         COMPLETED             FAILED
                                  |
                                  v
                                RETRY
                                  |
                                  v
                              EXECUTING
```

Allowed transition rules must be centralized and tested. A retry creates a
new attempt and event records while preserving the original Task and Trace
identities.

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
- add `TaskEventRepository` with memory and PostgreSQL implementations;
- make state transitions append-only;
- keep the materialized `Task.Status` for fast reads;
- reject invalid transitions before persistence.

### WP-2: Workflow Boundary and Temporal Adapter

Define a workflow-neutral interface so the domain does not depend directly on
Temporal types:

```go
type TaskWorkflow interface {
    Start(context.Context, TaskStartRequest) (TaskRun, error)
    Cancel(context.Context, string, string) error
    Get(context.Context, string) (TaskRun, error)
}
```

The first workflow should contain these activities:

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
workflow contract.

### WP-3: Worker and API Responsibilities

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

### WP-4: Cancellation, Timeout and Retry Policy

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

### WP-5: Unified Trace and Cost Reconciliation

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

### WP-6: Sandbox Integration Test

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

### WP-7: Developer Agent Acceptance Workflow

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

The first acceptance test may use a fake repository and fake PR publisher.
Real GitHub credentials, repository writes and automatic merge remain outside
this stage.

## 5. API changes

Add or stabilize the following endpoints:

```text
POST /api/v1/tasks/{id}/cancel
GET  /api/v1/tasks/{id}/events
GET  /api/v1/tasks/{id}/trace
GET  /api/v1/tasks/{id}/costs
```

`POST /api/v1/tasks` starts a workflow asynchronously and returns the Task
with its workflow run ID. `GET /events` returns ordered events and supports a
cursor or sequence number for polling. Cancellation is idempotent and records
the requesting actor.

## 6. Four-week delivery sequence

### Week 1: Persistence and transition safety

- finalize Task event schema;
- add migration and repository interfaces;
- implement transition validator;
- add memory/PostgreSQL tests;
- persist Task creation, routing and execution events.

Exit gate: invalid transitions are rejected and valid transitions are
replayable from event history.

### Week 2: Workflow and Worker boundary

- define workflow-neutral interfaces;
- implement in-process workflow adapter;
- add Temporal workflow skeleton and worker registration;
- move model invocation behind an activity;
- add activity timeout and retry policies.

Exit gate: a task runs through the worker boundary without API-side long-running
execution.

### Week 3: Control operations and secure execution

- add cancel and task-events APIs;
- add resume and retry tests;
- connect Tool Gateway and Policy events;
- add Kubernetes Job Sandbox contract;
- add cleanup and cancellation tests.

Exit gate: interrupted or cancelled tasks reach a deterministic terminal state.

### Week 4: End-to-end acceptance and hardening

- add Developer Agent fixture workflow;
- reconcile model, tool, sandbox and evaluation evidence;
- add trace completeness assertions;
- run race tests and failure-injection tests;
- update roadmap, API docs and operational runbook.

Exit gate: one complete review-ready Developer Agent run has ordered events,
trace, cost and evaluation evidence.

## 7. Definition of done

This stage is complete only when all conditions hold:

- `go test ./...`, `go vet ./...` and builds pass;
- a Task can be replayed from persisted events;
- API Server restart does not lose an active Task;
- Worker restart resumes or safely retries an activity;
- cancellation is idempotent and audited;
- model fallback and retry costs remain visible;
- tool and sandbox actions cannot bypass policy;
- trace IDs connect Task, workflow, model, tool, sandbox and evaluation events;
- the Developer Agent fixture produces a review-ready artifact without real
  production mutation;
- documentation records exactly what remains deferred.

## 8. Risks and mitigations

| Risk | Mitigation |
|---|---|
| Temporal adoption expands scope | Keep a workflow-neutral interface and one narrow workflow |
| Event history diverges from Task status | Treat events as append-only source and update materialized state transactionally |
| Non-idempotent tools retry unsafely | Mark tool idempotency explicitly and deny automatic retry otherwise |
| Sandbox tests become environment-dependent | Keep fake executor tests mandatory; isolate cluster tests by build tag |
| Cost events are duplicated on replay | Use workflow/activity idempotency keys and immutable source event IDs |
| Developer Agent becomes a product expansion | Use a deterministic fixture and fake PR publisher first |

## 9. Related documents

- [AI Cloud Module Implementation Plan and Status](AI-Cloud-Module-Implementation-Plan-and-Status.md)
- [Agent State Machine Design](../design/agent-state-machine.md)
- [AI Cloud API Specification v1](../api/api-spec-v1.md)
- [Agent Runtime Design](../design/Agent-Runtime-Design.md)
- [Sandbox Architecture](../design/Sandbox-Architecture.md)
- [Evaluation Platform Design](../design/Evaluation-Platform-Design.md)
