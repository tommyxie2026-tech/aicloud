# R6 Task Command API Idempotency

> Status: Issue #23 implementation slice

## Purpose

Connect the public Task creation API to the R6 transaction kernel so HTTP retries cannot create duplicate durable Tasks in the production PostgreSQL runtime.

## Public contract

`POST /api/v1/tasks` requires:

```text
Idempotency-Key: <client-stable-key>
```

The server parses the business request and computes a canonical SHA-256 digest from the normalized request structure. Transport-only metadata such as `X-Request-ID` is not part of the digest.

Rules:

```text
same tenant/project + operation + key + same digest
  -> replay the original logical Task result

same tenant/project + operation + key + different digest
  -> 409 IDEMPOTENCY_CONFLICT

missing Idempotency-Key
  -> 400
```

A successful replay may include:

```text
Idempotency-Replayed: true
```

## Production transaction

For PostgreSQL runtime repositories, Control Plane resolves the R6 `TaskCommandStore` and executes:

```text
BEGIN
  reserve command idempotency
  derive tenant/project/creator from Principal
  INSERT Task(CREATED, version=1)
  INSERT TaskCreated(sequence=1)
  INSERT workflow.start Outbox message
  complete Idempotency record
COMMIT
```

The workflow start is deliberately represented as durable Outbox intent rather than an immediate `engine.Start()` dual write. Dispatcher delivery is the next R6 slice.

## Repository exposure

The existing `TaskRepository` contract remains unchanged. Production scoped PostgreSQL Task repositories optionally expose `repository.TaskCommandStoreProvider`, which lets Control Plane use the R6 transaction kernel without binding the domain repository interface to PostgreSQL implementation details.

Development/in-memory repositories do not currently provide durable command idempotency. They retain the legacy creation path for local tests only and must not be treated as production evidence for R6 exactly-once business-command semantics.

## Current boundary

This slice closes public Task creation. Route/model transition APIs are not yet claimed as end-to-end idempotent because route decisions and higher-level command results still require convergence onto the transaction kernel.

## Acceptance evidence

- public Task creation rejects a missing `Idempotency-Key`;
- canonical request digest is stable for the same normalized business request;
- PostgreSQL atomic create/replay/conflict/rollback tests remain the durability evidence;
- workflow start intent is persisted in Outbox in the same transaction as Task creation;
- bilingual documentation remains synchronized.
