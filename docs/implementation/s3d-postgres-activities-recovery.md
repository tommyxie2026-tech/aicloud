# S3D PostgreSQL Activities, Durable Cancellation, and Recovery Contract

## 1. Goal

Replace the S3C test-only business-state seam with PostgreSQL/RLS-backed Task lifecycle Activities and freeze the recovery mechanisms required before automatic Task dispatch can ever be enabled.

S3D preserves the S3 ownership model:

```text
PostgreSQL  = canonical business/query state
TaskEvent   = append-only business history
Outbox      = transactional delivery intent
Temporal    = durable orchestration/execution history
```

S3D adds production-quality persistence and recovery primitives, but **does not enable automatic production Outbox -> Temporal Task execution**. Planning/routing/execution/validation are still not production business implementations until S3E.

## 2. Non-Goals

S3D does not:

- make Temporal the Task source of truth;
- allow an Activity to trust Tenant/Project fields merely because they are present in a Temporal payload;
- grant the Worker PostgreSQL `BYPASSRLS`, superuser, owner, or arbitrary database-admin capability;
- perform a cross-tenant scan of tenant-owned Outbox payloads;
- replace Policy, Approval, Router, model execution, or Tool Gateway with stubs in production;
- automatically mark a Task `FAILED` because a Worker crashed, Temporal replay failed, or an infrastructure Activity exhausted retries;
- enable production automatic dispatch while plan/route/execute/validate remain stubs;
- expose a public Task cancellation endpoint without a synchronized API/OpenAPI contract review;
- claim exactly-once Activity or Outbox delivery;
- use Temporal retry attempt number as business idempotency identity.

## 3. Security Boundary: Temporal Is a Transport, Not Authorization

The S3C Workflow input contains Tenant/Project/Task/Trace correlation identity. S3D must not treat those fields alone as authorization.

Before any Activity binds a PostgreSQL Tenant/Project scope, it must validate the execution context against the canonical S3 identity contract.

### 3.1 Execution attestation

A production lifecycle Activity validates at least:

```text
workflow_type == task-execution-v1
workflow_id   == task/<task_id>
worker namespace == configured trusted namespace
activity input task_id == task ID derived from workflow_id
```

The Activity implementation obtains Workflow execution information from the Temporal Activity context rather than accepting Workflow ID/type as arbitrary method arguments.

A mismatch fails non-retryably and performs no database operation.

### 3.2 Temporal control-plane trust

Production activation requires authenticated Temporal transport and namespace-level access control. Plain unauthenticated Temporal connectivity is allowed only in local development/test environments.

Activation requirements include one of the deployment-approved authenticated modes, for example:

- TLS plus mTLS client identity; or
- TLS plus a supported namespace/API-key authentication mechanism.

Only the AI Cloud control-plane starter identity may start/cancel `task-execution-v1` executions, and only the AI Cloud Worker identity may poll the Task Queue.

The exact SDK TLS/auth fields are verified against the current Temporal Go SDK during S3D implementation. Secrets remain process configuration/secret references and must never be copied into Workflow input/history.

### 3.3 Activity-scoped workload principal

After execution attestation, the Worker creates an explicit project-scoped internal service-account principal, not an implicit SystemPrincipal:

```text
type        = service_account
subject     = aicloud-workflow-worker
tenant_id   = attested tenant_id
project_id  = attested project_id
authn       = internal_workload_identity
issuer      = aicloud
```

Repository access then proceeds through the same `identity.RequireProject` and PostgreSQL RLS boundaries as other scoped runtime code.

The Worker database role remains RLS-enforced.

## 4. PostgreSQL-backed Activity Boundary

S3D replaces the S3C runtime `FailClosedLifecycleActivities` business-state methods with real PostgreSQL implementations for **LoadTask** and **TransitionTask**.

Planning/routing/execution/validation remain fail-closed for production until S3E.

### 4.1 LoadTask

`LoadTask` performs:

```text
Activity execution attestation
 -> create scoped service-account Principal
 -> repository.Get(TaskID) under Tenant/Project RLS
 -> verify returned Task tenant/project/trace correlation
 -> return TaskSnapshot
```

`LoadTask` is read-only and repeatable.

Cross-tenant, cross-project, Task/Trace mismatch, or missing Task must fail without revealing whether a Task exists in another scope.

### 4.2 TransitionTask

`TransitionTask` performs:

```text
Activity execution attestation
 -> create scoped service-account Principal
 -> load current Task under RLS
 -> terminal? return current snapshot
 -> expected_version mismatch? STALE_TASK_VERSION
 -> validate Task aggregate transition
 -> construct TaskEvent
 -> construct Activity idempotency record
 -> PostgreSQL CommitTransition(
        Task projection update,
        TaskEvent append,
        optional Outbox intents,
        idempotency completion
    )
 -> return committed TaskSnapshot
```

The business transition is never written with a plain `UPDATE tasks` outside the transaction kernel.

## 5. Activity Operation Idempotency

S3D reuses the existing R6 `idempotency_records` table. It does not create a second idempotency system.

### 5.1 Identity

Transition Activity records use a distinct operation namespace:

```text
operation = workflow.activity.task-transition.v1
key       = <task_id>:transition:<target_state>:lifecycle-v1
```

The primary key remains scoped by:

```text
tenant_id + project_id + operation + idempotency_key
```

### 5.2 Request digest

The canonical request digest includes at least:

```text
tenant_id
project_id
task_id
trace_id
expected_version
target_state
cause
operation_key
workflow_type
workflow_id
```

Same key + same digest returns the previously committed/equivalent result.

Same key + different digest is an invariant conflict and fails non-retryably.

### 5.3 Response replay

The idempotency response payload stores a bounded `TaskSnapshot` sufficient to answer a lost-response retry without appending a second TaskEvent.

On idempotency replay, the Activity returns the stored snapshot after scope validation.

### 5.4 Retention

Activity idempotency records must not expire while the related Task is nonterminal or while a supported Workflow execution can still legitimately retry/recover.

The initial default retention is at least 30 days and must be configurable. Cleanup must be Task/workflow-retention aware rather than deleting solely because wall-clock expiry was reached.

## 6. TaskEvent Contract for Workflow Transitions

Every committed lifecycle transition appends exactly one immutable TaskEvent in the same PostgreSQL transaction as the Task projection change.

Recommended event type:

```text
TaskStatusChanged
```

Payload contains bounded business evidence:

```text
from_status
to_status
cause
operation_key
workflow_id
workflow_type
```

Correlation fields continue to be first-class TaskEvent columns where already supported:

```text
tenant_id
project_id
task_id
trace_id
sequence
actor principal
occurred_at
schema_version
```

Temporal Activity ID/attempt may be recorded as diagnostic metadata, but Temporal attempt number is not the business idempotency key.

A replayed transition must not append another TaskEvent.

## 7. Error Classification

S3D freezes Activity error classes so Temporal retry behavior cannot accidentally change business semantics.

### Non-retryable / Workflow-handled

- invalid execution attestation;
- cross-scope/missing Task after trusted scope binding;
- invalid Task aggregate transition;
- idempotency key/digest conflict;
- malformed Activity input;
- unsupported workflow type;
- `STALE_TASK_VERSION` as a typed non-retryable Activity error that the Workflow explicitly handles by reloading authoritative Task state.

### Retryable infrastructure errors

- transient PostgreSQL connectivity failures;
- retryable transaction/serialization failures when safely repeatable;
- temporary dependency unavailability that has not produced a committed business effect.

### Business failure vs orchestration failure

An exhausted infrastructure retry does **not** automatically transition Task to `FAILED`.

Task `FAILED` is a business terminal state and requires an explicit business failure decision/command. Worker crash, Workflow non-determinism, Temporal outage, or database outage remains an orchestration/recovery condition.

## 8. Durable Business Cancellation

S3D introduces the internal durable cancellation command before any public cancellation API is advertised.

The required ordering is:

```text
Cancel command
 -> PostgreSQL transaction
      validate Task + expected version
      Task -> CANCELLED
      append TaskStatusChanged / TaskCancelled evidence
      insert workflow.cancel Outbox intent
      complete command idempotency
 -> COMMIT
 -> Outbox delivery
 -> DurableEngine.Cancel(task/<task_id>)
```

The orchestration cancellation call must never occur before the business transaction commits.

### Cancellation idempotency

Suggested delivery identity:

```text
workflow-cancel:<tenant_id>:<task_id>
```

Repeated cancel delivery is safe. Temporal `NotFound` is an orchestration-level idempotent success because PostgreSQL already owns the `CANCELLED` fact.

If `workflow.cancel` is delivered before a pending `workflow.start`, a later start remains safe: the PostgreSQL-backed Workflow loads the already-terminal Task and short-circuits without another business mutation.

### Terminal races

OCC decides completion-vs-cancellation races.

- If cancellation commits first, later lifecycle transitions observe terminal `CANCELLED` and stop.
- If `COMPLETED` commits first, a new cancellation request cannot rewrite the terminal Task.

No public REST cancellation route is added until the API/OpenAPI contract is reviewed in the same change.

## 9. Outbox Dispatch Without RLS Bypass

The current `ScopedPostgresOutbox` correctly requires an explicit Tenant/Project Principal and applies RLS. S3D must preserve this property.

A global dispatcher must therefore **not** solve scheduling by granting the Worker `BYPASSRLS` or by adding an application-controlled system GUC escape.

### 9.1 Global operational scope index

S3D introduces a minimal global operational resource such as:

```text
outbox_dispatch_scopes
----------------------
tenant_id
project_id
first_seen_at
last_seen_at
PRIMARY KEY (tenant_id, project_id)
```

This index:

- contains no Task ID;
- contains no Outbox ID;
- contains no event payload;
- contains no model/tool/user data;
- is classified as Global operational scheduling metadata, not tenant business data.

A migration-owned trigger on an RLS-validated `outbox_messages` insert upserts the scope row transactionally.

Runtime application code has no arbitrary direct DML permission to forge this index. Any `SECURITY DEFINER` trigger function must pin `search_path`, have PUBLIC execution revoked, and be covered by migration/security tests.

The Worker scheduler receives read-only access to the global scope index, enumerates explicit Tenant/Project scopes, then processes actual Outbox rows through `ScopedPostgresOutbox` under a scoped workload Principal and normal RLS.

```text
Global scope index
 -> (tenant, project)
 -> bind scoped Worker Principal
 -> ScopedPostgresOutbox.Lease
 -> destination adapter
 -> scoped ACK/failure update
```

This is an explicit two-stage scheduler. The global index never returns tenant-owned payloads.

### 9.2 Scope-before-delivery invariant

Every Outbox lease/ACK/failure operation re-establishes the same Tenant/Project scope. A delivery lease obtained in one scope cannot be acknowledged from another scope.

## 10. Outbox -> Temporal Activation Gate

S3D may implement the dispatcher wiring, but automatic production Task execution remains disabled by default:

```text
AICLOUD_WORKFLOW_DISPATCH_ENABLED=false
```

S3D integration tests may enable it against PostgreSQL + Temporal DevServer.

Production activation is prohibited until all of the following are true:

1. PostgreSQL-backed Load/Transition Activities pass RLS/idempotency tests.
2. Temporal transport/namespace authentication is configured.
3. Scope index scheduling passes cross-tenant tests without RLS bypass.
4. durable cancellation and duplicate delivery tests pass.
5. replay compatibility remains green.
6. real non-stub plan/route/execute/validate semantics are supplied by S3E.
7. S3F end-to-end recovery proof passes.

This prevents a no-op/stub lifecycle from ever auto-completing production Tasks.

## 11. Workflow Run Recovery Policy

Worker restart does not require a new Workflow run; Temporal replays the open run.

A different problem exists when a Workflow execution itself closes unsuccessfully while PostgreSQL still says the Task is nonterminal.

S3D freezes the desired engine-neutral behavior:

```text
nonterminal Task + active Workflow run     -> resume existing run
nonterminal Task + failed/closed run       -> recovery required; never auto-complete Task
terminal Task + active Workflow run        -> ensure workflow.cancel delivery
terminal Task + closed Workflow run        -> consistent
```

Recovery of a failed primary run must preserve the deterministic `workflow_id = task/<task_id>` while allowing a new RunID only under an explicitly reviewed failed-run recovery policy.

During implementation, the Temporal `WorkflowIDReusePolicy` is re-verified against the current official SDK. The target semantic is: a successful completed primary Task workflow is never duplicated; a failed orchestration run may be restarted only when PostgreSQL still owns a nonterminal Task and recovery is explicitly authorized.

The existing S3B `REJECT_DUPLICATE` policy must not be changed casually; changing it requires tests for stale start delivery, cancellation-before-start, completed Task, and failed-run recovery.

## 12. Reconciliation

S3D introduces reconciliation as diagnosis/repair orchestration, not a second source of truth.

A reconciler operates per explicit Tenant/Project scope and compares PostgreSQL/Outbox/Temporal execution evidence.

Minimum cases:

| PostgreSQL Task | Temporal | Required behavior |
|---|---|---|
| nonterminal | running/open | healthy |
| terminal | running/open | ensure durable cancel intent |
| nonterminal | missing/failed/closed | recovery anomaly; do not infer terminal business status |
| terminal | missing/closed | healthy or cleanup-only |

Additional Outbox anomalies:

- `workflow.start` dead-letter + nonterminal Task -> surface/review redrive;
- delivered start + missing failed execution + nonterminal Task -> recovery review;
- cancel dead-letter + terminal Task + running Workflow -> redrive/alert;
- lease expiration -> existing Dispatcher lease recovery remains authoritative.

### Redrive

If S3D adds Outbox redrive, it reuses the original scoped Outbox row/delivery identity where possible instead of inserting a semantically duplicate message that violates the existing unique key.

Automatic infinite dead-letter redrive is prohibited. Redrive is bounded, auditable, and policy-controlled.

A reconciler must never set Task `COMPLETED` merely because Temporal reports Workflow completion.

## 13. Temporal/Database Failure Matrix

S3D integration tests must cover at least:

- DB commit succeeds, Activity response is lost, Activity retries -> idempotency replay, one TaskEvent;
- DB transaction rolls back -> no Task projection change, no TaskEvent, no idempotency completion;
- stale ExpectedVersion -> typed stale error, Workflow reloads;
- cross-Tenant/Project Activity input -> no data disclosure, no mutation;
- forged/mismatched Workflow ID/type -> no database scope binding;
- Outbox start delivered twice -> one active primary Workflow semantics;
- start succeeds, ACK fails -> lease/redelivery safe;
- cancel commits, Temporal unavailable -> Task stays CANCELLED and cancel Outbox retries;
- cancel delivered before start -> any later Workflow observes terminal Task and exits;
- Worker restart with open Workflow -> replay/resume, no duplicate TaskEvent;
- failed Workflow + nonterminal Task -> reconciliation anomaly, no implicit Task failure;
- dead-letter delivery -> bounded redrive/alert semantics;
- two dispatcher instances -> SKIP LOCKED/lease ownership prevents concurrent delivery ownership;
- wrong-scope ACK -> rejected.

## 14. Observability and Audit

Every Activity and dispatcher operation carries/records as applicable:

```text
tenant_id
project_id
task_id
trace_id
workflow_id
workflow_run_id
activity_id
activity_type
operation_key
outbox_id
outbox_attempt
```

Security-sensitive failures include structured reason codes without leaking cross-tenant resource details.

Metrics include:

```text
workflow_activity_success_total
workflow_activity_retry_total
workflow_activity_idempotency_replay_total
workflow_activity_stale_version_total
outbox_dispatch_pending
outbox_dispatch_dead_letter
outbox_dispatch_retry_total
workflow_reconciliation_anomaly_total
```

Audit/business history remains TaskEvent and dedicated audit evidence; Temporal logs are not the authoritative audit ledger.

## 15. Migration Strategy

Recommended S3D delivery slices:

```text
D1 Activity execution-attestation + scoped workload identity
 -> D2 PostgreSQL LoadTask
 -> D3 PostgreSQL TransitionTask + TaskEvent + idempotency replay
 -> D4 durable Task cancellation + workflow.cancel adapter
 -> D5 global Outbox dispatch scope index + project-scoped scheduler
 -> D6 Temporal TLS/auth config + dispatcher wiring behind disabled flag
 -> D7 reconciliation/redrive primitives
 -> D8 PostgreSQL + Temporal integration/failure tests
 -> D9 final architecture/security review
```

Existing S3C `FailClosedLifecycleActivities` remains the default runtime backend until D1-D3 pass. Automatic workflow dispatch remains disabled through S3D production builds unless the later S3E/S3F activation gates are explicitly satisfied.

## 16. Acceptance Criteria

S3D passes only when:

1. Activity execution identity is attested before Tenant/Project scope binding.
2. Worker uses explicit service-account workload identity and an RLS-enforced DB role.
3. LoadTask is scoped/read-only/repeatable.
4. TransitionTask uses the existing PostgreSQL transaction kernel, not ad hoc Task updates.
5. one committed transition produces exactly one TaskEvent.
6. repeated same Activity operation returns the same/equivalent TaskSnapshot without a second TaskEvent.
7. same operation key with different digest fails.
8. stale version is reloadable and cannot overwrite newer business state.
9. cancellation commits business state and cancel Outbox before Temporal cancellation.
10. global Outbox scheduling enumerates only non-sensitive scope metadata and does not bypass RLS for actual messages.
11. wrong-scope lease/ACK/activity access is rejected.
12. Worker/Temporal production authentication has an explicit secure configuration path.
13. replay gate from S3C remains green.
14. reconciliation never derives Task terminal state from Temporal alone.
15. automatic production dispatch remains disabled while S3E execution steps are not real.
16. bilingual docs, migrations, unit/integration tests, `go vet`, and entrypoint builds are green.

## 17. Rollback Strategy

S3D migrations must be additive and backward compatible until production dispatch activation.

Rollback before activation:

- disable Workflow dispatch;
- keep committed Task/TaskEvent/Outbox records intact;
- Worker may return to fail-closed lifecycle Activities;
- no Task is silently handed to another executor.

After a real Workflow has started, rollback must preserve replay compatibility and single-executor ownership. Database rollback may not remove TaskEvent/idempotency/outbox evidence required to explain an already-committed effect.

A rollback must never use direct database edits to fabricate Task state in order to match Temporal history.
