# AI Cloud PostgreSQL RLS Model

> Status: S0 Contract Freeze

## 1. Purpose

Define database-level tenant isolation as defense in depth. RLS complements application authorization; it does not replace identity, RBAC or policy enforcement.

## 2. Security Principle

Application code must never obtain elevated database access merely by setting an untrusted session flag. Missing tenant context is an error for tenant-scoped roles, not a signal for administrative access.

## 3. Database Roles

Recommended production roles:

```text
aicloud_app_role
  - API queries/mutations
  - RLS enforced
  - no BYPASSRLS

aicloud_worker_role
  - workflow/activity queries/mutations
  - RLS enforced
  - no BYPASSRLS

aicloud_admin_role
  - narrowly controlled maintenance/reconciliation
  - separate credentials
  - audited use

aicloud_migration_role
  - schema migration only
  - not used by application runtime
```

Application and Worker credentials must not be able to assume the admin role through SQL/session configuration.

## 4. Tenant Session Context

Tenant-scoped transactions set verified values after Principal/Scope resolution:

```sql
SELECT set_config('aicloud.tenant_id', $1, true);
SELECT set_config('aicloud.project_id', $2, true);
```

`true` means transaction-local. Connection-pool reuse must not leak scope across requests.

## 5. RLS Policy Shape

For project-scoped tables containing tenant/project columns:

```sql
USING (
  tenant_id = current_setting('aicloud.tenant_id', true)
  AND project_id = current_setting('aicloud.project_id', true)
)
WITH CHECK (
  tenant_id = current_setting('aicloud.tenant_id', true)
  AND project_id = current_setting('aicloud.project_id', true)
)
```

Tenant-scoped resources may omit the project predicate. Global resources use separate access paths and are not made globally visible by a missing setting.

## 6. FORCE RLS

Tenant/project business tables should use:

```sql
ALTER TABLE ... ENABLE ROW LEVEL SECURITY;
ALTER TABLE ... FORCE ROW LEVEL SECURITY;
```

Runtime roles must not own tables in a way that silently bypasses policy.

## 7. Administrative Access

Administrative workflows use separate DB credentials/roles and explicit application-level System Principal authorization. The DB session is not elevated by:

```text
system_access=on
empty tenant ID
custom request header
```

The current S1 `aicloud.system_access` session-variable bypass is a prototype bridge and must be removed from the production RLS model before S2/S1 hardening is complete.

## 8. Schema Rules

Long-term tenant/project tables must carry scope directly where practical:

```text
tenant_id NOT NULL
project_id NOT NULL   -- for project resources
```

Task ownership should migrate from a side table into the `tasks` table. Task child tables should carry `task_id` plus denormalized tenant/project keys where this improves RLS enforcement and auditability; integrity must be protected by composite foreign keys or transaction rules.

## 9. Repository Transaction Pattern

```text
BeginTx
  -> set verified tenant/project config
  -> execute scoped queries
  -> commit/rollback
```

Queries outside a transaction must not rely on session-global scope state.

Repository code still applies explicit scope predicates where useful for clarity/performance. RLS is the second line of defense.

## 10. Connection Pool Safety

Tests must prove:

1. tenant A transaction cannot see tenant B;
2. pooled connection reused by tenant B does not retain tenant A scope;
3. failed/rolled-back transactions clear transaction-local settings;
4. app/worker role cannot bypass RLS;
5. admin role is not reachable from ordinary runtime credentials.

## 11. Migration Safety

Migrations that add RLS to existing tables require:

- backfill of tenant/project columns;
- validation that no orphan/unscoped rows remain;
- policy creation;
- shadow/test queries;
- enable RLS;
- force RLS;
- rollback plan.

Do not enable FORCE RLS before data backfill is validated.

## 12. Failure Semantics

Missing required tenant/project setting must yield zero rows or an explicit authorization/data-access error; it must never expose all rows.

Database authorization failures are mapped to stable application errors without leaking cross-tenant resource existence.

## 13. Acceptance Criteria

- Runtime app/worker DB roles have no BYPASSRLS capability.
- No application-controlled boolean can elevate a tenant transaction to admin.
- RLS is forced on tenant/project business tables after migration.
- Scope settings are transaction-local.
- Pool reuse isolation tests pass.
- Cross-tenant read/write tests pass at both Repository and raw SQL levels.
- Administrative access uses separate credentials and produces audit evidence.