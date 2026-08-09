# AI Cloud 身份与 Principal 契约

> 状态：S0 Contract Freeze

## 1. 目标

统一定义 API、Worker、Repository、Policy、Audit、Tool Execution 与数据库访问使用的身份模型。核心原则：

> 缺少身份代表未认证，绝不代表系统权限。

## 2. Principal 模型

```text
Principal
  ├─ User
  ├─ ServiceAccount
  └─ System
```

每个 Principal 必须具有显式类型、稳定 Subject ID 和可信认证来源。

```yaml
principal:
  type: user | service_account | system
  subject_id: string
  tenant_id: string?
  project_id: string?
  roles: []string
  capabilities: []string
  authn_method: oidc | workload_identity | internal_system
  issuer: string
  session_id: string?
```

User 或 Tenant Scoped ServiceAccount 必须有 Tenant；Project Scoped Operation 还必须有 Project。System Principal 必须显式创建，不能通过缺少 Tenant Context 推断。

## 3. 信任建立流程

```text
External Caller
  -> Authentication
  -> Verified Claims
  -> Principal Resolution
  -> Tenant/Project Resolution
  -> Authorization
```

v0.1 可以暂时保留 Trusted Ingress Header，但前提是 Ingress 已完成认证并覆盖这些 Header。生产环境绝不能直接信任客户端自行提交的身份 Header。

## 4. Principal 类型

### User

通过 OIDC/OAuth2 认证的人类用户。Tenant Membership 与 Project Assignment 必须从可信身份数据解析，不能任意信任 Request Body。

### ServiceAccount

供 SDK、CI/CD、平台集成使用的机器身份。必须使用 Workload Identity、短期 JWT 或等价机器凭证，并具有显式 Scope/Capability。

### System

仅供平台内部维护、迁移、对账或 Controller Workflow 使用的内部身份。

System Access 必须同时满足：

- `principal.type = system`；
- 明确的 system subject；
- 明确 capability；
- 可审计 purpose/reason；
- 当前入口允许 System Principal。

## 5. Context 契约

Runtime Context 必须携带已验证的 Principal 对象。Domain Layer 不解析 HTTP Header 或 JWT Claim。

```go
type Principal struct {
    Type         PrincipalType
    SubjectID    string
    TenantID     string
    ProjectID    string
    Roles        []string
    Capabilities []string
    AuthnMethod  string
    Issuer       string
    SessionID    string
}
```

预期统一接口：

```text
WithPrincipal(ctx, principal)
PrincipalFromContext(ctx)
RequirePrincipal(ctx)
RequireTenant(ctx)
RequireProject(ctx)
RequireCapability(ctx, capability)
```

## 6. 授权分层

Authentication 与 Authorization 必须分离：

```text
Authentication
  -> Principal
  -> Resource Scope Check
  -> RBAC
  -> ABAC / Policy Engine
  -> Domain Operation
```

RBAC 负责粗粒度角色权限；Policy/ABAC 处理 Data Classification、Model License、Region、Cost、Tool Risk、Production Environment 等上下文约束。

## 7. System Access 规则

禁止：

```text
ctx 没有 tenant -> 允许 system access
subject 为空 -> 认为内部调用
header system=true -> 绕过授权
```

必须：

```text
Explicit System Principal
  + Explicit Capability
  + Explicit Authorized Entry Point
  + Audit Event
```

## 8. 数据库边界

生产数据库不能仅依赖应用设置的布尔变量决定管理员权限。建议分离数据库角色：

```text
aicloud_app_role       强制 RLS
aicloud_worker_role    强制 RLS
aicloud_admin_role     受控维护
aicloud_migration_role 仅 Schema Migration
```

Tenant Operation 通过 RLS Scoped Transaction 执行。管理员访问使用独立凭证/角色并产生 Audit Evidence。

## 9. 审计要求

所有 Mutating Operation 至少记录：

```text
principal_type
subject_id
tenant_id
project_id
roles/capability used
request_id
trace_id
resource
operation
decision
reason
```

System Principal 的所有活动必须可审计。

## 10. 验收条件

- 缺少 Identity 一律 Fail Closed；
- 缺少 Tenant 绝不获得提升权限；
- User、ServiceAccount、System 在代码和 Audit 中可明确区分；
- Project API 缺少 Project Scope 时失败；
- System Access 必须检查 Capability；
- Cross-Tenant Access 在 Domain Mutation 前被拒绝；
- Repository 与 DB Test 同时覆盖普通路径和管理路径。

## 11. 从当前 S1 Prototype 的迁移

当前 Trusted Header Scope Contract 可以短期保留，但必须先解析为 `Principal`，后续模块不能直接消费 Header。现有“无 Scope 即可信 System Context”的行为正式标记为 Deprecated，并必须在 S2 完成前移除。