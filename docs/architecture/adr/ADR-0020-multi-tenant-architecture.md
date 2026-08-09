# ADR-0020: Multi-Tenant Architecture

## Status
Accepted

## Context
AI Cloud must support multiple organizations, tenants, projects, users, service accounts, agents, models, tools, workflows, budgets, traces, and artifacts without cross-tenant data leakage or policy bypass. Multi-tenancy is therefore a platform invariant, not an API feature added later.

## Decision
Every externally addressable resource and every execution record MUST belong to a tenant. Project is the primary sub-tenant boundary for application ownership and budgets. Tenant context MUST be propagated through API, workflow, model routing, tool execution, sandbox, persistence, cache, telemetry, cost ledger, and artifacts.

### Resource hierarchy

```text
Organization
  └── Tenant
       ├── Project
       │    ├── Agent
       │    ├── Workflow
       │    ├── Task
       │    ├── Policy Binding
       │    └── Budget
       ├── Model Access Policy
       ├── Tool Access Policy
       └── Audit / Cost / Trace
```

### Required identity context
Every request and internal command carries:

```text
request_id
trace_id
tenant_id
project_id
subject_id
subject_type
roles/scopes
```

`tenant_id` MUST NOT be accepted from an untrusted request body when it can be derived from authenticated identity and route context.

### Persistence isolation
For v0.1/v0.2, PostgreSQL uses shared schema with mandatory `tenant_id` columns, composite indexes, repository-level tenant filters, and PostgreSQL Row Level Security for high-value tables. Separate database/schema per tenant is not the default because it increases operational complexity before scale requires it.

All tenant-owned tables MUST include `tenant_id NOT NULL`. Project-owned tables also include `project_id NOT NULL`. Unique constraints that represent business identity MUST include tenant scope.

### Cache isolation
Redis keys MUST use a versioned tenant prefix:

```text
aicloud:v1:{tenant_id}:{project_id}:{resource}:{key}
```

Shared semantic or response caches are forbidden unless the cache entry is explicitly classified as public and contains no tenant-derived data.

### Object and artifact isolation
Object storage paths MUST be tenant/project scoped and authorization MUST be checked before issuing download/upload credentials:

```text
tenants/{tenant_id}/projects/{project_id}/tasks/{task_id}/...
```

### Runtime isolation
Every Task has an immutable tenant/project context. Workflow activities MUST reject commands whose resource tenant does not match task tenant. Sandboxes use task-scoped workload identity, namespace/label policy, network policy, resource quota, and short-lived credentials.

### Model and tool access
Model Registry entries may be global, tenant-private, or tenant-restricted. Routing MUST evaluate tenant model policy before scoring candidates. Tool Gateway MUST evaluate subject + tenant + project + tool + action + resource + risk before invocation.

### FinOps
Cost events MUST include tenant_id, project_id, task_id, trace_id, provider/model/tool dimensions, and immutable monetary/usage fields. Budgets are evaluated before execution and may deny, downgrade, or require approval.

### Observability
Telemetry MUST include tenant/project identifiers as controlled attributes. Secrets, prompts, retrieved documents, and model outputs are not automatically exported to shared telemetry backends. Payload capture is policy-controlled.

## Authorization model
Use OIDC for human authentication and workload identity for services/agents. Authorization combines RBAC for coarse permissions and policy evaluation for contextual decisions. Initial roles: tenant-admin, project-admin, developer, operator, auditor, viewer, service-agent.

## Failure policy
Missing or ambiguous tenant context is fail-closed. Cross-tenant mismatch is a security event. Fallback routing, retries, workflow replay, and recovery MUST preserve the original tenant boundary.

## Required tests
Implementation is not accepted until automated tests prove API isolation, repository isolation, cache key isolation, artifact isolation, trace/cost isolation, tool authorization isolation, sandbox identity isolation, and routing policy isolation.

## Consequences
This adds tenant context to most domain interfaces and persistence schemas, but prevents a costly security retrofit and provides the foundation for SaaS operation, enterprise chargeback, delegated administration, and tenant-specific governance.

## Implementation mapping
Canonical implementation details are defined under `docs/implementation/`, especially component contracts, API/data contracts, runtime/security flow, deployment/testing, and v0.1 milestones.