# R6 Outbox Dispatcher Lease and Retry

> Status: Issue #23 implementation slice

## Purpose

Define the durable delivery side of the R6 Transactional Outbox without weakening tenant/project RLS or pretending transport is exactly-once.

## Delivery model

Business commands commit external delivery intent as `pending` Outbox rows. A dispatcher later processes those rows with at-least-once semantics:

```text
pending
  -> delivering (leased)
       -> delivered
       -> pending (retry with backoff)
       -> dead_letter
```

A dispatcher crash after the downstream side effect but before `MarkDelivered` may cause another delivery. The stable Outbox `idempotency_key` remains the downstream deduplication identity.

## Lease contract

Migration 008 adds:

```text
lease_owner
lease_expires_at
last_error
```

Only a `delivering` row may hold a lease. Pending, delivered and dead-letter rows must have no active lease.

`Lease()` uses `FOR UPDATE SKIP LOCKED`, allowing multiple workers in the same project scope to claim different messages safely. An expired `delivering` lease is reclaimable, which provides process-crash recovery.

Each successful claim increments `attempts`. Delivery ownership is checked when recording success or failure; a stale worker cannot complete a message after another worker has reclaimed it.

## Retry and dead letter

`FailDelivery()` records the latest error and either:

- returns the row to `pending` with caller-provided `available_at` backoff; or
- moves it to `dead_letter` when the configured maximum attempt count is reached.

Retry scheduling policy remains outside the repository so backoff algorithms can evolve independently from persistence semantics.

## Tenant and project boundary

The dispatcher repository is intentionally scoped. It requires an explicit project Principal and sets transaction-local tenant/project context before leasing or updating rows.

It does not introduce a runtime RLS bypass and does not scan every tenant from one application-controlled session. Cross-project scheduling must be provided by an explicitly reviewed orchestration mechanism in a later runtime slice.

## Current implementation evidence

- `db/migrations/008_outbox_dispatch_leases.sql`
- `db/migrations/r6_outbox_contract_test.go`
- `internal/repository/postgres_outbox.go`
- `internal/repository/postgres_outbox_integration_test.go`

The integration test covers active-lease exclusion, expired-lease recovery, stale-owner rejection, retry, dead-letter transition and successful delivery completion.

## Remaining boundary

This slice provides durable dispatcher persistence primitives. The worker process still needs a delivery adapter that maps destinations such as `workflow.start` to concrete consumers and passes the stable delivery idempotency key downstream.
