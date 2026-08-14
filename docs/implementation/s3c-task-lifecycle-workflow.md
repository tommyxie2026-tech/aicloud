# S3C Task Lifecycle Workflow Contract

## 1. Goal

Introduce the first deterministic Task lifecycle Workflow and Worker registration boundary while preserving the S3 ownership model:

```text
PostgreSQL  = business/query state
TaskEvent   = append-only business history
Outbox      = transactional delivery intent
Temporal    = durable orchestration/execution history
```

S3C proves that the orchestration definition is deterministic, restart-safe, replay-testable, explicitly registered, and isolated from real model/tool/infrastructure side effects.

The slice is intentionally a workflow-kernel proof, not production activation of end-to-end Task execution.

## 2. Non-Goals

S3C does not:

- connect the API process Outbox Dispatcher to Temporal in the production runtime;
- execute a real model request;
- make a real routing decision;
- invoke Policy or Approval as a production decision source;
- invoke Tool Gateway, credentials, shell, Kubernetes, SSH, database mutation tools, or cloud APIs;
- add a second Task state machine owned by Workflow variables;
- make Temporal the source of truth for Task status;
- expose Workflow state as a new public REST API;
- introduce deprecated Build-ID / legacy Worker Versioning APIs;
- claim exactly-once Activity execution.

Production database-backed Task transition Activities and cancellation/retry recovery hardening belong to S3D.

## 3. Domain Changes

No Task status is added or changed.

Canonical Task states remain:

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

The Workflow does not create another status enum. It consumes Task status snapshots returned by Activities and requests valid business transitions through Activity commands.

### Workflow identity

```text
workflow_type = task-execution-v1
workflow_id   = task/<task_id>
task_queue    = aicloud-task-v1
```

### Workflow input

The stable v1 input contains correlation identity only:

```go
type TaskWorkflowInput struct {
    SchemaVersion int
    TenantID      string
    ProjectID     string
    TaskID        string
    TraceID       string
}
```

Rules:

1. `SchemaVersion` starts at `1` and is required.
2. Tenant/Project/Task/Trace are required.
3. The Workflow input does not contain credentials, provider secrets, unbounded model payloads, or mutable authorization claims.
4. Business state is loaded through Activities rather than copied into Workflow input as a long-lived source of truth.

### Workflow result

The Workflow returns orchestration evidence, not an alternate business projection:

```go
type TaskWorkflowResult struct {
    TaskID          string
    TraceID         string
    ObservedStatus  TaskStatus
    AlreadyTerminal bool
    Steps           []string
}
```

`ObservedStatus` is the last status returned by an Activity. It is not authoritative over PostgreSQL.

## 4. API Changes

None.

The public R7 API and OpenAPI contracts remain unchanged.

S3C does not advertise Workflow diagnostics, signals, queries, approval endpoints, cancellation endpoints, or Task events through new public HTTP routes.

## 5. Worker and Registration Contract

The Worker process receives explicit Temporal configuration:

```text
AICLOUD_TEMPORAL_ENABLED=false
AICLOUD_TEMPORAL_ADDRESS=localhost:7233
AICLOUD_TEMPORAL_NAMESPACE=default
AICLOUD_TEMPORAL_TASK_QUEUE=aicloud-task-v1
AICLOUD_TEMPORAL_WORKER_STOP_TIMEOUT_SECONDS=30
```

`AICLOUD_TEMPORAL_ENABLED` defaults to `false` for S3C.

When disabled, no Temporal connection or polling loop is started.

When enabled, Worker startup must fail fast if address, namespace, task queue, registration, or Temporal client initialization is invalid.

### Stable registration names

Workflow and Activity names are explicit strings, not inferred Go symbol names:

```text
Workflow:
  task-execution-v1

Activities:
  aicloud.task.load.v1
  aicloud.task.transition.v1
  aicloud.task.plan.stub.v1
  aicloud.task.route.stub.v1
  aicloud.task.execute.stub.v1
  aicloud.task.validate.stub.v1
```

The Worker sets registration aliasing disabled when custom names are used, so Workflow/Activity calls use stable external names only.

S3C production Worker panic policy remains the SDK default/block behavior: non-determinism must block progress for repair rather than automatically fail all open business Workflows.

Worker shutdown is graceful and bounded by the configured stop timeout.

## 6. Activity Boundary

Workflow code may only interact with business or external state through Activities.

S3C freezes two classes of Activities.

### Business-state seam

```go
type LoadTaskInput struct {
    TenantID  string
    ProjectID string
    TaskID    string
    TraceID   string
}

type TaskSnapshot struct {
    TaskID    string
    TraceID   string
    Status    TaskStatus
    Version   int64
    Terminal  bool
}

type TransitionTaskInput struct {
    TenantID        string
    ProjectID       string
    TaskID          string
    TraceID         string
    ExpectedVersion int64
    To              TaskStatus
    Cause           string
    OperationKey    string
}
```

The Workflow never mutates a Task directly. It requests transitions through the Activity seam.

In S3C executable tests, this seam is backed by an in-memory fake that enforces the same Task aggregate transition rules and idempotency expectations. S3D replaces it with PostgreSQL-backed Activities using the existing aggregate/OCC/TaskEvent transaction kernel.

### Stub execution seam

Planning, routing, execution, and validation Activities in S3C are deterministic test doubles from the business point of view: they create no model, tool, network, credential, infrastructure, or external database side effect.

They exist only to prove Workflow sequencing and retry/replay behavior.

The runtime API process still does not automatically dispatch committed Tasks to this Worker in S3C, so the stub lifecycle cannot become an accidental production Task executor.

## 7. Lifecycle Flow

The reference orchestration is:

```text
Workflow Start
  -> LoadTask
     -> if terminal: return AlreadyTerminal
  -> Transition CREATED -> PLANNING
  -> PlanStub
  -> Reload/Transition PLANNING -> ROUTING
  -> RouteStub
  -> Reload/Transition ROUTING -> EXECUTING
  -> ExecuteStub
  -> Reload/Transition EXECUTING -> VALIDATING
  -> ValidateStub
  -> Reload/Transition VALIDATING -> COMPLETED
  -> Load final snapshot
  -> return orchestration evidence
```

The Workflow must not assume a transition succeeded until the transition Activity returns the committed/observed result.

Every mutation request uses the version returned by the most recent Task snapshot/transition result.

If an Activity observes a terminal Task at any point, the Workflow short-circuits without requesting another transition.

`WAITING_APPROVAL` is not exercised by the S3C happy path; approval orchestration is introduced only after the Policy/Approval integration contract is wired in S3E.

## 8. Determinism Contract

Workflow code must not directly use:

- `time.Now` or wall-clock time;
- random generators or UUID generation;
- native goroutines for Workflow control flow;
- direct network calls;
- database calls;
- Provider/Tool/Credential clients;
- environment-variable reads that affect replay decisions;
- process-global mutable state;
- map iteration when ordering affects emitted Workflow commands.

Workflow code uses Temporal Workflow primitives and Activity results for decisions.

The Workflow registers and invokes Activities by stable external string names.

### Backward-compatible changes

A checked-in replay test becomes a merge gate once S3C produces its first stable history fixture.

Future non-backward-compatible Workflow-definition changes must use a replay-safe migration mechanism such as `workflow.GetVersion`, or a separately reviewed Worker Deployment Versioning rollout. Deprecated Build-ID/version-set mechanisms must not be introduced.

## 9. Activity Retry and Timeout Contract

S3C uses bounded Activity options rather than infinite retry assumptions.

Initial defaults for the stub lifecycle:

```text
StartToCloseTimeout = 10s
ScheduleToCloseTimeout = 60s
InitialRetryInterval = 1s
BackoffCoefficient = 2.0
MaximumRetryInterval = 10s
MaximumAttempts = 3
```

These values are execution defaults, not business SLA.

A retry may repeat an Activity attempt. Therefore:

- read Activities must be side-effect free;
- transition Activities must accept a stable `OperationKey` and ExpectedVersion;
- stub Activities must be safe to repeat;
- Temporal attempt number must not be used as business idempotency identity.

S3D may tune retry classes by error type once real PostgreSQL-backed Activities exist.

## 10. Idempotency Model

S3C preserves the previously separated identities:

```text
API command key
Outbox delivery key
Temporal Workflow ID
a Business Activity OperationKey
```

Transition operation keys are deterministic from Task and business step, for example:

```text
<task_id>:transition:PLANNING:lifecycle-v1
<task_id>:transition:ROUTING:lifecycle-v1
<task_id>:transition:EXECUTING:lifecycle-v1
<task_id>:transition:VALIDATING:lifecycle-v1
<task_id>:transition:COMPLETED:lifecycle-v1
```

A repeated transition Activity with the same OperationKey must return the previously committed/equivalent result or a safe already-applied result; it must not append a second business transition.

The in-memory fake used by S3C tests must model this rule so the Workflow is designed against the future S3D contract rather than against a weaker test-only behavior.

## 11. Security Boundary

The Workflow input is trusted only because it came from the S3B durable Engine/Outbox path; it is still not an authorization decision.

Every real business Activity implementation must re-establish Tenant/Project execution scope before repository access.

S3C fake Activities validate scope identity but do not grant any production capability.

No Activity payload contains long-lived credentials.

Worker startup configuration may contain Temporal connection/auth configuration, but those values are process configuration and are never copied into Workflow history.

## 12. Failure and Recovery Model

S3C must prove behavior for:

- Worker process restart while Workflow is open;
- Activity retry after transient failure;
- duplicate Workflow start already handled by S3B identity semantics;
- Task already terminal before Workflow begins;
- Task becoming terminal between lifecycle steps;
- stale ExpectedVersion returned by a transition seam;
- Worker graceful shutdown;
- Workflow non-determinism detected during replay tests.

Required semantics:

1. Worker restart does not restart the business Workflow from zero; Temporal replays history and resumes orchestration.
2. Workflow replay must not execute already-recorded stub Activity effects again.
3. terminal Task observation causes a successful short-circuit result, not a second terminal mutation.
4. stale version is not blindly overwritten; the Activity/Workflow reloads or returns a retryable conflict path according to the frozen Activity contract.
5. Workflow panic/non-determinism is not converted into a fake business Task failure by the Workflow layer.

## 13. Observability

Worker/Workflow logs carry:

```text
tenant_id
project_id
task_id
trace_id
workflow_id
workflow_run_id
activity_type
activity_id
```

Logs must not contain credentials or unbounded raw Task/model payloads.

Workflow replay logging remains suppressed by default to avoid duplicate operational logs.

S3C does not add a second business event stream. TaskEvent remains the business event history once real transition Activities are connected.

## 14. Testing Strategy

S3C requires four layers of tests.

### A. Pure contract tests

- input validation;
- stable Workflow/Activity names;
- deterministic transition OperationKeys;
- terminal-state recognition;
- configuration defaults/fail-closed behavior.

### B. Temporal Workflow test environment

Using the Temporal Go SDK test environment with fake Activities:

- happy lifecycle sequence;
- expected Activity order;
- retry of a stub Activity;
- terminal-before-start short-circuit;
- terminal-between-steps short-circuit;
- stale-version/reload path;
- cancellation propagation where applicable.

### C. Worker registration/lifecycle tests

- Workflow registered under `task-execution-v1`;
- custom Activity names registered explicitly;
- registration aliasing disabled;
- disabled configuration performs no Temporal dial;
- invalid enabled configuration fails closed;
- graceful start/stop lifecycle is bounded.

### D. Replay compatibility gate

The first stable S3C Workflow history becomes a checked-in non-secret test fixture or an equivalent generated fixture. `worker.WorkflowReplayer` must replay it with the current Workflow implementation.

Future Workflow changes must keep this replay gate green or introduce an explicit reviewed compatibility/versioning change.

Normal repository gates remain:

```text
bilingual docs
go mod tidy
gofmt
go test ./...
PostgreSQL integration tests
go vet ./...
entrypoint builds
```

## 15. Acceptance Criteria

S3C is complete when:

1. `cmd/worker` has an explicit Temporal-enabled/disabled lifecycle instead of a skeleton loop.
2. disabled mode is the default and does not dial Temporal.
3. Worker registration uses stable external names and disables alias ambiguity.
4. `task-execution-v1` is the only primary Task Workflow type.
5. Workflow input contains only bounded correlation identity/schema version.
6. Workflow makes no direct external/database/provider/tool call.
7. lifecycle state changes are requested only through the Activity seam.
8. fake Activities model idempotent transition semantics without real external side effects.
9. Task terminal observation short-circuits safely.
10. retry tests prove repeated Activity attempts do not imply repeated business effects.
11. Workflow test-environment and replay compatibility gates pass.
12. no production API-to-Temporal dispatch path is activated yet.
13. deprecated Worker Versioning APIs are not introduced.
14. full CI passes.

## 16. Rollback Strategy

S3C remains rollback-safe because automatic production Task dispatch to Temporal is still not activated.

If the Worker code is rolled back:

- new API Tasks remain durable in PostgreSQL/Outbox;
- no new executor is silently substituted;
- already-started development/test Workflows are handled by a replay-compatible worker version or intentionally terminated in the non-production environment;
- rollback must not create a second executor for the same Task.

Production activation of Outbox -> Temporal Worker, PostgreSQL-backed Activities, cancellation reconciliation, and cross-restart recovery is explicitly deferred to S3D/S3F review gates.

## 17. S3C Implementation Order

```text
C1 Temporal config + worker lifecycle
 -> C2 stable Workflow/Activity registration
 -> C3 deterministic Task lifecycle Workflow
 -> C4 in-memory idempotent fake Activities
 -> C5 Temporal testsuite lifecycle/retry/terminal tests
 -> C6 replay compatibility fixture/gate
 -> C7 final architecture/security review
```

Only after S3C passes does S3D replace the fake business-state seam with PostgreSQL-backed idempotent Activities and connect durable cancellation/retry/reconciliation behavior.
