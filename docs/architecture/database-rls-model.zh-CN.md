# AI Cloud PostgreSQL RLS 模型

> 状态：S0 Contract Freeze

## 1. 目标

定义数据库层 Tenant Isolation，作为应用层 Authorization 的第二道防线。RLS 不能替代 Identity、RBAC 或 Policy Engine，而是 Defense in Depth。

## 2. 安全原则

Application Code 不能通过设置一个不可信 Session Flag 就获得提升后的数据库权限。对于 Tenant-scoped Runtime Role，缺少 Tenant Context 必须视为 Error，而不是 Admin Access Signal。

## 3. Database Role

推荐生产角色：

```text
aicloud_app_role
  - API Query/Mutation
  - 强制 RLS
  - 无 BYPASSRLS

aicloud_worker_role
  - Workflow/Activity Query/Mutation
  - 强制 RLS
  - 无 BYPASSRLS

aicloud_admin_role
  - 受控 Maintenance/Reconciliation
  - 独立 Credential
  - 使用必须 Audit

aicloud_migration_role
  - 仅 Schema Migration
  - 不用于 Runtime
```

API/Worker Credential 不允许通过 SQL/Session Config 切换成 Admin Role。

## 4. Tenant Session Context

Tenant-scoped Transaction 在完成 Principal/Scope Verification 后设置：

```sql
SELECT set_config('aicloud.tenant_id', $1, true);
SELECT set_config('aicloud.project_id', $2, true);
```

第三个参数 `true` 表示 Transaction-local，避免 Connection Pool Reuse 把 Scope 泄漏到下一个 Request。

## 5. RLS Policy 形态

对于同时包含 Tenant/Project Column 的 Project-scoped Table：

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

Tenant-scoped Resource 可以只检查 tenant_id。Global Resource 使用独立访问路径；绝不能因为 Session 中没有 Setting 就自动变成全局可见。

## 6. FORCE RLS

Tenant/Project Business Table 应最终使用：

```sql
ALTER TABLE ... ENABLE ROW LEVEL SECURITY;
ALTER TABLE ... FORCE ROW LEVEL SECURITY;
```

Runtime Role 不能因为 Table Owner 身份静默绕过 Policy。

## 7. Administrative Access

Admin Workflow 使用独立 DB Credential/Role，同时要求 Application Layer 显式 System Principal Authorization。

不能通过以下方式提升：

```text
system_access=on
empty tenant ID
custom request header
```

当前 S1 中 `aicloud.system_access` Session Variable Bypass 仅视为 Prototype Bridge，必须在 S1/S2 Hardening 完成前从生产 RLS Model 中移除。

## 8. Schema Rule

长期 Tenant/Project Table 尽量直接保存：

```text
tenant_id NOT NULL
project_id NOT NULL   -- Project Resource
```

Task Ownership 应从 Side Table 最终迁移进 `tasks`。Task Child Table 可保存 `task_id` 加 Denormalized Tenant/Project Key，以增强 RLS 与 Audit，但必须用 Composite Foreign Key 或 Transaction Rule 保证一致性。

## 9. Repository Transaction Pattern

```text
BeginTx
  -> set verified tenant/project config
  -> execute scoped queries
  -> commit/rollback
```

不能在 Transaction 外依赖 Session-global Scope State。

Repository 可以继续显式添加 Scope Predicate，便于可读性和性能；RLS 是第二层保护。

## 10. Connection Pool Safety

必须验证：

1. Tenant A Transaction 看不到 Tenant B；
2. 同一个 Pooled Connection 被 Tenant B 复用时不会保留 Tenant A Scope；
3. Rollback/Failure 后 Transaction-local Setting 不残留；
4. App/Worker Role 无法 Bypass RLS；
5. 普通 Runtime Credential 无法切换到 Admin Role。

## 11. Migration Safety

给现有表增加 RLS 时需要：

- Backfill tenant/project column；
- 验证不存在 Orphan/Unscoped Row；
- 创建 Policy；
- Shadow/Test Query；
- Enable RLS；
- Force RLS；
- Rollback Plan。

在 Data Backfill 未验证前不能直接 FORCE RLS。

## 12. Failure Semantics

缺少 Required Tenant/Project Setting 时只能返回 Zero Row 或明确 Authorization/Data Access Error，绝不能暴露全部数据。

Database Authorization Failure 对外映射成稳定 Application Error，并避免泄露 Cross-tenant Resource 是否存在。

## 13. 验收条件

- Runtime App/Worker DB Role 无 BYPASSRLS；
- Application-controlled Boolean 不能把 Tenant Transaction 提升为 Admin；
- Migration 完成后 Tenant/Project Business Table 强制 RLS；
- Scope Setting 仅 Transaction-local；
- Connection Pool Reuse Isolation Test 通过；
- Repository 与 Raw SQL 两级 Cross-tenant Read/Write Test 通过；
- Admin Access 使用独立 Credential 并产生 Audit Evidence。