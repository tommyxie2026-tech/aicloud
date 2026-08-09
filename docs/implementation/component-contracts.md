# Component Contracts

## Architecture style
v0.1 is a modular monolith. Modules communicate through Go interfaces and domain commands/events, not through internal HTTP. External systems are adapters. A future service extraction must preserve these contracts.

## Core request context

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

All application-service entrypoints accept `context.Context` plus a trusted `RequestContext` produced by authentication/tenant middleware.

## Model Provider

```go
type Provider interface {
    Name() string
    Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error)
    Stream(ctx context.Context, req GenerateRequest) (Stream, error)
    Health(ctx context.Context) HealthStatus
    Capabilities(ctx context.Context) ProviderCapabilities
}
```

`GenerateRequest` is provider-neutral and contains ModelVersionID, messages/content, tool declarations, structured-output schema, inference effort, service tier, max output, timeout, and trace metadata. Adapters translate it to provider SDK types.

Provider errors are normalized into: InvalidRequest, Authentication, Permission, RateLimited, QuotaExceeded, Timeout, Unavailable, ContextExceeded, ContentRejected, ProviderInternal, Cancelled.

## Model Registry

```go
type ModelRegistry interface {
    GetVersion(ctx context.Context, rc RequestContext, id string) (ModelVersion, error)
    ListCandidates(ctx context.Context, rc RequestContext, q CandidateQuery) ([]ModelVersion, error)
    PutModel(ctx context.Context, rc RequestContext, model Model) error
    PutVersion(ctx context.Context, rc RequestContext, version ModelVersion) error
    SetAdmission(ctx context.Context, rc RequestContext, decision AdmissionDecision) error
}
```

A ModelVersion contains immutable provider/model identity plus capabilities, deployment mode, region/residency, pricing, license/provenance, risk, health reference, and lifecycle/admission state.

## Router

```go
type Router interface {
    Route(ctx context.Context, rc RequestContext, req RouteRequest) (RouteDecision, error)
}
```

Routing pipeline is fixed:

```text
load candidates
-> capability filter
-> tenant/model policy filter
-> license/residency/security filter
-> health/quota/capacity filter
-> budget filter
-> score eligible candidates
-> select primary + bounded fallback chain
-> persist decision reason
```

The Router never invokes a model directly.

## Task Service

```go
type TaskService interface {
    Create(ctx context.Context, rc RequestContext, cmd CreateTask) (Task, error)
    Get(ctx context.Context, rc RequestContext, taskID string) (Task, error)
    Cancel(ctx context.Context, rc RequestContext, taskID string) error
    Approve(ctx context.Context, rc RequestContext, cmd ApprovalCommand) error
}
```

TaskService owns validation, idempotency, initial persistence, and workflow start. Workflow owns durable state transitions after creation.

## Workflow Port

```go
type WorkflowEngine interface {
    StartTask(ctx context.Context, task Task) error
    SignalApproval(ctx context.Context, taskID string, decision ApprovalDecision) error
    CancelTask(ctx context.Context, taskID string) error
}
```

Temporal is the initial adapter. Domain code must not depend on Temporal SDK types.

## Agent Runtime

```go
type AgentRuntime interface {
    Plan(ctx context.Context, run AgentRunRequest) (Plan, error)
    ExecuteStep(ctx context.Context, step PlanStep) (StepResult, error)
    Validate(ctx context.Context, run AgentRunRequest, result TaskResult) (ValidationResult, error)
}
```

Agent Runtime can request model inference or tools through ports but cannot access provider SDKs, credentials, shell, Kubernetes, GitHub, or databases directly.

## Policy Engine

```go
type PolicyEngine interface {
    Evaluate(ctx context.Context, input PolicyInput) (PolicyDecision, error)
}
```

Decision: Allow, Deny, RequireApproval, AllowWithConstraints. Policy errors fail closed for side-effecting operations.

## Tool Gateway

```go
type ToolGateway interface {
    Invoke(ctx context.Context, rc RequestContext, req ToolRequest) (ToolResult, error)
}
```

Gateway order: resolve tool -> validate schema -> policy -> optional approval -> credential lease -> invoke adapter -> filter output -> audit/cost/trace.

## Credential Broker

```go
type CredentialBroker interface {
    Lease(ctx context.Context, req CredentialRequest) (CredentialLease, error)
    Revoke(ctx context.Context, leaseID string) error
}
```

Credentials are short-lived, task-scoped, tool/resource-scoped, never returned to model text context, and never persisted in Task events.

## Sandbox

```go
type SandboxRuntime interface {
    Create(ctx context.Context, spec SandboxSpec) (Sandbox, error)
    Execute(ctx context.Context, sandboxID string, cmd SandboxCommand) (SandboxResult, error)
    Collect(ctx context.Context, sandboxID string) ([]Artifact, error)
    Destroy(ctx context.Context, sandboxID string) error
}
```

Default policy: network deny, non-root, read-only base filesystem, explicit writable workspace, CPU/memory/time limits, task workload identity.

## Cost Ledger

```go
type CostLedger interface {
    Append(ctx context.Context, event CostEvent) error
    TaskSummary(ctx context.Context, rc RequestContext, taskID string) (CostSummary, error)
}
```

CostEvent is immutable and idempotent by CostEventID.

## Audit Ledger

```go
type AuditLedger interface {
    Append(ctx context.Context, event AuditEvent) error
}
```

Security-relevant decisions and side effects are append-only. Corrections are new events, not updates.

## Evaluation

```go
type Evaluator interface {
    Evaluate(ctx context.Context, req EvaluationRequest) (EvaluationResult, error)
}
```

Evaluation results reference exact model version, prompt/agent/workflow version, dataset version, policy version, and trace evidence.

## Repository rule
Every tenant repository method receives trusted tenant context and includes tenant predicates. Repository interfaces must not expose unscoped `GetByID(id)` for tenant-owned objects.

## Dependency rule
Forbidden dependencies: domain -> HTTP; domain -> PostgreSQL/Redis/Temporal/OPA/Kubernetes SDK; agent -> provider SDK; router -> provider SDK; model provider -> tenant database; tool adapter -> policy bypass; sandbox -> long-lived enterprise credentials.