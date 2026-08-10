# Tenant Boundary 实现 Slice

## 状态

`agent/tenant-contract-slice` 已完成 S0 后的 R1-R4 合规改造，等待最终 CI、Security Review 与 Domain Review。

## 契约映射

本 Slice 落地 ADR-0020、Identity Contract、Database RLS Model 与 Task Aggregate Contract 的第一组可强制执行约束：

```text
Trusted Authenticated Ingress
  -> Explicit Principal
  -> Tenant / Project Scope
  -> Task Identity
  -> Scoped Repository
  -> PostgreSQL RLS
  -> Task-owned Evidence
```

## R1：显式 Principal

Runtime Context 统一使用 `identity.Principal`：

```text
Principal
  ├─ User
  ├─ ServiceAccount
  └─ System
```

Trusted Ingress Header 只负责构造经过验证的 Principal，Domain/Repository 不直接解析 Header。

外部 API 可以接受 User 或 ServiceAccount，但拒绝客户端/Ingress Header 声明的 System Principal。System Principal 只能由内部代码显式创建，并需要明确 Capability 与 Purpose。

## R2：缺少身份 Fail Closed

以下旧行为已移除：

```text
missing tenant / missing scope
        -> trusted system access
```

现在：

```text
missing Principal
        -> unauthenticated
```

Task Repository 对缺少 Principal 的访问一律失败。系统级 Task 访问必须使用显式 `PrincipalSystem` 和 `task:system-access` Capability。

## R3：PostgreSQL RLS 与数据库角色

生产 Runtime 使用 `ScopedPostgresTasks`：

```text
RequireProject Principal
  -> BeginTx
  -> set_config(aicloud.tenant_id, ..., true)
  -> set_config(aicloud.project_id, ..., true)
  -> SQL
  -> Commit/Rollback
```

Runtime 启动会校验当前 PostgreSQL Role：

- 不允许 Superuser；
- 不允许 `BYPASSRLS`。

原 Prototype 中 `aicloud.system_access=on` Session Flag 不再用于生产访问。

管理员访问采用独立 `AdminPostgresTasks` 入口：

- 独立 Admin DB Credential；
- DB Role 必须具备受控 RLS Bypass；
- Application Context 必须是显式 System Principal；
- 必须具备 `database:admin` Capability；
- 不通过 API Runtime 自动暴露该入口。

## R4：Task Ownership 原子化

Migration `005_task_scope_identity.sql` 将：

```text
tenant_id
project_id
created_by
```

直接加入 `tasks` 表，并从旧 `task_ownership` Bridge 回填。

Migration 在发现任何无法确定 Ownership 的 Task 时直接失败，不会生成虚假 Tenant/Project。

新 Task 创建路径：

```text
Verified Principal
  -> ScopedTasks binds Task identity
  -> single Task INSERT
```

因此不再存在：

```text
INSERT task
  -> INSERT task_ownership 失败
  -> orphan task
```

`task_ownership` 只保留为 Migration Bridge，不再是 Runtime Source of Truth。

## Repository 安全不变量

1. Task 的 `tenant_id`、`project_id`、`created_by` 创建后不可通过普通 Update 修改；
2. Cross-Tenant/Cross-Project Task 访问对外表现为 Not Found；
3. RouteDecision 与 CostEvent 继续通过 Task Repository 继承 Task Scope；
4. PostgreSQL RLS 是 Application Authorization 的第二层防御；
5. Global Model/Provider 资产不因为 Tenant Task 隔离而复制；Model Access 仍由 Policy/Router 控制。

## 当前验收测试

已增加/调整测试覆盖：

- Missing Principal Fail Closed；
- Trusted Ingress -> Principal Resolution；
- External System Principal Header 被拒绝；
- User/ServiceAccount Scope 传播；
- Task Tenant/Project/CreatedBy 原子绑定；
- Cross-Tenant Task Get/List 被拒绝；
- Task Identity Mutation 被拒绝；
- Explicit System Principal + Capability；
- Route/Cost Evidence 继承 Task Ownership；
- Migration 不包含 `aicloud.system_access` Bypass；
- Migration 强制 Task RLS 与 Tenant/Project Context。

## 下一步

R1-R4 通过 CI 与 Review 后，PR #12 才能从 Draft 进入 Ready。之后进入 R5-R7：Task Aggregate State Transition、TaskEvent/Outbox/Idempotency，以及 OpenAPI + OIDC/JWT + RBAC/ABAC 收敛。
