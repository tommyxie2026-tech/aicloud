# AI Cloud Runtime 与安全执行链

## 1. 端到端 Task 执行顺序

```text
1 认证请求
2 解析 Tenant / Project / Subject
3 Schema Validation + Idempotency Check
4 持久化 Task(CREATED) + TaskEvent
5 启动 Durable Workflow
6 加载 Agent/Workflow Version
7 Policy Pre-check + Budget Pre-check
8 Router 选择 ModelVersion + Fallback Chain
9 Agent 使用模型生成 Plan
10 对 Structured Plan 做 Schema + Domain Validation
11 对每个有副作用的 Step：
   a Resolve Tool
   b Input Schema Validation
   c Policy Decision
   d 必要时 Human Approval
   e Lease Short-lived Credential
   f Tool/Sandbox Execution
   g Output Filter
   h Audit + Cost + Trace
12 Final Result Validation
13 Task Cost Reconciliation
14 持久化 Terminal State
15 执行或调度 Evaluation
```

任何开发实现不得绕过这条主链直接让模型执行生产副作用。

## 2. Router Contract
RouteRequest 至少包含：Task Type、Required Capability、Data Classification、Region/Residency、Max Cost、Latency/SLA、Inference Effort Preference、Service Tier Preference、Tenant/Project Policy Reference。

候选资格判断必须先于评分，而且是 Boolean Gate。违反 Policy、License、Residency、Capability、Admission State、Health 或 Hard Budget 的模型直接淘汰，不能因为 Benchmark/Score 高重新进入候选。

v0.1 使用可解释的确定性评分：

```text
score = quality_weight * quality_score
      + reliability_weight * reliability_score
      + latency_weight * normalized_latency
      + cost_weight * normalized_cost
```

Weight 属于 Versioned Policy/Config，并写入 RouteDecision。没有生产数据前禁止直接引入黑盒 ML Router。

## 3. Fallback
Router 首次决策同时产生有界 Fallback Chain；每次真正调用前再次校验 Policy/Health/Budget。必须设置 Max Attempts 和 Total Deadline。

RateLimit、Timeout、ProviderUnavailable 等 Retryable Error 可以触发 Fallback；InvalidRequest、PolicyDenied、ContentRejected、Context/Schema Error 默认不自动换 Provider，除非错误分类明确允许。

Fallback 永远不能突破原始 Tenant、Security、License、Residency 和 Budget Hard Constraint。

## 4. Structured Output
Plan 和高影响 Model Output 优先使用 Provider 支持的 Schema-constrained Structured Output；不支持时必须 Parse 后通过 JSON Schema + Domain Validator。无效 Output 不能进入 Tool Gateway。

## 5. Policy Input

```text
Subject Identity + Roles
Tenant / Project
Task / Agent / Workflow Version
Model / Tool / Action / Resource
Data Classification
Risk Level
Requested Side Effect
Budget / Cost Estimate
Environment(dev/stage/prod)
Time / Region
```

Policy Output 必须包含 Decision、Reason Code、Constraint、Policy Version，以及可选 Approval Requirement。

## 6. Human Approval
Approval 是不可变 Decision。Workflow 进入 `WAITING_APPROVAL` 时不得长期持有 Credential。Approval 必须绑定 Scope、Expiry、Approver、Reason 和精确 Action/Resource Digest；如果待执行动作发生变化，原 Approval 自动失效并重新申请。

## 7. Credential 生命周期
Credential 只能在 Policy Allow 或 Approval 完成后获取；必须 Task Scope + Tool/Resource/Action Scope + Short-lived；直接交给 Adapter/Runtime，不进入 Model Context，不写入 Log/TaskEvent，使用结束后 Revoke 或自然 Expire。

## 8. Tool Gateway
每个 Tool Registry Entry 必须包含：Name、Version、Input/Output Schema、Owner、Risk Level、Allowed Environment、Credential Type、Side-effect Class、Timeout、Audit Requirement。

Side-effect Class：

```text
READ_ONLY
REVERSIBLE_WRITE
DESTRUCTIVE_WRITE
PRIVILEGED_ADMIN
```

未知 Class 默认 Deny。

## 9. Sandbox 安全基线
所有模型生成代码、Shell-like Operation 必须进入 Sandbox，不得在 API/Worker Host 直接执行。Kubernetes 最低安全 Profile：Dedicated ServiceAccount、runAsNonRoot、seccomp RuntimeDefault、Read-only RootFS、Drop Linux Capabilities、CPU/Memory/Ephemeral Storage Limit、ActiveDeadline、Default Network Deny、显式 Egress Allowlist、Task-scoped Workspace、禁止 hostPath、禁止 Docker Socket、禁止 Kubernetes Admin Token。

## 10. Prompt/Tool Injection 边界
Retrieved Document 和 Tool Output 均视为 Untrusted Data，不能授予权限，也不能覆盖 System/Policy。Tool Selection 受 Tool Registry Schema 和 Policy Engine 约束。高风险副作用必须存在独立于模型文本的 Deterministic Check。

## 11. Trace Hierarchy

```text
HTTP Request span
  Task span
    Workflow span
      AgentRun span
        RouteDecision span
        ModelCall span(s)
        ToolCall span(s)
          Policy span
          CredentialLease span
        Sandbox span(s)
        Validation span
        Evaluation span
```

Trace Attribute 默认只记录 ID 和 Operational Metadata，不记录 Secret 和敏感 Payload。

## 12. Cost Accounting
ModelCall、ToolCall、Sandbox、Workflow Runtime、Retry/Failure、Storage/Network 和可选 Human Review 都产生 CostEvent。Task 终态执行 Reconciliation。平台首要成本指标是：

```text
Cost per Successful Task
```

而不是单纯 Token Price。

## 13. Failure Semantics
所有 Error 都必须明确 `retryable`。Workflow 只自动重试幂等 Activity。有副作用的 ToolCall 必须携带 Idempotency Token 或提供 Reconciliation Strategy。遇到“执行结果未知”不能盲目重试，而进入 Reconciliation Step。

## 14. 第一个 Reference Scenario

```text
Goal: scale dev-gpu-cluster gpu-workers 3 -> 6

Model
 -> Structured ChangePlan

Validator
 -> 校验 target/from/to/risk/rollback

Policy
 -> environment=dev
 -> reversible write
 -> bounds check

Approval
 -> 仅在 Policy/Risk Threshold 要求时等待

Tool Gateway
 -> Kubernetes Scale Adapter

Credential Broker
 -> Task-scoped Credential

Controller
 -> 带 expected resourceVersion 执行 Patch

Validator
 -> Read Back Replicas

Task
 -> COMPLETED

Audit / Cost / Trace
 -> Reconciled
```

任何 Model Generated Command 均不得被直接执行。