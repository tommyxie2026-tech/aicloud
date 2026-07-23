# AI Cloud Module Implementation Plan and Status

> Status date: 2026-07-23  
> Repository: `tommyxie2026-tech/aicloud`  
> Target: AI Cloud v0.1 Developer AI Cloud MVP

## 1. Status rules

This document distinguishes design progress from implementation progress.

| Status | Meaning |
|---|---|
| Completed | Design or code is complete and available on `main` |
| Implemented, pending merge | Code exists in an open branch or pull request but is not yet part of `main` |
| In progress | Implementation has started but does not yet satisfy acceptance criteria |
| Planned | Design exists, implementation has not started |
| Deferred | Explicitly outside the current v0.1 scope |

Percentages are engineering estimates used for planning. A module is not considered complete until its acceptance criteria are verified on `main`.

## 2. Current overall progress

| Area | Estimated progress | Current state |
|---|---:|---|
| Product and strategic positioning | 100% | Completed |
| Architecture and ADRs | 90% | Completed; minor consolidation remains |
| Engineering design | 85% | Core API, schema, state-machine, deployment and repository structure are documented |
| Skeleton code | 80% | Implemented in Draft PR #1, pending merge |
| Runnable v0.1 platform | 20% | Minimal Model and Task APIs exist in Draft PR #1; real execution path is not implemented |
| Enterprise governance | 5% | Mostly design only |
| End-to-end Developer AI Cloud MVP | 10% | Scenario and interfaces are defined; full GitHub Issue-to-PR workflow is not implemented |
| Overall AI Cloud v0.1 | 25% | Architecture is mature; implementation remains early |

### Important repository state

The architecture and roadmap documents are available on `main`.

The first runnable skeleton is currently in:

```text
Draft PR #1
branch: agent/aicloud-v01-skeleton
```

It includes API and worker entrypoints, modular packages, in-memory repositories, migration contracts, Docker Compose and a minimal Helm chart. It must be reviewed and merged before it can be counted as completed on the main development line.

## 3. Module implementation plan

### Module A: Engineering Foundation and Developer Experience

**Purpose:** provide a buildable, testable and deployable modular-monolith foundation.

| Item | Status | Progress |
|---|---|---:|
| Go module and repository layout | Implemented, pending merge | 90% |
| API server entrypoint | Implemented, pending merge | 90% |
| Worker entrypoint | Implemented, pending merge | 70% |
| Configuration and structured logging | Implemented, pending merge | 80% |
| Makefile and basic tests | Implemented, pending merge | 80% |
| Dockerfile | Implemented, pending merge | 85% |
| Docker Compose development dependencies | Implemented, pending merge | 75% |
| Helm chart | Implemented, pending merge | 60% |
| CI workflow and release automation | Planned | 0% |

**Next actions:**

1. Review and merge Draft PR #1.
2. Add GitHub Actions for `gofmt`, `go test`, `go vet`, build and Helm lint.
3. Add version information and reproducible image tags.
4. Add development bootstrap and migration commands.

**Acceptance criteria:**

- `go test ./...`, `go vet ./...` and build pass on `main`;
- API server starts without external dependencies;
- PostgreSQL and Redis can be started through Compose;
- Helm template renders successfully;
- CI blocks unverified changes.

---

### Module B: Control Plane and Public API

**Purpose:** expose stable resources and coordinate Models, Agents, Tasks and policies.

| Item | Status | Progress |
|---|---|---:|
| Health and readiness endpoints | Implemented, pending merge | 90% |
| Model list/create API | Implemented, pending merge | 60% |
| Task list/create/get API | Implemented, pending merge | 60% |
| Agent API | Planned | 10% |
| Tool and Policy API | Planned | 5% |
| API versioning and error model | In progress | 35% |
| Authentication and tenant context | Planned | 0% |
| Event and streaming API | Planned | 0% |
| OpenAPI document | Planned | 0% |

**Next actions:**

1. Freeze v0.1 resource schemas.
2. Add Agent, Tool and Policy CRUD APIs.
3. Add idempotency keys, pagination and consistent errors.
4. Add task event streaming and cancellation APIs.
5. Generate and validate OpenAPI specifications.

**Acceptance criteria:**

- API schema is versioned and documented;
- all writes validate tenant, resource version and idempotency;
- task events can be streamed or polled;
- API errors are stable and machine-readable.

---

### Module C: Unified Model Protocol and Model Registry

**Purpose:** prevent provider lock-in and manage models as governed enterprise assets.

| Item | Status | Progress |
|---|---|---:|
| Model domain type and repository interface | Implemented, pending merge | 55% |
| In-memory Model Registry | Implemented, pending merge | 50% |
| PostgreSQL schema contract | Implemented, pending merge | 30% |
| Unified provider interface | In progress | 45% |
| Commercial API adapters | Planned | 10% |
| Local/private model adapters | Planned | 5% |
| Model capability and pricing metadata | In progress | 35% |
| Health, quota and capacity metadata | Planned | 0% |
| License and supply-chain metadata | Planned | 5% |
| Production admission workflow | Planned | 0% |

**Next actions:**

1. Implement the PostgreSQL Model Registry adapter.
2. Add provider adapters behind one internal protocol.
3. Add model version, endpoint, deployment mode, pricing, capability and risk fields.
4. Add license, upstream model, artifact digest, approval and provenance fields.
5. Add health checks and runtime capacity reports.

**Acceptance criteria:**

- the same task can be executed through at least two commercial providers and one local/private model;
- models can be enabled or disabled without changing Agent code;
- unapproved or non-compliant models cannot enter production routing;
- model selection reason is recorded in the Task trace.

---

### Module D: Task Runtime, Agent Runtime and Workflow

**Purpose:** execute long-running, recoverable and observable Agent tasks.

| Item | Status | Progress |
|---|---|---:|
| Task domain model and repository interface | Implemented, pending merge | 55% |
| Task state-machine design | Completed | 80% |
| No-op workflow seam | Implemented, pending merge | 25% |
| Agent runtime interface | Implemented, pending merge | 20% |
| Temporal integration | Planned | 0% |
| Durable retries and resume | Planned | 0% |
| Human approval workflow | Planned | 0% |
| Task cancellation and timeout | Planned | 0% |
| Artifact and result persistence | Planned | 5% |

**Next actions:**

1. Implement PostgreSQL Task persistence and event history.
2. Introduce Temporal worker and first durable workflow.
3. Implement the full task state machine:

```text
CREATED -> PLANNING -> EXECUTING -> WAITING_APPROVAL
        -> VALIDATING -> COMPLETED / FAILED / CANCELLED
```

4. Add retry, timeout, cancellation and resume semantics.
5. Persist plans, outputs, validation results and artifacts.

**Acceptance criteria:**

- a task survives API server or worker restart;
- workflow state and retries are deterministic;
- approval pauses and resumes the same task;
- every state transition is audited and traceable.

---

### Module E: Tool Gateway, Policy Engine and Credential Boundary

**Purpose:** prevent Models and Agents from directly accessing enterprise systems.

| Item | Status | Progress |
|---|---|---:|
| Tool Gateway interface | Implemented, pending merge | 20% |
| Fail-closed policy seam | Implemented, pending merge | 20% |
| Tool Registry | Planned | 5% |
| OPA/Rego integration | Planned | 0% |
| Short-lived credentials | Planned | 0% |
| Tool input/output validation | Planned | 0% |
| Audit of tool calls | Planned | 5% |
| MCP adapter | Planned | 0% |
| GitHub, shell and filesystem tools | Planned | 5% |

**Next actions:**

1. Define the Tool package schema, versions, risk levels and permissions.
2. Integrate OPA for pre-execution decisions.
3. Implement short-lived credential issuance and revocation.
4. Implement GitHub, filesystem and restricted shell tools.
5. Add input validation, output filtering and audit events.

**Acceptance criteria:**

- Agents cannot call enterprise systems outside the Tool Gateway;
- policy is evaluated before every sensitive tool call;
- credentials are task-scoped and short-lived;
- every invocation records subject, action, resource, decision and result.

---

### Module F: Sandbox and Secure Execution

**Purpose:** isolate untrusted code and Agent-generated actions.

| Item | Status | Progress |
|---|---|---:|
| Sandbox architecture design | Completed | 70% |
| Sandbox API contract | Planned | 10% |
| Kubernetes Job execution | Planned | 0% |
| Namespace and service-account isolation | Planned | 0% |
| CPU, memory and timeout controls | Planned | 0% |
| Network-deny default | Planned | 0% |
| Workspace and artifact controls | Planned | 0% |
| gVisor/Kata high-isolation profile | Deferred after MVP | 0% |

**Next actions:**

1. Implement Sandbox create, execute, collect and destroy operations.
2. Start with Kubernetes Job and isolated namespace/service account.
3. Enforce resource limits, execution timeout and network-deny defaults.
4. Add signed workspace inputs and controlled artifact outputs.
5. Add stronger runtime profiles after the basic path is stable.

**Acceptance criteria:**

- commands cannot escape the sandbox identity, filesystem or network policy;
- all executions have resource and time limits;
- sandbox lifecycle is bound to the Task;
- artifacts are collected before the sandbox is destroyed.

---

### Module G: Observability, Trace and Continuous Evaluation

**Purpose:** measure quality, latency, cost, safety and task outcomes at the system level.

| Item | Status | Progress |
|---|---|---:|
| Evaluation platform design | Completed | 65% |
| Telemetry interface seam | Implemented, pending merge | 15% |
| OpenTelemetry SDK wiring | Planned | 0% |
| Task trace hierarchy | Planned | 10% |
| Model/tool/sandbox span conventions | Planned | 0% |
| Offline evaluation datasets | In progress | 20% |
| Online evaluation and regression gates | Planned | 0% |
| Human feedback capture | Planned | 0% |
| Evaluation-driven routing | Planned | 0% |

**Next actions:**

1. Wire OpenTelemetry traces, metrics and logs.
2. Define trace hierarchy:

```text
Request -> Task -> Workflow -> Agent Run
        -> Model Call / Tool Call / Sandbox Execution / Evaluation
```

3. Create golden datasets for the first Developer Agent scenario.
4. Add offline regression tests and online result scoring.
5. Feed quality and failure results into routing decisions.

**Acceptance criteria:**

- a Task can be reconstructed from one trace ID;
- model, tool and sandbox latency and failures are measurable;
- model changes cannot be promoted when regression thresholds fail;
- evaluation results are versioned and reproducible.

---

### Module H: Task-Level FinOps

**Purpose:** optimize the cost of successful work rather than only counting Tokens.

| Item | Status | Progress |
|---|---|---:|
| Cost-governance ADR and roadmap | Completed | 65% |
| Task cost field | Implemented, pending merge | 15% |
| Token accounting | Planned | 5% |
| Tool and sandbox cost accounting | Planned | 0% |
| Retry and failed-attempt cost | Planned | 0% |
| Budget policies and pre-checks | Planned | 0% |
| Showback/chargeback reports | Planned | 0% |
| Cost-aware routing | Planned | 0% |

**Cost model:**

```text
Task total cost
= model input/output/cache
+ tool calls
+ workflow runtime
+ sandbox compute
+ storage and network
+ retries and failures
+ human approval
```

**Next actions:**

1. Create an immutable task cost ledger.
2. Record usage at Model Call, Tool Call and Sandbox Execution level.
3. Add project and tenant budgets.
4. Add cost estimates before task execution.
5. Add successful-task cost and value metrics.

**Acceptance criteria:**

- every completed or failed Task has a reconciled cost breakdown;
- budgets can block or downgrade execution;
- reports aggregate by tenant, project, Agent and model;
- routing can consider predicted total task cost.

---

### Module I: Capacity-Aware Routing, Hybrid Deployment and Failover

**Purpose:** keep workloads available when one model, provider or deployment becomes unavailable or overloaded.

| Item | Status | Progress |
|---|---|---:|
| Hybrid deployment architecture | Completed | 60% |
| Provider abstraction foundation | In progress | 35% |
| Routing policy | Planned | 5% |
| Health and capacity probes | Planned | 0% |
| Circuit breaker | Planned | 0% |
| Fallback chains | Planned | 0% |
| Quota-aware routing | Planned | 0% |
| Data-residency-aware routing | Planned | 0% |
| Reason recording and replay | Planned | 0% |

**Next actions:**

1. Define routing input: capability, evaluation, health, quota, queue, budget, residency and tenant policy.
2. Implement health and capacity ingestion.
3. Add circuit breakers and provider/model fallback chains.
4. Add private/local model routes and commercial API routes.
5. Record selection, rejection and fallback reasons.

**Acceptance criteria:**

- the system continues when a primary provider fails;
- no fallback violates policy, license or residency requirements;
- routing decisions are explainable and auditable;
- overload and quota exhaustion do not cause uncontrolled retries.

---

### Module J: Multi-Tenancy, Identity and Enterprise Governance

**Purpose:** provide secure ownership, authorization, isolation and accountability.

| Item | Status | Progress |
|---|---|---:|
| Tenant and identity design | Completed | 55% |
| Tenant data model | Planned | 5% |
| Authentication | Planned | 0% |
| RBAC | Planned | 0% |
| Tenant-scoped quotas and budgets | Planned | 0% |
| Credential Vault integration | Planned | 0% |
| Audit retention and export | Planned | 0% |
| Data isolation tests | Planned | 0% |

**Next actions:**

1. Add organization, tenant, project, user and service-account resources.
2. Add OIDC authentication and tenant context propagation.
3. Add RBAC and workload identity.
4. Enforce tenant scoping in repositories, caches, traces and artifacts.
5. Add audit retention and export controls.

**Acceptance criteria:**

- every request and resource has a tenant context;
- cross-tenant data access is prevented and tested;
- Agents use workload identities instead of user secrets;
- budgets, policies and audit records are tenant-scoped.

---

### Module K: Model License and Supply-Chain Governance

**Purpose:** prevent unverified open or third-party models from entering production.

| Item | Status | Progress |
|---|---|---:|
| Strategic requirement | Completed | 55% |
| Registry metadata design | In progress | 25% |
| License evidence ingestion | Planned | 0% |
| Artifact digest and signature | Planned | 0% |
| Vulnerability and malware scanning | Planned | 0% |
| Approval workflow | Planned | 0% |
| Runtime admission checks | Planned | 0% |
| Provenance and SBOM-like report | Planned | 0% |

**Next actions:**

1. Add license, upstream dependency, training-data disclosure and usage restriction fields.
2. Store model artifact digest and signature evidence.
3. Define approval states: discovered, reviewed, approved, restricted, revoked.
4. Add admission checks to Model Registry and routing.
5. Export model provenance and compliance reports.

**Acceptance criteria:**

- production routing only uses approved model versions;
- license and provenance evidence is retained;
- revoked models are immediately removed from new routing;
- historical Tasks remain traceable to an immutable model version.

---

### Module L: Developer AI Cloud End-to-End Scenario

**Purpose:** prove the platform through one complete business workflow.

| Item | Status | Progress |
|---|---|---:|
| Scenario and architecture | Completed | 75% |
| GitHub connector/tool | Planned | 10% |
| Repository checkout and workspace | Planned | 5% |
| Planning and code modification workflow | Planned | 5% |
| Test execution in sandbox | Planned | 0% |
| Human approval | Planned | 0% |
| Pull request creation | Planned | 5% |
| Evaluation and cost report | Planned | 0% |
| Failure recovery | Planned | 0% |

**Target flow:**

```text
GitHub Issue
-> Task
-> Model Routing
-> Agent Planning
-> Policy and Approval
-> Sandbox
-> Tool Gateway
-> Code Change
-> Test and Evaluation
-> Pull Request
-> Trace and Cost Report
```

**Acceptance criteria:**

The system can answer for every run:

- which model and version were selected;
- why they were selected and whether fallback occurred;
- which tools and credentials were used;
- which policy decisions and approvals occurred;
- which commands and tests ran in the sandbox;
- what the total and successful-task costs were;
- what evaluation score was produced;
- where the complete trace and generated Pull Request are located.

## 4. Recommended implementation sequence

The order below follows dependencies and avoids building enterprise features before the first reliable execution path.

### Stage 0: Merge and stabilize the skeleton

**Target duration:** 1-2 weeks  
**Modules:** A, B, partial C and D

Deliverables:

- merge Draft PR #1;
- green CI on `main`;
- stable API resource schemas;
- PostgreSQL migration runner;
- development and Helm smoke tests.

**Exit condition:** the repository has a reproducible, tested baseline on `main`.

### Stage 1: Persistence and real model connectivity

**Target duration:** 2-3 weeks  
**Modules:** B, C, partial H and I

Deliverables:

- PostgreSQL repositories for Models and Tasks;
- unified Model Provider interface;
- at least two model adapters;
- model health, pricing and basic usage records;
- first routing policy with deterministic fallback.

**Exit condition:** one API Task can call a real model and persist the full result.

### Stage 2: Durable task and Agent execution

**Target duration:** 3-4 weeks  
**Modules:** D, G, partial H

Deliverables:

- Temporal worker;
- durable task state machine;
- restart-safe retries and cancellation;
- OpenTelemetry trace foundation;
- immutable Task event and cost records.

**Exit condition:** a long-running task survives component restart and can be reconstructed from trace and event history.

### Stage 3: Secure tools and sandbox

**Target duration:** 4-5 weeks  
**Modules:** E and F

Deliverables:

- Tool Registry and Gateway;
- OPA policy enforcement;
- short-lived credentials;
- GitHub, filesystem and restricted shell tools;
- Kubernetes Job sandbox with network-deny and resource limits.

**Exit condition:** the Agent can safely modify and test code without direct infrastructure credentials.

### Stage 4: Evaluation, FinOps, supply chain and reliable routing

**Target duration:** 4-6 weeks  
**Modules:** G, H, I and K

Deliverables:

- golden evaluation datasets and regression gates;
- full task-cost ledger;
- capacity, quota and health-aware routing;
- circuit breaker and compliant fallback;
- model license and provenance admission checks.

**Exit condition:** the system selects models by quality, cost, compliance and availability, and explains every decision.

### Stage 5: Enterprise boundary and end-to-end MVP

**Target duration:** 4-6 weeks  
**Modules:** J and L

Deliverables:

- tenant context, OIDC and RBAC;
- tenant-scoped policy, budget, trace and artifact isolation;
- GitHub Issue-to-Pull Request workflow;
- approval, evaluation, trace and cost report.

**Exit condition:** the Developer AI Cloud MVP is demonstrable as an enterprise-controlled workflow.

## 5. Critical path

```text
Merge skeleton
-> PostgreSQL persistence
-> real model adapter and routing
-> Temporal task execution
-> Tool Gateway and OPA
-> Kubernetes sandbox
-> GitHub end-to-end workflow
```

The following capabilities are parallel workstreams after the first execution path is stable:

```text
OpenTelemetry and evaluation
Task-level FinOps
Capacity and fallback routing
License and supply-chain governance
Multi-tenancy and identity
```

## 6. Immediate next sprint

### Sprint objective

Move from a branch-only skeleton to a durable baseline on `main` and begin the first real model-backed task path.

### Sprint backlog

1. Review, fix and merge Draft PR #1.
2. Add CI and branch protection requirements.
3. Implement migration runner and PostgreSQL Model/Task repositories.
4. Finalize Model and Task API schemas.
5. Add one commercial model adapter and retain the deterministic mock adapter.
6. Record model version, usage, latency, cost estimate and trace ID.
7. Add the first routing policy and a deterministic fallback test.

### Sprint success criteria

- all checks pass on `main`;
- API creates and retrieves persisted Models and Tasks;
- a Task can invoke one real provider through the unified protocol;
- the selected model and routing reason are stored;
- restart does not lose Model or Task data;
- tests cover provider failure and fallback behavior.

## 7. Progress reporting cadence

Update this document at the end of each sprint using the following rules:

- only count code as Completed after it is merged to `main`;
- link each completed item to a PR or commit;
- record validation commands and test results;
- record blocked items and dependency owners;
- revise percentages only when acceptance criteria materially change;
- keep a short changelog at the bottom of this file.

## 8. Changelog

### 2026-07-23

- Created the first module-based implementation plan.
- Distinguished design completion from branch implementation and main-line completion.
- Recorded Draft PR #1 as the first runnable skeleton, pending merge.
- Added market-driven workstreams for task-level FinOps, capacity-aware failover, hybrid deployment and model supply-chain governance.
