# Tenant Boundary 实现 Slice

## 状态

已在 `agent/tenant-contract-slice` 实现，等待 CI 与评审。

## 契约映射

本 Slice 落地 ADR-0020 与 v0.1 Implementation Contract 中第一组可强制执行的约束：

```text
Authenticated Request
  -> Tenant Scope
  -> Task API 的 Project Scope
  -> Scoped Task Repository
  -> Task-owned Route / Cost / Evidence
```

## 外部身份契约

v0.1 由认证 Ingress 注入并覆盖以下 Header：

```text
X-AICloud-Tenant-ID
X-AICloud-Project-ID
X-AICloud-Subject-ID
```

`/api/*` 必须具有 Tenant 和 Subject；`/api/v1/tasks*` 还必须具有 Project。Health 与 Readiness Endpoint 明确不进入该边界。

这些 Header 是 Trusted Ingress 的兼容机制，不是最终用户认证方案。生产 OIDC/JWT Verification 在下一个契约 Slice 实现。

## Repository 契约

`tenantrepo.ScopedTasks` 包装已有 `domain.TaskRepository`，不把 Provider 或 Storage 实现细节泄漏到 Domain Layer。

规则：

- 外部创建 Task 时绑定 Task -> Tenant/Project/Subject；
- Scoped Get/Update 必须匹配 Ownership；
- Scoped List 过滤其他 Tenant 的 Task；
- Foreign Task ID 返回 `repository.ErrNotFound`；
- 无 Scope Context 只保留给可信 Bootstrap/System Work。

Route Decision 与 Cost Event 同样通过 Task Ownership Wrapper 访问，因此直接 Service Call 也不能绕过 Task Boundary。

## PostgreSQL 契约

Migration `004_task_tenant_ownership.sql` 新增：

```text
task_ownership(
  task_id,
  tenant_id,
  project_id,
  subject_id,
  created_at
)
```

RLS 已启用并 FORCE。PostgreSQL Ownership Operation 使用 Transaction-local：

```text
aicloud.tenant_id
aicloud.system_access
```

Tenant Call 使用 `system_access=off`；可信内部调用必须显式使用 `system_access=on`。

## 安全不变量

1. Task Resource Dispatch 之前必须建立 Tenant Identity；
2. Task API 必须具有 Project Identity；
3. Route、Cost、Audit、Trace、Evaluation、Model Execution、Tool Execution 之前必须检查 Task Ownership；
4. Cross-tenant Authorization Failure 对外表现为 Not Found，降低 Resource Enumeration 风险；
5. 本 Slice 中 Model Provider 仍属于全局平台资产；Tenant Model Access Policy 由 Routing/Policy Layer 实现，而不是为每个 Tenant 复制 Model Record。

## 验收测试

必须覆盖：

- Unscoped API Request 被拒绝；
- Tenant Scope 正确传播；
- Health Endpoint 不受 Tenant Gate 影响；
- Cross-tenant Task Get/List 被拒绝；
- Trusted System Context；
- Route 与 Cost Evidence 继承 Task Ownership。

## 下一 Slice

Slice 2 将运行 API 与 OpenAPI 收敛：增加 OIDC/JWT Verification、RBAC、Idempotency、Stable Error Envelope、Task Schema/State Machine 收敛，以及可执行 OpenAPI Contract Test。
