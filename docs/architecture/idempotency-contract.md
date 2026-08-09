# AI Cloud Idempotency Contract

> Status: S0 Contract Freeze

## 1. Purpose

Prevent HTTP retry, workflow retry, provider fallback and tool retry from multiplying into duplicate business operations or external side effects.

## 2. Distinct Idempotency Layers

```text
API Command Idempotency
Workflow Activity Idempotency
Provider Attempt Identity
Tool/Side-effect Idempotency
Outbox Delivery Idempotency
```

These layers are related but not interchangeable.

## 3. Public Mutating API

Every public mutation requires `Idempotency-Key`, including at minimum:

```text
POST /tasks
POST /tasks/{id}:cancel
POST /tasks/{id}:approve
POST /tasks/{id}/...mutating-command
```

Idempotency scope is:

```text
tenant_id + project_id + operation + idempotency_key
```

## 4. Idempotency Record

```yaml
idempotency_record:
  tenant_id: string
  project_id: string
  operation: string
  key: string
  request_digest: string
  status: in_progress | completed | failed_retryable | failed_final
  resource_id: string?
  response_code: int?
  response_digest: string?
  response_payload: object?
  created_at: timestamp
  expires_at: timestamp
```

Required uniqueness:

```text
UNIQUE(tenant_id, project_id, operation, key)
```

## 5. Request Matching

Rules:

```text
same key + same canonical request digest
  -> return/replay same logical result

same key + different request digest
  -> 409 IDEMPOTENCY_CONFLICT
```

Canonical request digest excludes transport-only values such as request ID but includes all fields that change business intent.

## 6. Transaction Semantics

For Task creation:

```text
BEGIN
  reserve idempotency record
  create Task
  append TaskCreated event
  store resource/result reference
COMMIT
```

The idempotency record and business mutation must not be committed independently.

## 7. Concurrent Duplicate Requests

When two requests with the same key arrive concurrently, one becomes the owner of the operation. The other waits/polls or receives a stable in-progress response according to the API contract; it must not execute the business command twice.

## 8. Workflow Activity Idempotency

Every side-effecting activity has an `operation_id` derived from stable business identity, for example:

```text
hash(task_id + logical_step + proposal_digest + tool_id)
```

Workflow retries reuse the same operation identity unless policy explicitly starts a new logical attempt.

## 9. Provider Attempts

Provider model generation may legitimately be retried/fallbacked, so it is not globally idempotent. Every attempt gets a unique `model_attempt_id`, while the higher-level logical model operation has a stable operation ID.

Cost and trace records distinguish logical operation from physical attempts.

## 10. Tool Side Effects

Tool adapters must support one of:

1. native idempotency key passed to target API;
2. AI Cloud durable side-effect ledger checked before execution;
3. deterministic desired-state reconciliation where repeating the same operation is safe.

A non-idempotent tool without compensation or deduplication cannot be enabled for automatic retry.

## 11. Approval

Approval is idempotent on:

```text
task_id + proposal_digest + reviewer/action
```

Repeating the same approval returns the existing decision. A changed proposal digest requires a new approval.

## 12. Cancellation

Cancellation is idempotent. Repeated cancellation of an already cancelled/terminal Task returns its current stable state and does not create repeated external compensations.

## 13. Outbox Delivery

Each outbox message has a stable delivery idempotency key. At-least-once delivery requires downstream deduplication.

## 14. Retention

Idempotency retention must exceed the maximum client retry window and relevant workflow retry horizon. High-risk side-effect keys may require longer retention than ordinary API creation keys.

## 15. Acceptance Criteria

- Repeated identical Task creation returns the same Task identity.
- Same key with different body returns conflict.
- Concurrent identical commands execute once.
- Workflow restart does not duplicate tool side effects.
- Provider retries create distinct attempts but one logical operation.
- Approval and cancellation are idempotent.
- Outbox duplicate delivery is safe.
- Tests cover process crash between request receipt, DB commit and external delivery.