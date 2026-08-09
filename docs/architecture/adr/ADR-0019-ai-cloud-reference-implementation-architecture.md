# ADR-0019: AI Cloud Reference Implementation Architecture

## Status
Accepted

## Context
AI Cloud has evolved from a model gateway into an enterprise AI operating system. A layered architecture alone is insufficient for engineering: the repository needs a concrete process topology, module boundary, dependency direction, persistence strategy, runtime sequence, and acceptance model.

## Decision
The v0.1 reference implementation is a **Go 1.22 modular monolith** with two primary processes (`aicloud-api` and `aicloud-worker`) and explicit domain module boundaries. External infrastructure is accessed through ports/adapters. The architecture may later be split into services without changing domain contracts.

```text
                    AI Cloud Platform

Clients / SDK / Enterprise Integrations
                |
          API / Auth Boundary
                |
+---------------+----------------------------------+
|                  Control Plane                   |
| Model Registry | Router | Policy | Eval | FinOps |
+---------------+----------------------------------+
                |
+---------------+----------------------------------+
| Identity / Tenant / Governance / Audit Boundary |
+---------------+----------------------------------+
                |
+---------------+----------------------------------+
| Workflow / Agent Runtime / Tool Gateway          |
+---------------+----------------------------------+
                |
+---------------+----------------------------------+
| Data / Knowledge / Artifact / Task State         |
+---------------+----------------------------------+
                |
+---------------+----------------------------------+
| Execution: Sandbox / Controllers / Model Runtime |
+---------------+----------------------------------+
                |
 PostgreSQL / Redis / Temporal / OPA / Object Store
 Kubernetes / vLLM / SGLang / Commercial Providers
```

## Process topology

`aicloud-api` owns public HTTP contracts, authentication/tenant context, synchronous validation, resource CRUD, task submission, query and approval endpoints. It does not execute long-running agent work.

`aicloud-worker` owns durable workflow activities, agent planning, routing invocation, policy/tool/sandbox orchestration, validation, evaluation scheduling and task-state progression.

Both processes use the same domain packages and ports. PostgreSQL/Temporal hold durable state; Redis is optional coordination/cache and is never source of truth.

## Mandatory module boundaries

Core modules: tenant, identity, model/provider, model/registry, router, task, agent, workflow, policy, tool, credential, sandbox, cost, audit, evaluation, observability, storage adapters, infrastructure adapters.

Provider SDKs are restricted to provider adapters. Temporal SDK types are restricted to workflow adapters. OPA/Kubernetes/PostgreSQL/Redis SDK details do not enter domain types.

## Production invariants

1. Provider agnostic: business workflows never import provider SDKs.
2. Tenant context is mandatory across API, persistence, routing, workflow, tool, sandbox, cost and trace.
3. Models propose; policy decides; humans approve when required; controllers execute.
4. Side effects occur only through Tool Gateway/Sandbox after policy checks.
5. Routing filters hard constraints before deterministic scoring.
6. Task state is durable and replay-safe; side effects are idempotent or reconcilable.
7. Audit and cost events are append-only.
8. Production model versions are admitted, immutable references.
9. Fallback never weakens policy, license, residency, security or hard budget.
10. Every reference task is reconstructable from TaskEvents + Trace + Audit + Cost.

## Canonical implementation specification
The detailed development contract is intentionally moved out of ADR prose into `docs/implementation/`:

- `README.md`: engineering blueprint and repository structure;
- `component-contracts.md`: Go interfaces and dependency boundaries;
- `api-data-contracts.md`: HTTP, PostgreSQL, event and migration contracts;
- `runtime-security-flow.md`: execution and security sequence;
- `deployment-testing.md`: Kubernetes topology and test/release gates;
- `milestone-v0.1.md`: executable implementation slices;
- `contracts/openapi-v1.yaml`: machine-readable API skeleton;
- `contracts/postgres-v0.1.sql`: reference database schema.

## Consequences
This decision deliberately avoids premature microservice decomposition while preserving extraction boundaries. It makes architecture directly testable and implementable and provides one source of truth between ADRs, roadmap, code and CI.