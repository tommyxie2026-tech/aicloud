# Task Scope Identity 迁移运行手册

## 目的

用于把 S1 Prototype 从 `task_ownership` Bridge 迁移到 Task 自身持有的 `tenant_id`、`project_id`、`created_by` 字段。

本手册对应 `005_task_scope_identity.sql`。

## 数据库角色与连接

Runtime 与 Schema Migration 必须使用不同数据库连接：

```text
AICLOUD_DATABASE_URL
  -> API/Worker Runtime Role
  -> 禁止 SUPERUSER
  -> 禁止 BYPASSRLS
  -> 强制 RLS

AICLOUD_MIGRATION_DATABASE_URL
  -> Migration-only Role
  -> 只用于 Schema Migration
  -> API/Worker 不使用
```

API 进程不允许执行 Migration。在 PostgreSQL Runtime Mode 下配置 `AICLOUD_RUN_MIGRATIONS=true` 会被拒绝。

## 迁移前检查

应用 Migration 005 前，应停止或 Drain 仍可能按旧 Schema 创建 Task 的写流量。

检查每个现有 Task 是否都有 Ownership Bridge：

```sql
SELECT t.id
FROM tasks t
LEFT JOIN task_ownership o ON o.task_id = t.id
WHERE o.task_id IS NULL;
```

预期：0 行。

检查 Bridge Scope 是否完整：

```sql
SELECT task_id
FROM task_ownership
WHERE tenant_id IS NULL OR tenant_id = ''
   OR project_id IS NULL OR project_id = ''
   OR subject_id IS NULL OR subject_id = '';
```

预期：0 行。

任一查询返回记录时必须停止迁移。应根据权威业务记录或 Audit Evidence 修复 Ownership，禁止伪造默认 Tenant/Project。

## 执行 Migration

使用独立 Migration Connection 执行：

```bash
AICLOUD_MIGRATION_DATABASE_URL='<migration-dsn>' go run ./cmd/migrate
```

Migration 005 将依次：

1. 给 Task 增加可空 Scope 字段；
2. 仅在 Schema Migration 路径暂时关闭旧 Bridge RLS；
3. 从 `task_ownership` 回填 Task Scope；
4. 仍存在 Unscoped Task 时直接失败；
5. 将 Task Scope 字段设置为 `NOT NULL`；
6. 增加 Tenant/Project 索引；
7. 在 `tasks` 上启用并 FORCE RLS；
8. 删除 Prototype `aicloud.system_access` 绕过策略，改为严格 Tenant/Project RLS。

## 迁移后验证

验证 Task Scope 完整性：

```sql
SELECT count(*)
FROM tasks
WHERE tenant_id IS NULL OR tenant_id = ''
   OR project_id IS NULL OR project_id = ''
   OR created_by IS NULL OR created_by = '';
```

预期：`0`。

验证 RLS：

```sql
SELECT relname, relrowsecurity, relforcerowsecurity
FROM pg_class
WHERE relname IN ('tasks', 'task_ownership');
```

预期：两个表的两个 RLS Flag 都为 true。

使用 Runtime Connection 验证数据库角色：

```sql
SELECT current_user, rolsuper, rolbypassrls
FROM pg_roles
WHERE rolname = current_user;
```

Runtime Role 必须满足：`rolsuper=false`、`rolbypassrls=false`。

## Tenant Isolation 验证

通过两个独立 Project-scoped Principal Context 使用 Runtime Connection 验证：

```text
Tenant A / Project A
  -> 能读取 Task A
  -> 不能读取 Task B

Tenant B / Project B
  -> 能读取 Task B
  -> 不能读取 Task A
```

跨 Scope 的 Task 访问在 Application Boundary 必须表现为 Not Found。

## 回滚策略

Migration 005 属于 Security Hardening Migration，恢复业务流量后不应自动回退。

如果 Migration 在 Commit 前失败，Migration Transaction 会自动 Rollback；修复 Preflight 数据问题后重新执行即可。

如果 Migration 成功后出现 Application Regression，应优先修复 Scoped Repository/应用并前向发布，而不是删除 Task Identity 或关闭 RLS。只有紧急情况下才允许使用经过 Review 的维护流程做 Schema Rollback，并且必须保留已经回填的 Task Scope 与 Audit Evidence。

## 退出条件

只有同时满足以下条件才视为迁移完成：

- 所有现有 Task 都具有非空 Tenant/Project/CreatedBy；
- Task RLS 已启用并 FORCE；
- Runtime Role 不具备 SUPERUSER/BYPASSRLS；
- API/Worker 能使用 `AICLOUD_DATABASE_URL` 正常启动；
- Migration Command 仅使用 `AICLOUD_MIGRATION_DATABASE_URL`；
- Cross-Tenant/Cross-Project Verification 通过。
