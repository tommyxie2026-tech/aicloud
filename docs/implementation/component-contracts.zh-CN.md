# AI Cloud 组件与接口契约

## 1. 架构方式
v0.1 使用 Modular Monolith。内部模块通过 Go Interface、Domain Command/Event 协作，禁止为了“微服务化”在同一进程内部使用 HTTP。PostgreSQL、Redis、Temporal、OPA、Kubernetes、模型 Provider 等全部属于外部 Adapter。未来拆服务时继续保持本文件定义的 Domain Contract。

## 2. 核心请求上下文

```go
type RequestContext struct {
    RequestID string
    TraceID   string
    TenantID  string
    ProjectID string
    SubjectID string
    Roles     []string
}
```

所有 Application Service 入口统一接收 `context.Context` 和可信 `RequestContext`。后者只能由 Authentication/Tenant Middleware 构造，Domain 不直接信任 HTTP Body 中的 Tenant 信息。

## 3. Model Provider

```go
type Provider interface {
    Name() string
    Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error)
    Stream(ctx context.Context, req GenerateRequest) (Stream, error)
    Health(ctx context.Context) HealthStatus
    Capabilities(ctx context.Context) ProviderCapabilities
}
```

`GenerateRequest` 必须 Provider Neutral，包含 ModelVersionID、消息/内容、Tool Declaration、Structured Output Schema、Inference Effort、Service Tier、Max Output、Timeout 和 Trace Metadata。OpenAI/Gemini/Claude/vLLM/SGLang SDK Type 只能存在于各自 Adapter 内。

Provider Error 统一归一化为：InvalidRequest、Authentication、Permission、RateLimited、QuotaExceeded、Timeout、Unavailable、ContextExceeded、ContentRejected、ProviderInternal、Cancelled。

## 4. Model Registry

```go
type ModelRegistry interface {
    GetVersion(ctx context.Context, rc RequestContext, id string) (ModelVersion, error)
    ListCandidates(ctx context.Context, rc RequestContext, q CandidateQuery) ([]ModelVersion, error)
    PutModel(ctx context.Context, rc RequestContext, model Model) error
    PutVersion(ctx context.Context, rc RequestContext, version ModelVersion) error
    SetAdmission(ctx context.Context, rc RequestContext, decision AdmissionDecision) error
}
```

ModelVersion 是生产路由的不可变实体，至少包含 Provider/Model Identity、Capability、Deployment Mode、Region/Residency、Pricing、License/Provenance、Risk、Health Reference、Lifecycle 和 Admission State。

## 5. Intelligent Router

```go
type Router interface {
    Route(ctx context.Context, rc RequestContext, req RouteRequest) (RouteDecision, error)
}
```

路由流水线固定为：

```text
加载候选
-> Capability Filter
-> Tenant/Model Policy Filter
-> License/Residency/Security Filter
-> Health/Quota/Capacity Filter
-> Budget Filter
-> 对剩余候选评分
-> 选择 Primary + 有界 Fallback Chain
-> 持久化选择/淘汰原因
```

Router **只做决策，不直接调用模型**。

## 6. Task Service

```go
type TaskService interface {
    Create(ctx context.Context, rc RequestContext, cmd CreateTask) (Task, error)
    Get(ctx context.Context, rc RequestContext, taskID string) (Task, error)
    Cancel(ctx context.Context, rc RequestContext, taskID string) error
    Approve(ctx context.Context, rc RequestContext, cmd ApprovalCommand) error
}
```

TaskService 负责请求校验、Idempotency、初始持久化和启动 Workflow。Task 创建后的 Durable State Transition 由 Workflow 管理。

## 7. Workflow Port

```go
type WorkflowEngine interface {
    StartTask(ctx context.Context, task Task) error
    SignalApproval(ctx context.Context, taskID string, decision ApprovalDecision) error
    CancelTask(ctx context.Context, taskID string) error
}
```

首选 Temporal Adapter，但 Domain 不允许引用 Temporal SDK Type。

## 8. Agent Runtime

```go
type AgentRuntime interface {
    Plan(ctx context.Context, run AgentRunRequest) (Plan, error)
    ExecuteStep(ctx context.Context, step PlanStep) (StepResult, error)
    Validate(ctx context.Context, run AgentRunRequest, result TaskResult) (ValidationResult, error)
}
```

Agent Runtime 可以通过 Port 请求 Model 和 Tool，但不能直接访问 Provider SDK、Credential、Shell、Kubernetes、GitHub 或业务数据库。

## 9. Policy Engine

```go
type PolicyEngine interface {
    Evaluate(ctx context.Context, input PolicyInput) (PolicyDecision, error)
}
```

Decision 统一为 Allow、Deny、RequireApproval、AllowWithConstraints。所有有副作用的操作遇到 Policy Error 时 Fail Closed。

## 10. Tool Gateway

```go
type ToolGateway interface {
    Invoke(ctx context.Context, rc RequestContext, req ToolRequest) (ToolResult, error)
}
```

强制执行顺序：

```text
Resolve Tool
-> Input Schema Validation
-> Policy
-> Optional Human Approval
-> Credential Lease
-> Invoke Adapter
-> Output Filter
-> Audit + Cost + Trace
```

## 11. Credential Broker

```go
type CredentialBroker interface {
    Lease(ctx context.Context, req CredentialRequest) (CredentialLease, error)
    Revoke(ctx context.Context, leaseID string) error
}
```

Credential 必须短期、Task Scope、Tool/Resource Scope；不得返回模型文本上下文，不得写入 TaskEvent。

## 12. Sandbox

```go
type SandboxRuntime interface {
    Create(ctx context.Context, spec SandboxSpec) (Sandbox, error)
    Execute(ctx context.Context, sandboxID string, cmd SandboxCommand) (SandboxResult, error)
    Collect(ctx context.Context, sandboxID string) ([]Artifact, error)
    Destroy(ctx context.Context, sandboxID string) error
}
```

默认安全基线：Network Deny、Non-root、Read-only Base FS、显式 Writable Workspace、CPU/Memory/Timeout Limit、Task Workload Identity。

## 13. Cost Ledger

```go
type CostLedger interface {
    Append(ctx context.Context, event CostEvent) error
    TaskSummary(ctx context.Context, rc RequestContext, taskID string) (CostSummary, error)
}
```

CostEvent Append-only，并以 CostEventID 保证幂等。

## 14. Audit Ledger

```go
type AuditLedger interface {
    Append(ctx context.Context, event AuditEvent) error
}
```

安全决策和副作用记录不可更新；需要修正时追加 Correction Event。

## 15. Evaluation

```go
type Evaluator interface {
    Evaluate(ctx context.Context, req EvaluationRequest) (EvaluationResult, error)
}
```

EvaluationResult 必须关联精确的 ModelVersion、Prompt/Agent/Workflow Version、Dataset Version、Policy Version 和 Trace Evidence。

## 16. Repository 强制规则
所有 Tenant Repository Method 都必须接收可信 Tenant Context，并在 SQL 中包含 Tenant Predicate。禁止为 Tenant Resource 暴露无作用域的 `GetByID(id)`。

## 17. 禁止依赖

```text
Domain -> HTTP                         禁止
Domain -> PostgreSQL/Redis SDK        禁止
Domain -> Temporal/OPA/K8s SDK        禁止
Agent -> Provider SDK                 禁止
Router -> Provider SDK                禁止
Provider Adapter -> Tenant DB         禁止
Tool Adapter 绕过 Policy              禁止
Sandbox 使用长期企业 Credential       禁止
```

该契约是后续直接编码和 Code Review 的依赖边界基线。