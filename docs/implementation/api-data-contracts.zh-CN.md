# AI Cloud API 与数据契约

## 1. API 通用规范
Base Path 固定为 `/api/v1`；JSON 使用 `snake_case`；时间使用 RFC3339 UTC；ID 为 Opaque String。所有 Write API 支持 `Idempotency-Key`：同一 Tenant/Project/Key 且 Body 等价时返回第一次结果；Body 冲突返回 HTTP 409。

客户端只能直接提供 Authorization、Idempotency-Key 等公开 Header。Tenant/Subject 等内部可信上下文必须由 Gateway Middleware 根据认证结果构造，禁止客户端伪造。

## 2. 统一错误 Envelope

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

首批稳定 Error Code：INVALID_ARGUMENT、UNAUTHENTICATED、FORBIDDEN、NOT_FOUND、CONFLICT、IDEMPOTENCY_CONFLICT、POLICY_DENIED、APPROVAL_REQUIRED、BUDGET_EXCEEDED、MODEL_NOT_ELIGIBLE、PROVIDER_UNAVAILABLE、RATE_LIMITED、TASK_TERMINAL、INTERNAL。

## 3. v0.1 最低 REST API

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

Task 实时事件 v0.1 优先采用 SSE：`/api/v1/tasks/{task_id}/events:stream`，暂不因为“实时”直接引入 WebSocket 复杂度。

## 4. Create Task Contract

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

返回 HTTP 202：

```json
{
  "task_id": "tsk_...",
  "trace_id": "tr_...",
  "status": "CREATED",
  "created_at": "..."
}
```

## 5. PostgreSQL 核心 Schema
所有可变表统一包含 `created_at`、`updated_at`、Optimistic `version bigint`。金额统一使用 `numeric`，禁止 Float。

### tenants
`id pk, organization_id, name, status, created_at, updated_at`

### projects
`id pk, tenant_id not null, name, status, default_policy_id, created_at, updated_at`；唯一键 `(tenant_id,name)`。

### subjects
`id pk, tenant_id, type(user|service|agent), external_subject, status, metadata jsonb`；唯一键 `(tenant_id,type,external_subject)`。

### models
`id pk, owner_tenant_id nullable, name, visibility(global|private|restricted), description, created_at, updated_at`。

### model_versions
`id pk, model_id, owner_tenant_id nullable, provider_id, provider_model_ref, version_ref, deployment_mode, capabilities jsonb, context_limits jsonb, pricing jsonb, residency jsonb, license jsonb, provenance jsonb, risk_level, lifecycle_state, admission_state, artifact_digest, created_at`。

ModelVersion Admission 后除 Operational State Reference 外保持不可变。

### provider_endpoints
`id pk, tenant_id nullable, provider_type, endpoint_ref, region, credential_ref, config jsonb, enabled`。数据库只保存 Secret Reference，不保存明文 Secret。

### tasks
`id pk, tenant_id, project_id, agent_id, subject_id, trace_id, idempotency_key, goal, input jsonb, constraints jsonb, status, result jsonb, failure_code, created_at, updated_at, version`；存在 Key 时唯一 `(tenant_id,project_id,idempotency_key)`。

### task_events
`id pk, tenant_id, project_id, task_id, sequence, event_type, payload jsonb, occurred_at`；Append-only；唯一 `(task_id,sequence)`。

### route_decisions
`id pk, tenant_id, project_id, task_id, trace_id, request_hash, selected_model_version_id, selected_provider_endpoint_id, eligible_candidates jsonb, rejected_candidates jsonb, score_breakdown jsonb, fallback_chain jsonb, created_at`。

### tool_invocations
`id pk, tenant_id, project_id, task_id, tool_id, action, resource_ref, policy_decision_id, credential_lease_ref, input_hash, status, result_ref, started_at, ended_at`。

### approvals
`id pk, tenant_id, project_id, task_id, reason, risk_level, requested_by, decided_by, decision, expires_at, created_at, decided_at`。

### cost_events
`id pk, tenant_id, project_id, task_id, trace_id, source_type, source_id, provider_id, model_version_id, usage jsonb, currency, amount numeric(20,8), pricing_version, occurred_at`；Append-only。

### audit_events
`id pk, tenant_id, project_id nullable, trace_id, subject_id, action, resource_type, resource_id, decision, metadata jsonb, occurred_at`；Append-only。

### artifacts
`id pk, tenant_id, project_id, task_id, kind, object_key, digest, size_bytes, classification, created_at`。

## 6. 必须建立的索引
Tenant Table 的高频查询索引必须以 tenant_id 开头。至少包括：Tasks `(tenant_id,project_id,status,created_at desc)`；TaskEvents `(tenant_id,task_id,sequence)`；Cost/Audit `(tenant_id,task_id,occurred_at)`；ModelVersions `(model_id,admission_state,lifecycle_state)`。

## 7. Row Level Security
对 tasks、task_events、approvals、cost_events、audit_events、artifacts、Tenant Private Model Metadata 启用 PostgreSQL RLS。连接/事务 Session 设置可信 Tenant ID。Repository SQL 仍必须显式包含 Tenant Predicate，RLS 只作为 Defense in Depth，不能代替应用隔离。

## 8. Internal Event Envelope

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

事件语义采用 At-least-once，Consumer 必须按 event_id 幂等。需要异步发布时，Domain State Change 与 Durable Event 使用 Transactional Outbox 保证一致性。

## 9. Migration 规则
Production SQL Migration Forward-only。破坏性 Schema Change 使用：

```text
Expand
-> Backfill/Migrate
-> Switch Reader/Writer
-> Contract
```

Migration 必须兼容 Rolling Deployment。

## 10. API 兼容规则
`/api/v1` 内新增 Optional Field 属于兼容变更；删除/重命名字段、改变字段语义或 Enum 含义属于不兼容变更，必须进入新 API Version 或明确 Migration Window。