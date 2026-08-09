# AI Cloud 工程实现蓝图

## 目的
`docs/implementation/` 是 AI Cloud 的**可执行工程规范（Canonical Implementation Specification）**。ADR 负责解释“为什么这样设计”，本目录负责定义“开发人员具体实现什么、模块如何交互、哪些契约不可随意变化、如何验收”。

当 Roadmap、架构示意图或研究文档与本目录冲突时，除非存在新的 ADR 明确覆盖，否则开发以本目录为准。

## v0.1 总体工程原则
AI Cloud v0.1 采用 **Go 1.22 Modular Monolith（模块化单体）**，提供可独立运行的 API Process 与 Worker Process。当前阶段不把 Control Plane 各模块过早拆成微服务；只有在吞吐、故障隔离、团队边界或独立扩缩容出现真实需求后再拆分。

PostgreSQL、Redis、Temporal、OPA、Object Storage、模型 Provider、Kubernetes、OpenTelemetry 等全部通过 Port/Adapter 隔离。

```text
Clients / SDK
    |
API Server
    |
Application Services
    |
+---------------- Domain Modules ----------------+
| tenant identity model routing task agent tool  |
| policy workflow sandbox cost audit evaluation  |
+-------------------------------------------------+
    |
Ports / Interfaces
    |
Adapters
    +-- PostgreSQL
    +-- Redis
    +-- Temporal
    +-- OPA
    +-- Object Storage
    +-- Provider APIs / vLLM / SGLang
    +-- Kubernetes
    +-- OpenTelemetry
```

## 目标仓库结构

```text
cmd/
  aicloud-api/          # HTTP API 进程
  aicloud-worker/       # Workflow/异步任务 Worker
api/
  http/                 # Handler/Router
  middleware/           # Auth/Tenant/RequestID
  dto/                  # API DTO，不直接复用 Provider SDK Type
  errors/               # 稳定错误模型
model/
  domain/               # Model/ModelVersion/Capability
  provider/             # Provider Port + Adapter
  registry/             # Model Registry
  router/               # Candidate Filter + Scoring + Fallback
agent/
  domain/
  runtime/
workflow/
  domain/
  temporal/
tool/
  domain/
  gateway/
policy/
  engine/
identity/
  domain/
  authn/
  authz/
tenant/
  domain/
sandbox/
  runtime/
cost/
  ledger/
eval/
  runner/
audit/
  ledger/
observability/
  telemetry/
storage/
  postgres/
  redis/
  object/
infra/
  kubernetes/
integrations/
migrations/
deploy/
  helm/
  compose/
docs/
```

目录名后续可以调整，但依赖方向必须保持：

```text
HTTP / Worker Adapter
        ↓
Application / Domain
        ↓
Ports
        ↑
Infrastructure Adapter
```

Infrastructure Adapter 只能实现 Port，不能拥有业务策略。

## 十项强制 Domain Invariant

1. 所有 Tenant Resource 必须携带 TenantID；Project Resource 同时携带 ProjectID。
2. Task 必须拥有 TaskID、TraceID、TenantID、ProjectID、SubjectID，并保持创建身份上下文不可变。
3. Provider SDK 类型不得越过 Provider Adapter 边界进入 Domain。
4. `Models propose. Policy decides. Humans approve when required. Controllers execute.`
5. Tool Call 和 Sandbox Execution 在产生副作用前必须完成 Policy Evaluation。
6. Model/Tool/Sandbox 每次执行都必须产生 Usage、Cost、Audit 和 Trace 数据。
7. Router 必须先过滤 Policy 不允许的候选，再进行质量/成本/延迟评分。
8. Retry 必须有界且幂等；Fallback 不得降低 Tenant、License、Residency 或 Security 约束。
9. Production ModelVersion 使用不可变引用；升级必须产生新版本和新的 Admission Decision。
10. Tenant 或 Authorization Context 缺失时 Fail Closed。

## 稳定 ID
API 对 ID 一律视为 Opaque String，内部优先使用 UUIDv7 兼容 ID。最低集合：

```text
tenant_id, project_id, subject_id, model_id, model_version_id,
agent_id, workflow_id, task_id, tool_id, sandbox_id,
policy_id, approval_id, trace_id, cost_event_id, audit_event_id
```

## Task 状态机

```text
CREATED
  -> PLANNING
  -> EXECUTING
  -> WAITING_APPROVAL
  -> EXECUTING
  -> VALIDATING
  -> COMPLETED

任意非终态 -> FAILED / CANCELLED
```

状态变化必须写入 Append-only `TaskEvent`；`Task` 主表只是当前状态的 Materialized Projection，用于高效查询。

## 开发规范文档

- `component-contracts.zh-CN.md`：模块职责、依赖和 Go Interface。
- `api-data-contracts.zh-CN.md`：HTTP API、数据库 Schema、幂等、错误和事件 Envelope。
- `runtime-security-flow.zh-CN.md`：完整执行链、Router、Policy、Tool、Credential、Sandbox、Approval。
- `deployment-testing.zh-CN.md`：本地/Kubernetes 部署、HA、Observability、测试门禁。
- `milestone-v0.1.zh-CN.md`：可以直接领取开发的迭代顺序和验收标准。

## Definition of Done
一个功能只有同时完成 Domain Behavior、Persistence、Authorization、适用的 Telemetry/Audit/Cost、Unit Test、Integration Test、API Contract Test 和 Failure-path Test 后才算完成。仅有设计文档或仅存在未合并 Branch 均不计为 Main 上完成。

## 变更纪律
以下内容发生不兼容变化时必须新增/修订 ADR：Resource Identity、Tenant Boundary、Provider Abstraction、Task State Machine、Policy Boundary、Audit/Cost Immutability、Public API。实现文档及中英文版本必须在同一变更周期同步更新。