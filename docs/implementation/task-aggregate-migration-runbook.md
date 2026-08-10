# R5 Task Aggregate Migration Runbook

## Status

Required deployment procedure for migration `006_task_aggregate_state.sql`.

## Why a controlled cutover is required

R5 introduces a concurrency contract, not only new columns. S1 writers do not understand Task `version` and can write legacy `PENDING` / `RUNNING` states. After migration 006, the database enforces the canonical R5 state set and R5 repositories rely on optimistic concurrency.

Therefore an S1 mutating writer and an R5 mutating writer must not operate concurrently against the same database.

> Migration 006 is a controlled contract cutover, not a zero-downtime expand migration.

## Preconditions

Before migration:

1. PR #22/R5 code has passed unit, PostgreSQL integration, vet and build gates.
2. A database backup/snapshot is available according to the environment recovery policy.
3. Existing Task rows have valid S1 tenant/project identity from migration 005.
4. No unsupported Task statuses exist.
5. Operators have separate migration credentials; runtime API credentials are not used for schema changes.

Recommended checks:

```sql
SELECT status, count(*)
FROM tasks
GROUP BY status
ORDER BY status;

SELECT count(*)
FROM tasks
WHERE tenant_id IS NULL
   OR project_id IS NULL
   OR created_by IS NULL;
```

Expected pre-R5 statuses are `PENDING`, `RUNNING`, `COMPLETED`, `FAILED`, plus any terminal statuses already introduced by controlled maintenance. Unknown states must be resolved before proceeding.

## Cutover sequence

```text
1. Stop accepting new mutating Task requests
2. Drain S1 API/Worker Task writers
3. Confirm no active S1 Task mutation is running
4. Run cmd/migrate with migration credential
5. Verify migration 006 invariants
6. Deploy R5 API/Worker fleet
7. Run tenant/project and concurrency smoke tests
8. Re-enable Task mutations
```

Read-only health/readiness checks may remain available while writers are drained.

## Apply migration

Use the dedicated migration connection:

```text
AICLOUD_MIGRATION_DATABASE_URL
```

Do not use the runtime `AICLOUD_DATABASE_URL` credential for migrations.

Migration 006 performs:

- `PENDING -> CREATED`;
- `RUNNING -> EXECUTING`;
- `version=1` backfill and NOT NULL enforcement;
- terminal `completed_at` backfill;
- canonical Task status constraint;
- Task scope/status index creation.

## Post-migration verification

Run:

```sql
SELECT status, count(*)
FROM tasks
GROUP BY status
ORDER BY status;

SELECT count(*)
FROM tasks
WHERE version IS NULL OR version < 1;

SELECT count(*)
FROM tasks
WHERE status NOT IN (
  'CREATED','PLANNING','ROUTING','EXECUTING','WAITING_APPROVAL',
  'VALIDATING','COMPLETED','FAILED','CANCELLED','EXPIRED'
);
```

Both validation counts must be zero.

Verify schema constraints and indexes exist, and repeat the Task RLS isolation smoke test used by S1.

## Application smoke test

With an authenticated project-scoped Principal:

```text
Create Task
  -> CREATED/version=1

Route Task
  -> PLANNING -> ROUTING
  -> version increases

Execute deterministic/mock model
  -> EXECUTING -> VALIDATING -> COMPLETED
  -> version increases at each persisted mutation
```

Then intentionally submit a stale repository update in the integration/smoke environment and verify it returns a version-conflict error rather than overwriting the newer Task.

## Failure handling

### Migration fails before commit

Each repository migration runs transactionally through `cmd/migrate`. Fix the cause and rerun after confirming the migration was not recorded in `schema_migrations`.

### Migration succeeds but R5 application deployment fails

Do **not** immediately restart the S1 mutating fleet. S1 can emit legacy status values and does not enforce optimistic concurrency.

Preferred recovery order:

```text
fix/roll forward R5 deployment
    ↓
validate R5 runtime
    ↓
re-enable mutations
```

If an emergency application rollback to S1 is unavoidable, database rollback must be treated as an explicit operator procedure reviewed for the affected environment. Do not blindly remove version/state constraints while R5 writers may still be active.

## Rollback boundary

R5 migration changes the write contract. For that reason:

- database backup is the authoritative disaster-recovery boundary;
- normal application rollback is forward-fix after migration 006;
- mixed S1/R5 mutating writers are unsupported;
- a future expand/contract deployment mechanism may remove this maintenance-window requirement, but that is outside R5.

## Evidence to retain

Record:

- database/environment identifier;
- preflight status counts;
- migration start/end time;
- migration operator/system principal;
- applied migration version;
- post-migration status/version validation;
- RLS smoke-test result;
- optimistic-concurrency smoke-test result;
- deployed R5 commit SHA.

These records become operational evidence for later Audit/Release Gate integration.
