# R7C API 授权边界

## 状态

实施阶段：R7C RBAC + 属性感知的 API 授权。

这一层运行在 R7B Authentication 之后、公共 API Handler 之前。它不会替代 PostgreSQL RLS、Task Ownership Check、Tool Gateway Policy、Approval 或 Domain Policy。

## 运行时流程

```text
HTTP Request
    |
    v
Request / Trace Metadata
    |
    v
OIDC/JWT 或 Trusted Ingress Authentication
    |
    v
Verified identity.Principal
    |
    v
API Authorization
    |
    +-- RBAC：该角色是否可以调用这个 API Action？
    |
    `-- ABAC：已验证的 Tenant/Project 上下文是否满足资源 Scope？
    |
    v
Task Ownership / PostgreSQL RLS / Domain Policy
    |
    v
Handler
```

RBAC 与属性策略必须同时允许请求。下游 Policy 可以进一步限制已经允许的请求，但绝不能扩大 API Authorization 已经拒绝的权限。

## 内置角色

| 角色 | 定位 |
|---|---|
| `tenant_admin` | Tenant 管理，以及当前所有已映射公共 API Action，但仍受已验证 Scope 约束 |
| `project_admin` | Project 操作，并允许写入 Evaluation |
| `developer` | Project 读取、Task 创建/路由/模型执行以及 Tool 执行 |
| `operator` | 当前 v0.1 与 Developer Execution Set 对齐的运行操作 |
| `reviewer` | 读取权限并允许写入 Evaluation |
| `viewer` | 只读访问 |

没有角色时默认拒绝。

公共授权路径不接受 `system` Principal。内部 SystemPrincipal 操作必须通过独立可信入口，并满足显式 Capability/Purpose 契约。

## API Action

v0.1 API Boundary 把公共 Route 显式映射为稳定 Action，而不是直接基于任意 URL 字符串授权。当前 Action 包括 Model 读写、Admission 读写、Task 读取/创建/路由/模型执行、Route/Cost/Audit/Trace/Evaluation 读取、Evaluation 写入、Tool 读取以及 Tool 执行。

Route Map 是显式且 Fail Closed 的：

- 未知 `/api/` 路径在业务 Handler 执行前返回 `NOT_FOUND`；
- 已知资源上的不支持 Method 返回 `METHOD_NOT_ALLOWED`；
- Authorization Service 缺失返回 `AUTHORIZATION_NOT_CONFIGURED`；
- Authorization Evaluation 发生错误时返回可重试的 `AUTHORIZATION_UNAVAILABLE`；
- 明确拒绝返回 `FORBIDDEN`。

Health 与 Readiness Endpoint 继续位于公共认证 API 边界之外。

## Scope Policy

v0.1 ABAC Layer 根据已验证 Principal 检查声明的 API Resource Scope：

- Tenant Scope 要求存在 `tenant_id`；
- Project Scope 同时要求 `tenant_id` 和 `project_id`；
- 当 Policy 收到显式目标 Tenant/Project 时，它必须与已验证 Principal 完全一致；
- 不支持的 Resource Scope 属于配置错误。

对于具体 Task ID，API Policy 不是 Ownership 的最终事实来源。`taskScopeGuard`、Tenant-aware Repository 以及 PostgreSQL FORCE RLS 继续作为独立 Enforcement Boundary，并解析真实 Task Ownership。这样保留 Defense in Depth，避免 RBAC/ABAC 削弱存储隔离。

## 与 Domain Policy 的分离

`internal/authorization` 决定一个已认证调用方是否允许发起公共 API Operation。

`internal/policy` 继续处理 Tool Gateway Policy、Approval 等上下文相关的执行决策。两类契约明确分开：

```text
Authentication -> API RBAC/ABAC -> Domain/Tool Policy -> Approval -> Execution
```

API Authorization 回答“调用方能否请求这项操作”；Domain Policy 回答“在当前业务、风险和治理上下文下，这项操作是否允许执行”。

## Role 来源

Role 与 Principal Attribute 只能从已经验证的 `identity.Principal` 读取。在 OIDC 模式下，它们来自完成密码学验证后的 Claim。Authorization Layer 不解析 JWT，也不直接信任原始身份 Header。

## 测试门禁

R7C 必须覆盖：

- 合法 Role/Action 组合允许；
- Read-only Role 的 Mutation 被拒绝；
- 缺少 Role 时拒绝；
- Tenant/Project Attribute 不匹配时拒绝；
- 公共 SystemPrincipal 被拒绝；
- 当前每个公共 Endpoint 都具有 Route/Action Mapping；
- Unknown Path 与 Method Fail Closed；
- Authentication 必须先于 Authorization；
- 既有 Task Scope/RLS Test 必须继续通过。

R7D 将继续收敛 Request/Response Schema 和可执行 OpenAPI Contract。R7C 不改变已经冻结的 Task Aggregate 与 R6 Transaction Semantics。
