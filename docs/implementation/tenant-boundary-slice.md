# Tenant Boundary Implementation Slice

## Status

Implemented in `agent/tenant-contract-slice`, pending CI and review.

## Contract Mapping

This slice implements the first enforceable part of ADR-0020 and the v0.1 implementation contract:

```text
Authenticated Request
  -> Tenant Scope
  -> Project Scope for Task APIs
  -> Scoped Task Repository
  -> Task-owned Route / Cost / Evidence
```

## External Identity Contract

For v0.1, an authenticating ingress injects and replaces:

```text
X-AICloud-Tenant-ID
X-AICloud-Project-ID
X-AICloud-Subject-ID
```

`/api/*` requires Tenant and Subject. `/api/v1/tasks*` also requires Project. Health and readiness are intentionally outside this boundary.

These headers are a trusted-ingress compatibility mechanism, not end-user authentication. Production OIDC/JWT verification is the next contract slice.

## Repository Contract

`tenantrepo.ScopedTasks` wraps the existing `domain.TaskRepository` without leaking provider or storage implementation details into the domain.

Rules:

- external Task creation binds Task -> Tenant/Project/Subject;
- scoped Get/Update must match ownership;
- scoped List filters foreign Tasks;
- foreign Task IDs return `repository.ErrNotFound`;
- no-scope context is reserved for trusted bootstrap/system work.

Route decisions and cost events are wrapped by Task ownership, so direct service calls cannot bypass the Task boundary.

## PostgreSQL Contract

Migration `004_task_tenant_ownership.sql` adds:

```text
task_ownership(
  task_id,
  tenant_id,
  project_id,
  subject_id,
  created_at
)
```

RLS is enabled and forced. PostgreSQL ownership operations set transaction-local:

```text
aicloud.tenant_id
aicloud.system_access
```

Tenant calls use `system_access=off`; trusted internal calls explicitly use `system_access=on`.

## Security Invariants

1. Tenant identity is established before Task resource dispatch.
2. Project identity is mandatory for Task APIs.
3. Task ownership is checked before route, cost, audit, trace, evaluation, model execution or tool execution.
4. Cross-tenant authorization failure is represented as not-found to reduce resource enumeration.
5. Model providers remain global platform assets in this slice; tenant model-access policy is implemented by the routing/policy layer rather than duplicating model records per tenant.

## Acceptance Tests

Required tests cover:

- unscoped API rejection;
- tenant scope propagation;
- health endpoint bypass;
- cross-tenant Task Get/List denial;
- trusted system context;
- route and cost evidence inheriting Task ownership.

## Next Slice

Slice 2 converges the running API with OpenAPI by adding OIDC/JWT verification, RBAC, idempotency, stable error envelopes, Task schema/state-machine convergence and executable OpenAPI contract tests.
