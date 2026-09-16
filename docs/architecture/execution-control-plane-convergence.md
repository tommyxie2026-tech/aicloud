# Execution Control Plane Convergence (S1 → ECP)

**English** | [简体中文](execution-control-plane-convergence.zh-CN.md)

> Status: DRAFT / Architecture Gate Input  
> Date: 2026-09-16

## 1. Conclusion

The repository does not need a parallel Execution OS rewrite. It already contains most of the required control-plane primitives, but they are spread across Task, execution, workflow, router, policy/evaluation and repository layers. The next step is **convergence around one Task → Execution lineage**.

Frozen ownership remains:

```text
Task            = business execution aggregate / business truth
TaskEvent       = immutable business history
Temporal        = durable orchestration
execution/      = execution domain and side-effect safety semantics
Router          = optimization among policy-approved targets
Policy          = permission authority
PostgreSQL      = durable business/execution evidence
Evaluation      = outcome/release gates
```

## 2. Existing capabilities to preserve

### 2.1 Task Domain — KEEP

`internal/domain.Task` already owns the canonical business lifecycle: CREATED → PLANNING → ROUTING → EXECUTING → optional WAITING_APPROVAL → VALIDATING → COMPLETED, plus terminal failure/cancellation/expiry states.

Do not introduce a second Task aggregate. Structured goals and constraints become immutable execution snapshots or compatible Task extensions.

### 2.2 TaskEvent / Outbox / Idempotency — KEEP

The existing Task event kernel already provides tenant/project scope, ordered append-only business history, transactional outbox, command idempotency and RLS. Execution lifecycle facts should be expressed through the canonical TaskEvent stream with execution/plan/node/attempt references in payloads.

### 2.3 Temporal Workflow — KEEP + EXTEND

The existing lifecycle already has Plan/Route/Execute/Validate stages. Temporal remains the durable orchestration substrate; replace stubs with real activities rather than building another workflow engine.

### 2.4 `execution/` — canonical Execution Domain

The package already defines Goal, versioned bounded-DAG plans, Execution, Nodes, Targets, CapabilityRequirement, policy decisions, EffectClass, Budget, Approval, Attempts, persistent workers, leases/fences and mutation uncertainty handling.

Do not recreate these contracts under `internal/taskexec` or another package.

### 2.5 Router / Evaluation / Policy — KEEP + EXTEND

The existing Router has a pure planning boundary and versioned deployment/model/pricing evidence. Evaluation already measures task success, cost per successful task, human intervention, quality, safety, reliability and latency. Admission, authorization and approval packages already provide security primitives. ECP should compose these capabilities instead of replacing them.

## 3. Real gaps

### G1 — Task ↔ Execution binding

Freeze the relationship as:

```text
Task 1 ────── N Execution
               ├── Goal snapshot
               ├── active PlanRevision
               └── N Node Attempts
```

Goal is an immutable snapshot of Task intent, not a second business Task. `Execution.TaskRef` provides direct lineage.

### G2 — durable parent persistence

The database already has `execution_node_runtime` and `execution_attempts`, but parent records are missing. Add:

```text
execution_goals
execution_plans
executions
```

using the next migration number, PostgreSQL JSONB/TIMESTAMPTZ and tenant/project RLS.

### G3 — workflow stubs

Converge the lifecycle to:

```text
Plan   → Goal snapshot + validated immutable ExecutionPlan
Route  → Policy + existing Router + frozen ExecutionTarget
Execute→ node runtime + PersistentWorker + durable Attempt
Validate→ evaluation + Task outcome / replan decision
```

### G4 — agentruntime overlap

`internal/agentruntime` should become an Agent `TargetInvoker` adapter/facade or later be deprecated. It must not become a second runtime domain.

### G5 — Execution Authorization facade

Compose existing identity, admission, RBAC/ABAC, data sensitivity, effects, budget and human approval into one execution policy facade that returns `execution.PolicyDecisionRecord`.

### G6 — API alignment

Keep `/api/v1/tasks` as the business API and extend it with plan/execution resources. Attempts remain primarily evidence/observability resources.

## 4. Package ownership

`internal/domain` owns Task/TaskEvent; `execution/` owns execution semantics; `internal/controlplane` coordinates commands/transactions; `internal/workflow` owns Temporal orchestration; `internal/router` selects targets; admission/authorization/approval/evaluation/repository remain specialized services; `internal/agentruntime` becomes an adapter; `internal/taskexec` must not be reintroduced.

## 5. Two state machines, not one

TaskStatus represents the user/business lifecycle. ExecutionPhase represents one internal execution lifecycle. An Execution failure does not necessarily force Task FAILED: if policy and budget allow, the control plane may create a new plan revision or another Execution.

## 6. Sources of truth

```text
Business state          → tasks
Business history        → task_events
Execution definition    → execution_goals / execution_plans / executions
Node mutable authority  → execution_node_runtime
Attempt evidence        → execution_attempts
Orchestration history   → Temporal
Operational telemetry   → traces / metrics / logs
Evaluation evidence     → evaluation_runs
```

Temporal history and observability data do not replace PostgreSQL business/audit truth.

## 7. Convergence roadmap

Use `ECP-Cx` naming to avoid collision with the repository's existing R1–R7 remediation sequence.

- **ECP-C1 — Task ↔ Execution Binding:** TaskRef, Goal snapshot, durable goals/plans/executions, scoped repositories and lineage tests.
- **ECP-C2 — Replace Plan/Route Stubs:** real plan builder, existing Router and policy decisions, target resolution.
- **ECP-C3 — Persistent execution:** replace ExecuteStub with node materialization, scheduler, PersistentWorker and model/tool/agent adapters.
- **ECP-C4 — Policy / Approval / Budget convergence:** unified execution authorization, durable budget and credential/tool permissions.
- **ECP-C5 — Evaluation / Outcome:** replace ValidateStub, persist evidence/outcomes and support controlled replanning.
- **ECP-C6 — Organization Memory:** promote evaluated successful trajectories into governed procedural/organization memory.

## 8. Corrections already applied

The gap analysis removed the duplicate `internal/taskexec` package and the conflicting duplicate execution migration, retained canonical `task_events`, aligned the API namespace to `/api/v1`, and reserved migration 019+ for ECP convergence.

## 9. ECP-C1 architecture gate

ECP-C1 must preserve canonical Task ownership, existing Temporal/Router semantics, tenant/project RLS, existing idempotency/outbox contracts, Task-attributed cost, and reconstructable task/goal/plan/execution/node/attempt lineage. Migration 019 is additive and must not break the existing Task API.

## 10. Immediate implementation slice

```text
execution.Execution.TaskRef
+ execution.Goal.TaskRef
+ 019_execution_control_plane.sql
+ ExecutionLineageStore
+ ScopedPostgresExecutionLineage
+ Task → GoalSnapshot adapter
+ contract/unit/integration tests
```

Only after this slice is stable should the Temporal PlanStub be replaced.
