# Task Scope Identity Migration Runbook

## Purpose

Operational procedure for migrating the S1 prototype from the `task_ownership` bridge to Task-owned `tenant_id`, `project_id`, and `created_by` fields.

This runbook applies to migration `005_task_scope_identity.sql`.

## Roles and Connections

Runtime and schema migration use different database connections:

```text
AICLOUD_DATABASE_URL
  -> API/Worker runtime role
  -> no SUPERUSER
  -> no BYPASSRLS
  -> RLS enforced

AICLOUD_MIGRATION_DATABASE_URL
  -> migration-only role
  -> schema migration entry point
  -> not used by API/Worker
```

The API process must not run migrations. `AICLOUD_RUN_MIGRATIONS=true` is rejected for PostgreSQL runtime mode.

## Preflight

Before applying migration 005, stop or drain writes that can create Tasks using the old schema.

Verify every existing Task has a matching ownership bridge row:

```sql
SELECT t.id
FROM tasks t
LEFT JOIN task_ownership o ON o.task_id = t.id
WHERE o.task_id IS NULL;
```

Expected result: zero rows.

Verify bridge scope is complete:

```sql
SELECT task_id
FROM task_ownership
WHERE tenant_id IS NULL OR tenant_id = ''
   OR project_id IS NULL OR project_id = ''
   OR subject_id IS NULL OR subject_id = '';
```

Expected result: zero rows.

If either query returns rows, stop. Repair ownership from authoritative business/audit records before migration. Do not invent a fallback tenant or project.

## Apply Migration

Run the dedicated migration entry point with the migration connection configuration:

```bash
AICLOUD_MIGRATION_DATABASE_URL='<migration-dsn>' go run ./cmd/migrate
```

Migration 005 will:

1. add nullable Task scope columns;
2. temporarily disable the old bridge RLS only inside the schema migration path;
3. backfill Task scope from `task_ownership`;
4. abort if any Task remains unscoped;
5. set Task scope columns `NOT NULL`;
6. add the tenant/project index;
7. enable and force RLS on `tasks`;
8. replace the prototype `aicloud.system_access` bridge policy with strict tenant/project RLS.

## Post-Migration Verification

Verify Task scope completeness:

```sql
SELECT count(*)
FROM tasks
WHERE tenant_id IS NULL OR tenant_id = ''
   OR project_id IS NULL OR project_id = ''
   OR created_by IS NULL OR created_by = '';
```

Expected result: `0`.

Verify RLS flags:

```sql
SELECT relname, relrowsecurity, relforcerowsecurity
FROM pg_class
WHERE relname IN ('tasks', 'task_ownership');
```

Expected result: both RLS flags are true for both tables.

Verify the runtime database role is not privileged:

```sql
SELECT current_user, rolsuper, rolbypassrls
FROM pg_roles
WHERE rolname = current_user;
```

For the runtime connection, `rolsuper=false` and `rolbypassrls=false` are mandatory.

## Tenant Isolation Verification

Using the runtime connection, execute requests through two independent project-scoped Principal contexts and verify:

```text
Tenant A / Project A
  -> can read Task A
  -> cannot read Task B

Tenant B / Project B
  -> can read Task B
  -> cannot read Task A
```

Cross-scope Task access must surface as not-found semantics at the application boundary.

## Rollback Strategy

Migration 005 is a security-hardening migration and should not be automatically rolled back after application traffic resumes.

If migration fails before commit, the migration transaction rolls back automatically. Fix the preflight data problem and rerun.

If a post-migration application regression requires rollback, prefer rolling the application forward/fixing the scoped repository rather than removing Task identity or RLS. An emergency schema rollback requires a reviewed maintenance procedure and must preserve the backfilled Task scope columns and audit evidence.

## Exit Criteria

Migration is accepted only when:

- all existing Tasks have non-empty Tenant/Project/CreatedBy identity;
- Task RLS is enabled and forced;
- runtime role has neither SUPERUSER nor BYPASSRLS;
- API/Worker starts successfully with `AICLOUD_DATABASE_URL`;
- migration command uses only `AICLOUD_MIGRATION_DATABASE_URL`;
- cross-tenant and cross-project verification passes.
