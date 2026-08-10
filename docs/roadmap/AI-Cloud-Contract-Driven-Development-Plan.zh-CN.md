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
5. Failure/Idempotency Model；
6. Observability/Cost Evidence；
7. Tests 与验收证据；
8. 同一个 PR 中的英文和简体中文文档。

只有 PR Head 上的 `gofmt`、`go test ./...`、`go vet ./...`、入口程序构建和对应 Contract Test 全部通过，Slice 才算完成。

## 2. 关键开发路径

```text
S0 Architecture & Contract Freeze
 -> S1 Tenant / Identity Boundary
 -> S2 API + Domain Contract Convergence
 -> S3 Durable Task Workflow
 -> S4 Governed Execution
 -> S5 Trace + Cost + Audit
 -> S6 Reliable Model Routing
 -> S7 Evaluation Release Gates
 -> S8 End-to-End Developer Scenario
```

接入多少 Provider 不是里程碑。能够完整、可恢复、受治理、可审计且可核算成本地执行一个 Task 才是里程碑。

## 3. S0 —— Architecture & Contract Freeze

### 目标

在继续 Coding 前消除关键歧义，冻结 Domain、Identity、Tenant Scope、Task、Event、Workflow、RLS、Provider/Model Separation、Idempotency、Trace、Policy/Router、Audit、Cost、Evaluation 等平台不变量。

### Frozen Contract Pack

见 `docs/architecture/S0-Contract-Freeze.zh-CN.md` 与 `pre-code-architecture-gate.zh-CN.md`。

### 退出条件

- 20/20 Architecture Contract Category 已定义；
- 中英文 Contract Pack 完整；
- S1 Merge Blocker 明确；
- Implementation 只能按照批准后的 Remediation Order 恢复。

### S0 后 Remediation Order

```text
R1 Explicit Principal Model
 -> R2 Remove No-scope System Behavior
 -> R3 DB Role / RLS Hardening
 -> R4 Atomic Task Scope Persistence
 -> R5 Task Aggregate / State Transition
 -> R6 TaskEvent + Outbox + Idempotency
 -> R7 OpenAPI / OIDC / RBAC / ABAC Convergence
```

R1-R4 必须在当前 S1 PR Ready for Merge 前完成。

## 4. S1 —— Tenant / Identity Boundary 与 Task Ownership

### 目标

在扩大 Agent 自主执行能力之前，把 Verified Principal、Tenant、Project 建立为平台不变量，并阻止 Cross-Tenant/Cross-Project Task Access。

### 实现内容

- 显式 `Principal` Contract：User、ServiceAccount、System；
- Protected API 之前完成 Authentication 与 Principal Resolution；
- Scoped Task Repository 与 Task Subresource Guard；
- Task Scope Persistence 原子化；
- 长期 Task Schema 直接保存 `tenant_id`、`project_id`、`created_by`；
- PostgreSQL Runtime Role 使用 Transaction-local Tenant/Project Context + RLS；
- App/Worker DB Role 不具备 Administrative Bypass；
- Health/Readiness 不进入 Tenant Authorization，但仍遵循 Infrastructure Access Policy。

### 验收条件

- Missing Identity Fail Closed；
- Missing Tenant 绝不代表 System Access；
- Project API 必须具有 Project Context；
- Tenant A 无法访问 Tenant B Task；
- 同 Tenant 不同 Project 的隔离遵循 Scope Policy；
- Cross-scope Task ID 在需要时使用 Not Found Semantics；
- Task Creation + Scope Ownership 原子提交；
- App/Worker DB Role 无法 Bypass RLS；
- System Access 必须使用 Explicit System Principal/Capability，并在需要时走独立 Admin DB Path。

### Migration Note

当前 Trusted Header 与 `task_ownership` 属于 Compatibility Bridge，在 Merge 前必须逐步收敛到 Frozen Identity/RLS/Task Contract。

## 5. S2 —— API + Domain Contract Convergence

### 目标

让运行 API 与 Persistence Model 同时收敛到 Frozen Task/Identity/Event Contract 以及 `docs/implementation/contracts/openapi-v1.yaml`。

### 开发包

- OIDC/JWT Verifier Interface 与生产实现；
- RBAC + Policy/ABAC Authorization Seam；
- Stable `ErrorEnvelope`：Request ID、Trace ID、Error Code、Retryable；
- 所有 Mutating API 强制 `Idempotency-Key`；
- Canonical Task Schema：Tenant、Project、CreatedBy、Agent、Goal/Input、Constraints、Status、Version；
- Task Transition API / State Machine；
- Append-only TaskEvent Store；
- Transactional Outbox；
- Command Idempotency Record；
- Optimistic Concurrency / Resource Version；
- Pagination 与 Executable OpenAPI Contract Test。

### 退出条件

- Public Handler 只接受 Documented Request Shape；
- 所有 v0.1 Public Path 有 Executable Contract Test；
- Task State 不允许任意 Field Mutation；
- Task Mutation + Canonical Event 原子提交；
- Duplicate Mutation Request 不重复执行 Business Operation。

## 6. S3 —— Durable Task Workflow

### 目标

用 Restart-safe Orchestration 替换 No-op Workflow Seam，同时保证 Workflow Runtime 不成为 Business Database。

### 开发包

- Temporal Client/Worker Adapter 隐藏在 `workflow.Engine` 后；
- Deterministic Planning/Router/Model/Policy/Approval/Tool/Validation Workflow；
- 必要时使用 Outbox 驱动 Workflow Start/Signal；
- Durable Retry、Timeout、Cancellation、Resume；
- Idempotent Activity 与 Replay Test；
- Task Projection、TaskEvent、Workflow Runtime Reconciliation。

### 退出条件

API/Worker Restart 后 Task 可以继续执行且不重复 External Side Effect；不读取 Workflow History 也能从 PostgreSQL 查询 Business State。

## 7. S4 —— Governed Execution

### 目标

把 Tool Gateway 与 Sandbox 变成唯一 Controlled Side-effect Path。

### 开发包

- 先用 Deterministic/Fake Executor 证明业务路径；
- 再增加 Kubernetes Job Create/Watch/Collect/Destroy Executor；
- Namespace/ServiceAccount Isolation；
- Default-deny Network Policy；
- Task/Tool Scoped Short-lived Credential；
- OPA Policy Adapter；
- Human Approval Pause/Resume；
- Approval 与 Proposal Digest 强绑定；
- Signed Workspace Input 与 Controlled Artifact Output。

### 退出条件

Agent 不能绕过 `Tool Gateway -> Policy -> Approval when required -> Credential Broker -> Sandbox/Adapter` 直接访问 Enterprise Resource。

## 8. S5 —— Trace、Cost 与 Audit 完整化

### 目标

让每个成功或失败 Task 都可以完整重建，并能够核算真实经济成本。

### 开发包

- OpenTelemetry SDK 与 OTLP Export；
- Request -> Task -> Workflow -> Agent -> Model/Tool/Sandbox/Evaluation Span Hierarchy；
- Evidence 全部带 Tenant/Project/Task Correlation；
- Immutable AuditEvent 与 CostEvent；
- Model/Tool/Workflow/Sandbox/Storage/Network/Retry/Human Review Cost Activity；
- Pricing Version 与 Reconciliation；
- `Cost per Successful Task`。

### 退出条件

仅通过 Task ID 或 Trace ID 即可重建 Decision、Action、Failure、Retry、Approval、Total Cost，无需人工关联日志。

## 9. S6 —— Reliable Model Routing

### 目标

让 Provider Independence 在 Outage、Quota Exhaustion、Load 下真实成立，同时保持 Policy Hard Constraint 不被优化逻辑突破。

### 开发包

- Provider / Model / ModelVersion / Deployment Runtime Mapping；
- Redis-backed Shared Circuit Breaker；
- Provider Health/Quota/Capacity Collector；
- Latency、Queue、Residency、Budget、Evaluation Routing Input；
- Hard-policy Filter 先于 Soft Scoring；
- Bounded Retry 与 Fallback；
- Explainable Eligible/Rejected Candidate；
- Commercial API 与 Private vLLM/SGLang 统一内部协议。

### 退出条件

Primary Deployment Failure 或 Quota Exhaustion 时能够执行 Policy-compliant Fallback，不出现无限 Retry、Policy Bypass 或 Cross-tenant Leakage。

## 10. S7 —— Evaluation Release Gates

### 目标

把 Evaluation Evidence 变成可执行 Promotion、Routing Eligibility 与 Rollback Policy。

### 开发包

- Versioned Golden Dataset；
- Model/Prompt/Agent/Workflow Regression Matrix；
- L1 Offline、L2 Pre-production、L3 Production Evaluation；
- Quality、Safety、Reliability、Latency、Cost、Human Intervention Threshold；
- Immutable GateDecision；
- Production Trace Sampling 遵循 Data Governance；
- Release Gate 与 Rollback Decision。

### 退出条件

Required Gate 失败时，Version/Configuration 不能被 Promotion 或 Route Eligible，且每个 Decision 都能从存储 Evidence 复现。

## 11. S8 —— End-to-End Product Proof

使用三个 Governance Path：

```text
A. Read-only Repository/Cluster Inspection        -> ALLOW
B. Scale dev-gpu-cluster gpu-workers 3 -> 6       -> REQUIRE APPROVAL
C. Destructive Production Request                 -> DENY
```

Mutation Path 必须经过：

```text
API
 -> Authenticated Principal/Tenant/Project
 -> Idempotent Task Creation
 -> TaskEvent + Outbox
 -> Durable Workflow
 -> Policy-eligible Candidate Set
 -> Router
 -> Provider/Deployment
 -> Structured ChangePlan
 -> Validator
 -> Policy
 -> Human Approval when required
 -> Tool Gateway
 -> Short-lived Credential
 -> Sandbox/Kubernetes or Fake Adapter
 -> Read-back Validation
 -> COMPLETED
```

必须产生：

- Task Events；
- Route Decision 与 Model Attempts；
- Policy Decision；
- Approval Record；
- Tool Invocation；
- Audit Events；
- Cost Events；
- OpenTelemetry Trace；
- Evaluation Result。

这三个 ALLOW / APPROVE / DENY 路径共同组成 AI Cloud v0.1 Definition of Done。

## 12. Slice Review Template

每一个 Future Slice 在 Coding 前必须回答：

```text
1. Goal
2. Non-Goals
3. Domain Changes
4. API Changes
5. Data Model Changes
6. Security Boundary
7. Runtime Flow
8. Failure Model
9. Idempotency Model
10. Observability
11. Cost Model
12. Migration
13. Tests
14. Acceptance Criteria
15. Rollback Strategy
```

## 13. Pull Request 规则

每个 Slice/Remediation Unit 使用独立 Branch 与 Draft PR。Implementation Code 不直接提交到 `main`。只有 Contract Review、Acceptance Criteria 与 CI Gate 全部通过后，PR 才进入 Ready。英文文档与 `.zh-CN.md` 必须在同一个 PR 中同步维护。