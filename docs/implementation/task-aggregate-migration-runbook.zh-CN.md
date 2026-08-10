# R5 Task Aggregate 迁移运行手册

## 状态

本手册是部署 `006_task_aggregate_state.sql` 时必须执行的运行流程。

## 为什么需要受控切换

R5 引入的不只是数据库新字段，而是一套新的并发写契约。S1 Writer 不理解 Task `version`，仍可能写入旧的 `PENDING` / `RUNNING` 状态；Migration 006 之后，数据库开始约束标准 R5 状态，而 R5 Repository 开始依赖乐观并发控制。

因此：

> S1 Mutating Writer 与 R5 Mutating Writer 不能同时对同一个数据库执行 Task 写操作。

Migration 006 是一次 **受控 Contract Cutover**，不是零停机 Expand Migration。

## 迁移前置条件

迁移前必须满足：

1. PR #22 / R5 已通过 Unit Test、PostgreSQL Integration Test、Vet 与 Build Gate；
2. 已按照环境恢复策略准备数据库 Backup/Snapshot；
3. 历史 Task 已通过 Migration 005 获得合法 Tenant/Project Identity；
4. 数据库中不存在未知 Task Status；
5. Migration 使用独立 Migration Credential，Runtime API Credential 不参与 Schema Change。

建议执行：

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

R5 迁移前通常应看到 `PENDING`、`RUNNING`、`COMPLETED`、`FAILED`，以及通过受控维护引入的其他终态。发现未知状态必须先处理，不能直接继续迁移。

## Cutover 顺序

```text
1. 停止接收新的 Task Mutation Request
2. Drain S1 API / Worker Task Writer
3. 确认没有仍在运行的 S1 Task Mutation
4. 使用 Migration Credential 执行 cmd/migrate
5. 验证 Migration 006 不变量
6. 部署 R5 API / Worker Fleet
7. 执行 Tenant/Project 与并发 Smoke Test
8. 重新开放 Task Mutation
```

Writer Drain 期间，Health/Readiness 等只读检查可以继续保留。

## 执行 Migration

必须使用：

```text
AICLOUD_MIGRATION_DATABASE_URL
```

禁止使用 Runtime：

```text
AICLOUD_DATABASE_URL
```

执行 Schema Migration。

Migration 006 会完成：

- `PENDING -> CREATED`；
- `RUNNING -> EXECUTING`；
- `version=1` Backfill，并设置 NOT NULL；
- 为历史 Terminal Task 补齐 `completed_at`；
- 建立标准 Task Status Constraint；
- 创建 Task Scope/Status Index。

## Migration 后验证

执行：

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

后两个结果必须都是：

```text
0
```

同时需要确认：

- Version Constraint 存在；
- Task Status Constraint 存在；
- Scope/Status Index 存在；
- S1 已验证过的 Task RLS Cross-Tenant Isolation Smoke Test 继续通过。

## Application Smoke Test

使用经过认证并带 Project Scope 的 Principal：

```text
Create Task
  ↓
CREATED / version=1

Route Task
  ↓
PLANNING
  ↓
ROUTING
  ↓
version 持续增加

Execute deterministic/mock model
  ↓
EXECUTING
  ↓
VALIDATING
  ↓
COMPLETED
  ↓
每一次持久化 Mutation 都增加 version
```

随后在 Integration/Smoke 环境故意提交一次旧版本 Task Update，必须得到：

```text
repository.ErrVersionConflict
```

而不是覆盖更新后的 Task。

## 故障处理

### Migration 在 Commit 前失败

`cmd/migrate` 会以事务方式执行每一个 Migration。确认该 Migration 未写入 `schema_migrations` 后，修复问题并重新执行。

### Migration 成功，但 R5 Application 部署失败

**不要直接重新启动 S1 Mutating Fleet。**

原因是 S1：

- 仍可能写入旧状态；
- 不执行 Task Version Optimistic Concurrency。

优先恢复路径：

```text
修复 / Roll Forward R5
        ↓
验证 R5 Runtime
        ↓
重新开放 Mutation
```

如果极端情况下必须回滚到 S1，则数据库回滚必须作为单独的 Operator Procedure 评审和执行。在仍可能存在 R5 Writer 的情况下，禁止直接删除 Version/Status Constraint。

## Rollback Boundary

Migration 006 改变了 Task Write Contract，因此正式冻结：

- Database Backup 是灾难恢复的最终边界；
- Migration 006 完成之后，常规 Application Recovery 策略是 **Forward Fix**；
- 不支持 S1/R5 Mixed Mutating Writer；
- 如果未来需要真正 Zero-Downtime Rolling Upgrade，应通过独立 Expand/Contract Migration 机制实现，不在 R5 范围内完成。

## 必须保留的迁移证据

每次生产/准生产迁移至少保留：

```text
Database / Environment ID
Preflight Status Count
Migration Start / End Time
Migration Operator / System Principal
Applied Migration Version
Post-Migration Status Validation
Post-Migration Version Validation
RLS Smoke Test Result
Optimistic Concurrency Smoke Test Result
Deployed R5 Commit SHA
```

这些记录后续会进入 Audit Evidence 与 Release Gate 体系。
