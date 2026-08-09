# v0.1 Executable Development Milestones

## Goal
Deliver one complete, safe, provider-independent task path for the reference infrastructure-control scenario. v0.1 optimizes for correctness of boundaries, not breadth of providers or agents.

## Slice 0 — Build and repository foundation
Deliver API/worker entrypoints, typed config, structured logging, Makefile, migrations, Compose, Helm, CI. Acceptance: `go test ./...`, `go vet ./...`, build, migration validation, Helm template all pass on main.

## Slice 1 — Tenant context and persistence foundation
Implement Tenant, Project, Subject, RequestContext middleware, PostgreSQL transaction/repository base, tenant-scoped query helpers, initial RLS policies, idempotency store. Acceptance: two-tenant integration suite proves no cross-tenant reads/writes; missing tenant context fails closed.

## Slice 2 — Model domain and MockProvider
Implement provider-neutral GenerateRequest/Response, normalized ProviderError, deterministic MockProvider, Model/ModelVersion/ProviderEndpoint repositories. Acceptance: same domain request executes through MockProvider without provider SDK types leaking into domain; model admission/lifecycle validation passes.

## Slice 3 — Router v1
Implement candidate filters and deterministic score, RouteDecision persistence, fallback chain, health/quota input interfaces. Acceptance: table-driven tests cover capability, policy, residency, license, health, budget rejection; selected/rejected reasons are persisted.

## Slice 4 — Task API and durable state
Implement POST/GET/cancel Task, TaskEvent append log, optimistic versioning, Temporal Workflow adapter, restart-safe state machine. Acceptance: Task survives API/worker restart; duplicate create with same idempotency key returns same task; cancellation reaches terminal state.

## Slice 5 — Agent plan and validation
Implement first Agent version that asks model for structured ChangePlan and validates target, expected current value, desired value, risk, rollback, and allowed operation type. Acceptance: malformed/unsafe plan never reaches execution; golden tests are deterministic with MockProvider.

## Slice 6 — Policy and approval
Implement PolicyEngine port, OPA adapter, policy decision persistence, WAITING_APPROVAL state and approve API. Acceptance: deny blocks execution; require-approval pauses durable workflow; expired/mismatched approval cannot resume action.

## Slice 7 — Tool Gateway and credential boundary
Implement Tool Registry, ToolGateway pipeline, CredentialBroker interface with fake/local adapter, Kubernetes scale Tool adapter behind an interface, audit records. Acceptance: Agent has no direct Kubernetes client; every invocation has policy decision and task-scoped credential reference; duplicate invocation is idempotent.

## Slice 8 — Sandbox foundation
Implement Kubernetes Job Sandbox adapter for generated validation/scripts where needed, restricted security context, resource/time/network controls, artifact collection. Acceptance: negative tests prove no hostPath, no privileged mode, no default external network, bounded runtime/resources.

## Slice 9 — Cost, audit and trace
Implement append-only CostEvent/AuditEvent, OpenTelemetry hierarchy, model/tool/sandbox usage emission, Task cost reconciliation. Acceptance: one Task can be reconstructed by trace_id and has reconciled cost even after retry/failure.

## Slice 10 — First end-to-end scenario
Run: `scale dev-gpu-cluster gpu-workers from 3 to 6` through API -> Task -> Workflow -> Router -> MockProvider -> ChangePlan -> Validator -> Policy -> optional Approval -> Tool Gateway -> fake Kubernetes adapter -> read-back validation -> COMPLETED. Acceptance: no direct execution path bypasses policy/tool boundary; full TaskEvents, RouteDecision, AuditEvents, CostEvents and trace exist.

## Slice 11 — Real provider and private model adapter
Add one commercial provider adapter and one OpenAI-compatible private/self-hosted adapter (vLLM/SGLang compatible). Acceptance: provider contract suite passes; Router can choose/fallback without changing Agent/Task code; credentials remain adapter-scoped.

## Slice 12 — Release hardening
Add load/smoke tests, backup/restore drill, migration roll-forward test, provider outage simulation, quota exhaustion, policy outage, tenant isolation regression, Helm production values documentation. Acceptance: production-ready definition in deployment-testing is met.

## Out of scope for v0.1
Marketplace, autonomous multi-agent swarms, ML-based router, per-tenant Kubernetes cluster, model training/fine-tuning, broad MCP ecosystem, complex billing settlement, multi-region active-active. Interfaces should permit later addition without implementing them now.

## Development issue template
Every slice should be decomposed into issues with: objective, affected packages, contract references, schema/API changes, security impact, telemetry/cost impact, tests, migration plan, acceptance criteria. A PR that changes a public contract must update implementation docs and both language versions.

## Recommended first coding order
Start with Slice 0/1 before expanding ADRs further. Then implement 2 -> 3 -> 4 -> 5 -> 6 -> 7 -> 9 -> 10. Sandbox can proceed in parallel after Tool Gateway interfaces stabilize. Real providers come only after MockProvider end-to-end is green.