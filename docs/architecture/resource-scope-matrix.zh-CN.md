# AI Cloud 资源作用域矩阵

> 状态：S0 Contract Freeze  
> 目标：冻结每类核心资源的安全边界、拥有关系和授权入口，使数据库 Schema、Repository、RLS、API 和测试使用同一作用域模型。

## 1. 作用域层级

AI Cloud v0.1 使用四级资源作用域：

```text
Global
  ↓
Tenant
  ↓
Project
  ↓
Task
```

原则：

1. 子作用域资源必须继承父作用域身份；
2. 不允许通过资源 ID 绕过 Tenant/Project 边界；
3. 资源的作用域在创建后原则上不可变，跨作用域迁移必须走显式迁移流程；
4. 查询、事件、审计、成本和 Trace 必须保留作用域维度；
5. “无作用域”不代表系统权限。

## 2. Resource Scope Matrix

| Resource | Default Scope | Owner Key | Authorization Boundary | Notes |
|---|---|---|---|---|
| Provider | Global | provider_id | Platform Admin | Provider 只是连接/适配能力 |
| Model | Global / Tenant | model_id | Platform/Tenant Policy | 公共模型可全局，私有模型必须 Tenant scoped |
| ModelVersion | Same as Model | model_version_id | Inherit Model | 版本不可变 |
| ModelDeployment | Global / Tenant | deployment_id | Policy + Residency | 真实 Endpoint/Capacity 归 Deployment |
| AgentDefinition | Project | project_id | Project RBAC/Policy | Agent 不拥有真实凭证 |
| AgentVersion | Project | project_id | Inherit Agent | Immutable release artifact |
| Task | Project | tenant_id + project_id | Task Aggregate Root | 核心业务执行单位 |
| TaskEvent | Task | task_id | Inherit Task | Append-only |
| Approval | Task | task_id | Reviewer Policy | 不可跨 Task 复用 |
| RouteDecision | Task | task_id | Inherit Task | 必须保存约束/评分证据 |
| ModelAttempt | Task | task_id | Inherit Task | 每次 Provider 调用独立记录 |
| ToolDefinition | Global / Tenant | tool_id | Platform/Tenant Policy | 定义和执行分离 |
| ToolInvocation | Task | task_id | Tool Gateway | 所有副作用必须归属 Task |
| CredentialGrant | Task | task_id | Credential Broker | 短期、最小权限、不可暴露给 Agent |
| SandboxExecution | Task | task_id | Execution Policy | 必须关联 ToolInvocation |
| Policy | Global / Tenant / Project | scope_id | Policy Admin | 越具体的作用域可覆盖更宽作用域规则，但不能突破平台硬约束 |
| PolicyDecision | Task | task_id | Inherit Task | 保存 policy_version 和原因 |
| EvaluationDataset | Global / Tenant | dataset_id | Dataset Policy | 敏感数据集必须 Tenant scoped |
| EvaluationRun | Project / Task | project_id/task_id | Evaluation Policy | 生产 Task 评估优先绑定 Task |
| AdmissionEvidence | ModelVersion | model_version_id | Governance | License/Provenance/Security Evidence |
| CostEvent | Task | task_id | FinOps Policy | Immutable ledger event |
| AuditEvent | Tenant + Project | tenant_id + project_id | Audit Policy | 可选 task_id，但 Tenant/Project 必须存在 |
| Trace | Task / Request | trace_id | Observability Policy | 生产执行必须能关联 Task |
| Artifact | Project / Task | project_id/task_id | Data Policy | 输出 Artifact 默认继承 Task |

## 3. 数据模型不变量

对于 Project/Task 资源，数据库记录至少应包含：

```text
tenant_id
project_id
```

Task 子资源至少包含或可通过强外键确定：

```text
tenant_id
project_id
task_id
```

长期目标是把 Tenant/Project 作为核心表自身的不可变列，而不是仅依赖 side table。

## 4. 授权顺序

```text
Principal
  ↓
Resolve Tenant/Project
  ↓
Resource Scope Check
  ↓
RBAC
  ↓
ABAC / Policy
  ↓
Domain Operation
```

资源作用域检查属于硬约束，不能被 Router Score、业务参数或 Agent 推理覆盖。

## 5. Repository Contract

所有 Project/Task Repository 必须满足：

- scoped Get 不返回其他作用域资源；
- scoped List 永不混入其他 Tenant/Project 数据；
- Create 从可信 Context 获取作用域，而不是信任请求 Body 中任意 tenant_id；
- Update/Delete 首先验证作用域；
- 跨租户访问统一使用 Not Found 语义，降低 ID 枚举风险；
- 系统维护访问必须使用显式 System Principal/DB Role，而不是缺少 Scope。

## 6. 测试矩阵

每种 Tenant/Project/Task 资源至少覆盖：

1. Owner Tenant 可读写；
2. 同 Tenant、不同 Project 默认不可访问；
3. 不同 Tenant 不可访问；
4. List 不泄漏其他作用域数据；
5. ID 猜测不暴露资源存在性；
6. Event/Cost/Audit/Trace 继承正确作用域；
7. System/Maintenance 路径必须显式授权。

## 7. S0 Gate

任何新增资源在编码前必须先在本矩阵中定义：

```text
Scope
Owner Key
Authorization Boundary
Lifecycle
Cross-scope Migration Rule
```

未定义作用域的资源不得进入 v0.1 Public API。