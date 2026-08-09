# ADR-0020：AI Cloud 多租户架构

## 状态
已接受

## 背景
AI Cloud 必须同时服务多个组织、租户、项目、用户、服务账号、Agent、模型、工具、工作流、预算、Trace 与制品，并保证不存在跨租户数据泄露和策略绕过。因此，多租户不是后续补充的 API 功能，而是平台级不变量。

## 决策
所有对外可寻址资源和所有执行记录都必须归属于 Tenant。Project 作为租户内部应用所有权、预算和资源管理的主要边界。Tenant Context 必须贯穿 API、Workflow、Model Routing、Tool Execution、Sandbox、Persistence、Cache、Telemetry、Cost Ledger 和 Artifact。

### 资源层级

```text
Organization
  └── Tenant
       ├── Project
       │    ├── Agent
       │    ├── Workflow
       │    ├── Task
       │    ├── Policy Binding
       │    └── Budget
       ├── Model Access Policy
       ├── Tool Access Policy
       └── Audit / Cost / Trace
```

### 强制身份上下文
每个外部请求和内部命令必须携带：

```text
request_id
trace_id
tenant_id
project_id
subject_id
subject_type
roles/scopes
```

当 `tenant_id` 可以由认证身份和路由上下文确定时，禁止直接信任请求 Body 中的 tenant_id。

### 数据库隔离
v0.1/v0.2 默认采用 PostgreSQL 共享 Schema + 强制 `tenant_id` + 复合索引 + Repository 层租户过滤；高价值表同时启用 PostgreSQL Row Level Security。当前不默认采用每租户独立数据库/Schema，避免在规模尚未需要时引入过高运维复杂度。

所有租户资源表必须包含 `tenant_id NOT NULL`；项目资源还必须包含 `project_id NOT NULL`。所有代表业务唯一性的 Unique Constraint 必须包含租户作用域。

### Cache 隔离
Redis Key 强制采用版本化租户前缀：

```text
aicloud:v1:{tenant_id}:{project_id}:{resource}:{key}
```

除非数据被明确分类为 Public 且不包含任何租户派生数据，否则禁止跨租户共享语义缓存和响应缓存。

### Artifact 隔离
对象存储路径必须包含 Tenant/Project，并在签发上传/下载凭据前执行授权：

```text
tenants/{tenant_id}/projects/{project_id}/tasks/{task_id}/...
```

### Runtime 隔离
Task 创建后，其 Tenant/Project Context 不可变。Workflow Activity 必须校验目标资源 Tenant 与 Task Tenant 一致。Sandbox 使用 Task Scope 的 Workload Identity、Namespace/Label Policy、NetworkPolicy、ResourceQuota 和短期凭据。

### 模型与工具访问
Model Registry 中的模型可定义为 Global、Tenant Private 或 Tenant Restricted。Router 在模型评分前必须先执行租户模型策略过滤。Tool Gateway 每次调用必须评估 Subject + Tenant + Project + Tool + Action + Resource + Risk。

### FinOps
所有 CostEvent 必须记录 tenant_id、project_id、task_id、trace_id、Provider/Model/Tool 维度，以及不可变的用量和金额字段。预算在执行前评估，可产生 Deny、Downgrade 或 Require Approval。

### Observability
Trace/Metric/Log 必须带受控的 Tenant/Project 属性。Secret、Prompt、检索文档和模型输出不能默认导出到共享可观测后端，Payload Capture 必须由策略控制。

## 授权模型
人类用户采用 OIDC；服务和 Agent 采用 Workload Identity。授权采用 RBAC + Contextual Policy：RBAC 管理粗粒度权限，Policy Engine 管理上下文决策。初始角色：tenant-admin、project-admin、developer、operator、auditor、viewer、service-agent。

## Fail-Closed 原则
Tenant Context 缺失或存在歧义时必须拒绝请求。发现跨租户不匹配时记录 Security Event。Fallback、Retry、Workflow Replay 和 Recovery 必须保持原始租户边界。

## 强制测试
实现只有在自动化测试覆盖以下隔离面后才能验收：API、Repository、Cache Key、Artifact、Trace/Cost、Tool Authorization、Sandbox Identity、Routing Policy。

## 影响
该设计会让 Tenant Context 进入多数 Domain Interface 和 Persistence Schema，但可以避免后期高成本安全改造，并为 SaaS、企业 Chargeback、委派管理和租户级治理建立基础。

## 实现映射
可直接开发的实现规范统一放在 `docs/implementation/`，包括组件边界、API/数据契约、Runtime/Security 流程、部署与测试，以及 v0.1 里程碑。