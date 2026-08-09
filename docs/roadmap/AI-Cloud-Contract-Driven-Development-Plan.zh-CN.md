# AI Cloud 契约驱动开发计划

> 状态日期：2026-08-09  
> 范围：v0.1 具备生产形态的 MVP  
> 规则：架构文档定义意图，Implementation Contract 定义代码，测试定义完成。

## 1. 交付模型

AI Cloud v0.1 继续采用 Go 1.22 Modular Monolith，并保持 API 与 Worker 两种进程角色。不会为了对应架构 Plane 而过早拆分微服务。

每一个开发 Slice 在适用时必须同步更新：

1. Domain Contract；
2. API/OpenAPI Contract；
3. Persistence/Migration Contract；
4. Runtime/Security Flow；
5. Tests 与验收证据；
6. 同一个 PR 中的英文和简体中文文档。

只有 PR Head 上的 `gofmt`、`go test ./...`、`go vet ./...`、入口程序构建和对应契约测试全部通过，Slice 才算完成。

## 2. 关键开发路径

```text
S1 Tenant Boundary
 -> S2 API Contract Convergence
 -> S3 Durable Task Workflow
 -> S4 Governed Execution
 -> S5 Trace + Cost + Audit
 -> S6 Reliable Model Routing
 -> S7 Evaluation Release Gates
 -> S8 End-to-End Developer Scenario
```

接入多少 Provider 不是里程碑。能够完整、可恢复、受治理地执行一个 Task 才是里程碑。

## 3. Slice 1 —— Tenant Boundary 与 Task Ownership

### 目标

在继续扩大 Agent 自主执行能力之前，把 Tenant/Project Context 建立为外部 API 的平台不变量，并阻止跨租户 Task 访问。

### 实现内容

- `internal/tenant`：请求 Scope 契约；
- `internal/httpapi`：Trusted Ingress Tenant Middleware；
- `internal/tenantrepo`：Tenant Scoped Task Repository Decorator；
- PostgreSQL `task_ownership` 表，绑定 Tenant/Project/Subject；
- `task_ownership` 启用 PostgreSQL RLS，并使用 Transaction-local Tenant Setting；
- Task Subresource Guard，使 Route、Cost、Audit、Trace、Evaluation、Model Execution、Tool Execution 自动继承 Task Ownership；
- `/healthz` 和 `/readyz` 不进入 Tenant Gate。

### 验收条件

- API 请求缺少已认证 Tenant/Subject Context 时 Fail Closed；
- Task API 必须具有 Project Context；
- Tenant A 无法读取或列出 Tenant B 的 Task；
- 跨租户 Task ID 返回 Not Found，不泄露授权细节；
- PostgreSQL Ownership 读写使用事务级 RLS Context；
- Bootstrap 和维护场景保留可信 Internal/System Context。

### 明确边界

v0.1 当前使用由认证 Ingress 注入的 Trusted Identity Headers。客户端自行提供这些 Header 不是生产认证机制。OIDC/JWT Verification 与 RBAC 在 Slice 2 实现。

## 4. Slice 2 —— API Contract Convergence

### 目标

让运行中的 API 与 `docs/implementation/contracts/openapi-v1.yaml` 收敛，同时保留一个短期兼容窗口。

### 开发包

- OIDC/JWT Verifier Interface 与生产实现；
- RBAC：Tenant Admin、Project Admin、Developer、Operator、Reviewer、Service Account；
- 稳定 `ErrorEnvelope`：Request ID、Trace ID、Error Code、Retryable；
- 所有 Mutating API 强制 `Idempotency-Key`；
- Task Schema 收敛：Tenant、Project、Agent、Goal/Input、Status、Version；
- Task 状态机：CREATED -> PLANNING -> EXECUTING -> WAITING_APPROVAL -> VALIDATING -> Terminal State；
- Pagination 与 Resource Version；
- OpenAPI Contract Tests。

### 退出条件

所有 Public Handler 都不能接受未文档化的请求结构，所有已声明的 v0.1 API 都具备可执行契约测试。

## 5. Slice 3 —— Durable Task Workflow

### 目标

使用可恢复执行替换当前 No-op Workflow Seam。

### 开发包

- 在 `workflow.Engine` 后接入 Temporal Client/Worker Adapter；
- Planning、Routing、Model Call、Policy、Approval、Tool Execution、Validation 的 Deterministic Workflow；
- Durable Retry、Timeout、Cancellation、Resume；
- 每次状态转换写入 Append-only Task Event；
- Idempotent Activity 与 Replay Test。

### 退出条件

API 或 Worker 重启后，同一个 Task 能继续执行，并且不会重复产生外部副作用。

## 6. Slice 4 —— Governed Execution

### 目标

把已有 Tool Gateway 与 Sandbox Planning Foundation 变成受控的真实执行路径。

### 开发包

- Kubernetes Job Create/Watch/Collect/Destroy Executor；
- Namespace 与 ServiceAccount 隔离；
- Network Default Deny；
- Task/Tool 级短期 Credential；
- OPA Policy Adapter 进入生产路径；
- Human Approval 与 Durable Workflow Pause/Resume 集成；
- Signed Workspace Input 与 Controlled Artifact Output。

### 退出条件

Agent 不能绕过 `Tool Gateway -> Policy -> Credential -> Sandbox/Adapter` 直接访问企业资源。

## 7. Slice 5 —— Trace、Cost 与 Audit 完整化

### 目标

让每一个成功或失败 Task 都可以完整重建，同时能够计算真实经济成本。

### 开发包

- OpenTelemetry SDK 与 OTLP Export；
- Request -> Task -> Workflow -> Agent -> Model/Tool/Sandbox/Evaluation Span Hierarchy；
- Trace、Audit、Cost Evidence 增加 Tenant/Project Dimension；
- Model、Tool、Workflow、Sandbox、Storage/Network、Retry、Human Review 的 Immutable Cost Event；
- Cost Reconciliation 与 `Cost per Successful Task`。

### 退出条件

只通过 Task ID 或 Trace ID 即可重建 Decision、Action、Failure、Retry 和 Total Cost，无需人工翻应用日志。

## 8. Slice 6 —— Reliable Model Routing

### 目标

让 Provider Independence 在真实故障和负载条件下成立。

### 开发包

- Redis-backed Shared Circuit Breaker；
- Provider Health/Quota/Capacity Collector；
- Latency、Queue、Residency、Budget、Evaluation Routing Input；
- Bounded Retry 与 Fallback；
- Explainable Selection/Rejection Evidence；
- Commercial API 与 Private vLLM/SGLang 统一走内部协议。

### 退出条件

Primary Provider 故障或 Quota Exhaustion 时能够执行 Policy-compliant Fallback，不出现无限重试或跨租户泄露。

## 9. Slice 7 —— Evaluation Release Gates

### 目标

把已经记录的 Evaluation Evidence 升级为可执行的发布策略。

### 开发包

- Versioned Golden Dataset；
- Model/Prompt/Workflow Regression Matrix；
- Quality、Safety、Reliability、Latency、Cost、Human Intervention Threshold；
- Release Gate 与 Rollback Decision；
- Production Trace Sampling 进入 Evaluation Candidate。

### 退出条件

Model、Prompt 或 Workflow Version 未通过要求的 Regression Gate 时不能进入生产。

## 10. Slice 8 —— Developer End-to-End Scenario

参考场景：

```text
scale dev-gpu-cluster gpu-workers from 3 to 6
```

必须完整经过：

```text
API
 -> Authenticated Tenant/Project
 -> Task Persistence
 -> Durable Workflow
 -> Router
 -> Provider
 -> Structured ChangePlan
 -> Validator
 -> Policy
 -> Human Approval when required
 -> Tool Gateway
 -> Short-lived Credential
 -> Kubernetes/Fake Adapter
 -> Read-back Validation
 -> COMPLETED
```

必须产生：

- Task Events；
- Route Decision；
- Model Attempts；
- Policy Decision；
- Approval Record；
- Tool Invocation；
- Audit Events；
- Cost Events；
- OpenTelemetry Trace；
- Evaluation Result。

这个场景作为 AI Cloud v0.1 的 Definition of Done。

## 11. Pull Request 规则

每个 Slice 使用独立 Branch，并以 Draft PR 进入主线评审。实现代码不直接提交到 `main`。只有验收条件和 CI Gate 全部通过后，PR 才进入 Ready 状态。英文文档与 `.zh-CN.md` 必须在同一个 PR 中保持同步。
