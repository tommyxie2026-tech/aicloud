# API and Data Contracts

## API conventions
Base path: `/api/v1`. JSON uses snake_case. Timestamps are RFC3339 UTC. IDs are opaque strings. Writes accept `Idempotency-Key`; repeated requests with the same tenant/project/key and equivalent body return the original result. Conflicting body returns 409.

Trusted headers are produced/validated by gateway middleware; clients may send Authorization and Idempotency-Key but cannot forge internal tenant/subject headers.

## Error envelope

```json
{
  "error": {
    "code": "MODEL_NOT_ELIGIBLE",
    "message": "no model satisfies policy and capability requirements",
    "request_id": "req_...",
    "trace_id": "tr_...",
    "retryable": false,
    "details": {}
  }
}
```

Stable codes include INVALID_ARGUMENT, UNAUTHENTICATED, FORBIDDEN, NOT_FOUND, CONFLICT, IDEMPOTENCY_CONFLICT, POLICY_DENIED, APPROVAL_REQUIRED, BUDGET_EXCEEDED, MODEL_NOT_ELIGIBLE, PROVIDER_UNAVAILABLE, RATE_LIMITED, TASK_TERMINAL, INTERNAL.

## Minimum REST resources

```text
GET/POST   /api/v1/models
GET        /api/v1/models/{model_id}
POST       /api/v1/models/{model_id}/versions
POST       /api/v1/model-versions/{version_id}/admission

POST       /api/v1/tasks
GET        /api/v1/tasks/{task_id}
POST       /api/v1/tasks/{task_id}:cancel
POST       /api/v1/tasks/{task_id}:approve
GET        /api/v1/tasks/{task_id}/events
GET        /api/v1/tasks/{task_id}/cost

GET/POST   /api/v1/agents
GET/POST   /api/v1/tools
GET/POST   /api/v1/policies
GET/POST   /api/v1/projects
GET        /api/v1/audit-events
```

Task event streaming uses SSE at `/api/v1/tasks/{task_id}/events:stream` before WebSocket is considered.

## Create Task

```json
{
  "project_id": "prj_...",
  "agent_id": "agt_...",
  "goal": "scale dev-gpu-cluster gpu-workers from 3 to 6",
  "inputs": {},
  "constraints": {
    "max_cost_usd": "1.00",
    "deadline_seconds": 300,
    "data_classification": "internal"
  }
}
```

Response 202:

```json
{
  "task_id": "tsk_...",
  "trace_id": "tr_...",
  "status": "CREATED",
  "created_at": "..."
}
```

## Core PostgreSQL schema
All mutable tables include `created_at`, `updated_at`, and optimistic `version bigint`. Monetary values use `numeric`, never float.

### tenants
`id pk, organization_id, name, status, created_at, updated_at`

### projects
`id pk, tenant_id not null, name, status, default_policy_id, created_at, updated_at`; unique `(tenant_id,name)`.

### subjects
`id pk, tenant_id, type(user|service|agent), external_subject, status, metadata jsonb`; unique `(tenant_id,type,external_subject)`.

### models
`id pk, owner_tenant_id nullable, name, visibility(global|private|restricted), description, created_at, updated_at`.

### model_versions
`id pk, model_id, owner_tenant_id nullable, provider_id, provider_model_ref, version_ref, deployment_mode, capabilities jsonb, context_limits jsonb, pricing jsonb, residency jsonb, license jsonb, provenance jsonb, risk_level, lifecycle_state, admission_state, artifact_digest, created_at`; immutable after admission except operational state references.

### provider_endpoints
`id pk, tenant_id nullable, provider_type, endpoint_ref, region, credential_ref, config jsonb, enabled`; secrets are references only.

### tasks
`id pk, tenant_id, project_id, agent_id, subject_id, trace_id, idempotency_key, goal, input jsonb, constraints jsonb, status, result jsonb, failure_code, created_at, updated_at, version`; unique `(tenant_id,project_id,idempotency_key)` when key present.

### task_events
`id pk, tenant_id, project_id, task_id, sequence, event_type, payload jsonb, occurred_at`; append-only; unique `(task_id,sequence)`.

### route_decisions
`id pk, tenant_id, project_id, task_id, trace_id, request_hash, selected_model_version_id, selected_provider_endpoint_id, eligible_candidates jsonb, rejected_candidates jsonb, score_breakdown jsonb, fallback_chain jsonb, created_at`.

### tool_invocations
`id pk, tenant_id, project_id, task_id, tool_id, action, resource_ref, policy_decision_id, credential_lease_ref, input_hash, status, result_ref, started_at, ended_at`.

### approvals
`id pk, tenant_id, project_id, task_id, reason, risk_level, requested_by, decided_by, decision, expires_at, created_at, decided_at`.

### cost_events
`id pk, tenant_id, project_id, task_id, trace_id, source_type, source_id, provider_id, model_version_id, usage jsonb, currency, amount numeric(20,8), pricing_version, occurred_at`; append-only.

### audit_events
`id pk, tenant_id, project_id nullable, trace_id, subject_id, action, resource_type, resource_id, decision, metadata jsonb, occurred_at`; append-only.

### artifacts
`id pk, tenant_id, project_id, task_id, kind, object_key, digest, size_bytes, classification, created_at`.

## Required indexes
Every tenant table starts indexes with tenant_id where tenant-scoped queries dominate. Required examples: `(tenant_id,project_id,status,created_at desc)` on tasks; `(tenant_id,task_id,sequence)` on task_events; `(tenant_id,task_id,occurred_at)` on cost/audit; `(model_id,admission_state,lifecycle_state)` on model_versions.

## Row Level Security
Enable RLS for tasks, task_events, approvals, cost_events, audit_events, artifacts and tenant-private model metadata. Connection/session context sets the trusted tenant identifier. Application repository predicates remain mandatory; RLS is defense in depth.

## Event envelope
Internal durable events use:

```json
{
  "event_id": "evt_...",
  "event_type": "task.status.changed",
  "schema_version": 1,
  "occurred_at": "...",
  "tenant_id": "ten_...",
  "project_id": "prj_...",
  "task_id": "tsk_...",
  "trace_id": "tr_...",
  "producer": "workflow",
  "payload": {}
}
```

Events are at-least-once; consumers must be idempotent by event_id. Domain state changes and outbound durable events use transactional outbox when asynchronous publication is required.

## Migration rules
SQL migrations are forward-only in production. Destructive changes use expand -> migrate/backfill -> switch readers/writers -> contract. Schema migrations must be safe for rolling deployment.

## API compatibility
Within `/api/v1`, adding optional fields is compatible. Removing/renaming fields, changing semantics, or changing enum meaning is incompatible and requires a new version or migration window.