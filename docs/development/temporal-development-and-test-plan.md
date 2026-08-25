# Temporal Development and Test Plan

> Scope: first durable AI Cloud Task workflow
> Goal: production Temporal semantics without making the default developer path heavy

## 1. Runtime modes

| Mode | Workflow adapter | Persistence | Use |
|---|---|---|---|
| `inprocess` | deterministic local adapter | memory or PostgreSQL | default startup, unit tests |
| `temporal` | Temporal client/worker | PostgreSQL plus Temporal | integration, staging, production |

The in-process adapter implements the same control-plane client semantics but
does not claim restart durability. Acceptance of durable resume requires the
Temporal mode.

## 2. Configuration

```text
AICLOUD_WORKFLOW_MODE=inprocess|temporal
AICLOUD_TEMPORAL_ADDRESS=localhost:7233
AICLOUD_TEMPORAL_NAMESPACE=default
AICLOUD_TEMPORAL_TASK_QUEUE=aicloud-task-v1
AICLOUD_TEMPORAL_WORKER_IDENTITY=<host-or-pod>
AICLOUD_OUTBOX_POLL_INTERVAL=1s
AICLOUD_OUTBOX_LEASE_DURATION=30s
AICLOUD_RECONCILE_INTERVAL=30s
```

Configuration validation fails startup when `temporal` mode lacks an address,
namespace or task queue. API readiness in Temporal mode requires PostgreSQL and
the ability to reach Temporal; Worker readiness additionally requires successful
worker registration.

## 3. Docker Compose profile

Default development remains:

```bash
make compose-up       # PostgreSQL and Redis only
make run              # in-process workflow adapter
```

The implementation adds an opt-in profile:

```bash
docker compose --profile temporal up -d
AICLOUD_WORKFLOW_MODE=temporal make run
AICLOUD_WORKFLOW_MODE=temporal make worker
```

The profile should include the minimum Temporal development server and any
required UI as an optional secondary service. It must not make `make compose-up`
download or start Temporal.

## 4. Task queues and versioning

- first queue: `aicloud-task-v1`;
- Workflow ID: stable Task ID;
- workflow type: versioned exported name;
- Activity names: stable exported names;
- incompatible workflow changes use Temporal versioning/patch APIs or a new
  workflow type/queue;
- Worker deployment keeps old workflow code available until open executions no
  longer require it.

## 5. Worker composition

`cmd/worker` must:

1. load and validate configuration;
2. connect PostgreSQL and Temporal;
3. compose repositories and governed runtime dependencies;
4. register Task workflow and Activities;
5. start outbox dispatcher/reconciler ownership where configured;
6. expose structured startup/shutdown logs and readiness;
7. drain gracefully on `SIGTERM` within the deployment termination window.

Outbox dispatch may run in the API or Worker process, but ownership and lease
identity must guarantee safe concurrent instances. v0.1 should host it in the
Worker to keep API request handling side-effect free.

## 6. Test layers

### 6.1 Unit tests (mandatory on every change)

- state transition table;
- in-memory repository transaction and version conflict;
- workflow branch logic using Temporal test environment or local adapter;
- Activity retry classification;
- cancellation and approval Signal handling;
- cost/tool source-event deduplication.

No Docker or Kubernetes dependency.

### 6.2 PostgreSQL integration tests

- migrations on empty and pre-existing schema;
- atomic Task event/materialized state update;
- concurrent transition conflict;
- outbox lease, expiry and redelivery;
- scoped Task creation idempotency;
- duplicate cost-event rejection.

These tests run through an explicit integration target or CI service container.

### 6.3 Temporal integration tests

- outbox starts exactly one stable Workflow ID;
- API termination after commit does not orphan the Task;
- Worker termination during Activity resumes/retries safely;
- duplicate Activity delivery does not duplicate cost;
- cancellation reaches cleanup and one terminal event;
- approval Signal resumes the same Task and attempt rules are preserved;
- replay of recorded workflow history is deterministic.

Use Temporal's test environment for deterministic/time-skipping cases and the
Compose profile for process restart tests.

### 6.4 Sandbox integration tests

Fake Sandbox tests are mandatory. Kubernetes-backed tests use an explicit build
tag or separate CI job and verify namespace/service-account, resources,
network-deny, artifacts and cleanup. They are stretch scope for this stage.

## 7. Failure-injection matrix

| Injection point | Expected result |
|---|---|
| after Task/event/outbox commit, before response | duplicate client request returns original Task |
| Temporal unavailable during dispatch | outbox remains retryable; Task remains visible |
| dispatcher dies while leased | lease expires; another dispatcher starts same Workflow ID |
| Worker dies before Activity completion | Temporal retries according to policy |
| Worker dies after external success, before acknowledgement | source ID deduplicates repeated write/action |
| concurrent transition | one commit; one typed version conflict |
| cancellation during model/tool/sandbox | cooperative stop, cleanup and one `CANCELLED` transition |
| cost repository timeout after insert | retry returns existing source event; total unchanged |

## 8. CI gates

Required for core implementation PRs:

```bash
make fmt
git diff --check
go test ./...
go vet ./...
go test -race ./internal/...
```

Additional jobs:

- PostgreSQL migration/repository integration;
- Temporal replay and restart integration;
- Docker image build and Compose profile smoke test;
- optional cluster Sandbox test.

## 9. Operational evidence

Before declaring the durable stage complete, retain:

- test output and CI run links;
- one Task event timeline across API and Worker restart;
- one cancellation timeline;
- one duplicate-delivery cost reconciliation report;
- configuration and queue version used;
- list of open workflow executions before Worker upgrade.

## 10. Related documents

- [Durable Workflow Engineering Contract](../design/durable-task-workflow-engineering-contract.md)
- [Task Workflow API](../api/task-workflow-api-v1.md)
- [Database Schema](../design/database-schema.md)
- [Implementation Plan](../roadmap/2026-08-25-durable-task-workflow-implementation-plan.md)
