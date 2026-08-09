# ADR-0019：AI Cloud 参考实现架构

## 状态
已接受

## 背景
AI Cloud 已经从 Model Gateway 演进为企业 AI Operating System。但只有“Control Plane / Data Plane / Execution Plane”分层还不足以直接开发，仓库必须进一步确定 Process Topology、Module Boundary、Dependency Direction、Persistence Strategy、Runtime Sequence 和 Acceptance Model。

## 决策
v0.1 参考实现明确采用 **Go 1.22 Modular Monolith（模块化单体）**，提供两个主要 Process：`aicloud-api` 与 `aicloud-worker`。Domain Module 之间通过明确 Interface 协作；外部基础设施通过 Port/Adapter 访问。未来需要拆微服务时保持 Domain Contract 不变。

```text
                    AI Cloud Platform

Clients / SDK / Enterprise Integrations
                |
          API / Auth Boundary
                |
+---------------+----------------------------------+
|                  Control Plane                   |
| Model Registry | Router | Policy | Eval | FinOps |
+---------------+----------------------------------+
                |
+---------------+----------------------------------+
| Identity / Tenant / Governance / Audit Boundary |
+---------------+----------------------------------+
                |
+---------------+----------------------------------+
| Workflow / Agent Runtime / Tool Gateway          |
+---------------+----------------------------------+
                |
+---------------+----------------------------------+
| Data / Knowledge / Artifact / Task State         |
+---------------+----------------------------------+
                |
+---------------+----------------------------------+
| Execution: Sandbox / Controllers / Model Runtime |
+---------------+----------------------------------+
                |
 PostgreSQL / Redis / Temporal / OPA / Object Store
 Kubernetes / vLLM / SGLang / Commercial Providers
```

## Process Topology

### aicloud-api
负责 Public HTTP Contract、Authentication/Tenant Context、同步 Validation、Resource CRUD、Task Submission、Query、Approval API。禁止在 API Process 中直接执行长时间 Agent Workflow。

### aicloud-worker
负责 Durable Workflow Activity、Agent Planning、Router Invocation、Policy/Tool/Sandbox Orchestration、Validation、Evaluation Scheduling 和 Task State Progression。

两个 Process 共享同一组 Domain Package 和 Port。PostgreSQL/Temporal 保存 Durable State；Redis 只用于 Cache/Coordination，不得成为 Source of Truth。

## 强制 Module Boundary
核心模块：tenant、identity、model/provider、model/registry、router、task、agent、workflow、policy、tool、credential、sandbox、cost、audit、evaluation、observability、storage adapter、infrastructure adapter。

Provider SDK 只能存在 Provider Adapter；Temporal SDK Type 只能存在 Workflow Adapter；OPA/Kubernetes/PostgreSQL/Redis SDK 细节禁止进入 Domain Type。

## Production Invariant

1. Provider Agnostic：业务 Workflow 不得 Import Provider SDK。
2. Tenant Context 必须贯穿 API、Persistence、Routing、Workflow、Tool、Sandbox、Cost、Trace。
3. Models propose；Policy decides；Humans approve when required；Controllers execute。
4. Side Effect 只能在 Policy Check 后通过 Tool Gateway/Sandbox 产生。
5. Router 先执行 Hard Constraint Filter，再做 Deterministic Scoring。
6. Task State 必须 Durable + Replay-safe；Side Effect 必须 Idempotent 或可 Reconcile。
7. AuditEvent 和 CostEvent Append-only。
8. Production ModelVersion 必须完成 Admission，并使用不可变 Version Reference。
9. Fallback 不得降低 Policy、License、Residency、Security 和 Hard Budget。
10. 任意 Reference Task 必须可以通过 TaskEvents + Trace + Audit + Cost 完整重建。

## Canonical Implementation Specification
真正可以直接编码的规范统一进入 `docs/implementation/`：

```text
docs/implementation/
├── README.zh-CN.md
├── component-contracts.zh-CN.md
├── api-data-contracts.zh-CN.md
├── runtime-security-flow.zh-CN.md
├── deployment-testing.zh-CN.md
├── milestone-v0.1.zh-CN.md
└── contracts/
    ├── openapi-v1.yaml
    └── postgres-v0.1.sql
```

对应职责：工程蓝图与仓库结构、Go Interface 与依赖边界、HTTP/PostgreSQL/Event/Migration Contract、执行与安全链、Kubernetes/Test/Release Gate、可直接领取的开发 Slice，以及 Machine-readable API/DB Contract。

## 影响
本决策明确避免当前阶段过早微服务化，同时保留未来 Service Extraction Boundary。至此 ADR 不再只是架构思想，而是与 Implementation Spec、Machine-readable Contract、CI Acceptance Criteria 形成闭环，可直接驱动代码实现。