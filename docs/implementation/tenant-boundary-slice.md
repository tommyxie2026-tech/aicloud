# Tenant Boundary Implementation Slice

## Status

`agent/tenant-contract-slice` now contains the S0-driven R1-R4 compliance changes and is pending final CI, security review and domain review.

## Contract Mapping

This slice implements the first enforceable set of ADR-0020, Identity Contract, Database RLS Model and Task Aggregate Contract requirements:

```text
Trusted Authenticated Ingress
  -> Explicit Principal
  -> Tenant / Project Scope
  -> Task Identity
  -> Scoped Repository
  -> PostgreSQL RLS
  -> Task-owned Evidence
```

## R1: Explicit Principal

Runtime context now uses `identity.Principal`:

```text
Principal
  ├─ User
  ├─ ServiceAccount
  └─ System
```

Trusted ingress headers only resolve into a verified Principal. Domain and repository code do not parse identity headers.

External APIs accept User or ServiceAccount principals, but reject an externally asserted System principal. System principals can only be created explicitly by internal code and require explicit capability and purpose.

## R2: Missing Identity Fails Closed

The old behavior is removed:

```text
missing tenant / missing scope
        -> trusted system access
```

The contract is now:

```text
missing Principal
        -> unauthenticated
```

Task repository access fails when Principal is absent. System-level Task access requires an explicit `PrincipalSystem` plus the `task:system-access` capability.

## R3: PostgreSQL RLS and Database Roles

Production runtime uses `ScopedPostgresTasks`:

```text
RequireProject Principal
  -> BeginTx
  -> set_config(aicloud.tenant_id, ..., true)
  -> set_config(aicloud.project_id, ..., true)
  -> SQL
  -> Commit/Rollback
```

Runtime startup validates the active PostgreSQL role:

- it must not be superuser;
- it must not have `BYPASSRLS`.

The prototype `aicloud.system_access=on` session flag is no longer used by the production access path.

Administrative access uses the separate `AdminPostgresTasks` entry point:

- separate admin database credential;
- controlled RLS-bypass database role;
- explicit System Principal in application context;
- required `database:admin` capability;
- not automatically exposed through the API runtime.

## R4: Atomic Task Ownership

Migration `005_task_scope_identity.sql` adds the following directly to `tasks`:

```text
tenant_id
project_id
created_by
```

It backfills from the old `task_ownership` migration bridge and aborts if any Task cannot be assigned verified ownership. It never invents fallback tenant/project identity.

The new Task creation path is:

```text
Verified Principal
  -> ScopedTasks binds Task identity
  -> single Task INSERT
```

This removes the previous orphan risk:

```text
INSERT task
  -> INSERT task_ownership fails
  -> orphan task
```

`task_ownership` remains only as a migration bridge and is no longer a runtime source of truth.

## Repository Security Invariants

1. Task `tenant_id`, `project_id` and `created_by` are immutable under normal Update.
2. Cross-tenant/cross-project Task access is represented as Not Found.
3. RouteDecision and CostEvent continue to inherit Task scope through the Task repository.
4. PostgreSQL RLS is defense in depth behind application authorization.
5. Global Model/Provider assets are not duplicated per tenant; model access remains a Policy/Router concern.

## Current Acceptance Coverage

Tests now cover:

- missing Principal fail-closed behavior;
- trusted ingress -> Principal resolution;
- rejection of external System-principal headers;
- User/ServiceAccount scope propagation;
- atomic binding of Task Tenant/Project/CreatedBy identity;
- cross-tenant Task Get/List denial;
- rejection of Task identity mutation;
- explicit System Principal + capability semantics;
- Route/Cost evidence inheriting Task ownership;
- migration removal of `aicloud.system_access` bypass;
- forced Task RLS with tenant/project transaction context.

## Next Step

PR #12 can become Ready only after R1-R4 pass CI and review. The next implementation sequence is R5-R7: Task aggregate transitions, TaskEvent/Outbox/Idempotency, then OpenAPI + OIDC/JWT + RBAC/ABAC convergence.
