# AI Cloud 部署与测试规范

## 1. 环境
最低环境集合：Local、CI、Dev、Staging、Production。Local/CI 可以减少依赖，但 Staging 必须覆盖与 Production 相同的 Security Boundary、Persistence Boundary 和 Tenant Boundary。

## 2. 本地开发
Docker Compose 提供 PostgreSQL、Redis、Temporal Development Server、OPA，以及可选 S3-compatible Object Storage。API/Worker 可以直接运行在 Host 提高迭代速度。

`MockProvider` 是强制组件，必须 Deterministic，使 Unit/Integration/E2E Test 不依赖外部商业模型 API，也不产生不可预测 Token 成本。

## 3. Kubernetes Production Topology

```text
Ingress / API Gateway
        |
        +-> aicloud-api Deployment >= 2
        |
        +-> aicloud-worker Deployment >= 2

Stateful / External:
        PostgreSQL HA
        Redis HA / Managed
        Temporal Cluster / Managed
        OPA Adapter
        Object Storage
        OpenTelemetry Collector

Execution:
        Sandbox Jobs -> Restricted Namespace
        Self-hosted Models -> Dedicated GPU Namespace/Cluster
```

API 必须 Stateless；Worker 可 Horizontal Scale；Durable State 存储在 PostgreSQL/Temporal/Object Storage。Redis 永远不是 Source of Truth。

## 4. Namespace Model
初始 Production Namespace：

```text
aicloud-system      API / Worker / Control Components
aicloud-sandbox     Default Sandbox Jobs
aicloud-models      Self-hosted Model Runtime
aicloud-observe     Optional Telemetry Collectors
```

默认不采用“一个 Tenant 一个 Kubernetes Namespace”。Tenant 主要通过 Domain Context、DB Isolation、Workload Label/ServiceAccount/Policy 实现。确有高隔离要求的 Tenant 后续可进入 Dedicated Namespace/Cluster Profile。

## 5. Kubernetes 安全基线
API/Worker：Non-root、尽可能 Read-only RootFS、Drop Capability、Resource Request/Limit、PodDisruptionBudget、Readiness/Liveness/Startup Probe、受限 ServiceAccount，禁止 Wildcard Cluster-admin。

Sandbox 使用 `runtime-security-flow.zh-CN.md` 中更严格的 Profile。NetworkPolicy Default Deny，Egress 必须显式 Allowlist。

## 6. Configuration / Secret
配置采用 Environment + Typed Config，并在 Startup 完成 Validation。Secret 通过 Secret Manager/Kubernetes Secret Adapter 引用，禁止进入 Git。Model Registry 只保存 Credential Reference，禁止在 ModelVersion 中保存 API Key 明文。

## 7. 初始 Production 可用性目标
API 初始目标 99.9% Monthly Availability（不把上游 Provider 全局故障算作自身可用性）。单个 API/Worker Pod 故障不能丢失 Task State。Provider 故障必须进入 Policy-compliant Fallback 或明确 Unavailable，不得无限重试。

## 8. Observability 最低要求
统一 OpenTelemetry Trace、Prometheus-compatible Metrics、Structured JSON Log。

必须具备的 Metric 范围：HTTP Rate/Error/Latency、Task State/Duration、Provider Latency/Error/RateLimit、Router Candidate Rejection Reason、Tool/Policy/Approval Count、Sandbox Duration/Failure、Token/Usage/Cost、Workflow Retry、Queue/Backlog。

## 9. CI Gate
每个 PR 至少执行：

```text
gofmt check
go test ./...
go vet ./...
build API + worker
migration validation
OpenAPI validation
unit tests
PostgreSQL repository integration tests
policy tests
multi-tenant isolation tests
MockProvider provider-contract tests
Helm lint/template
```

进入 Production Release 前增加 Image Scan、Dependency/Supply-chain Scan 和 SBOM Check。

## 10. Test Pyramid
Unit Test：Domain Invariant、State Machine、Deterministic Routing、Policy Input Builder。

Integration Test：PostgreSQL、必要的 Redis、OPA、Temporal Adapter、Tool/Sandbox Adapter（使用安全 Fake）。

Provider Contract Test：每个 Provider Adapter 必须通过完全相同的 Behavior Suite。

E2E：完整执行：

```text
API
-> Workflow
-> Router
-> MockProvider
-> Policy
-> Fake Tool/Sandbox
-> Validation
-> Cost/Audit/Trace
```

## 11. 强制 Negative Test

- 伪造 tenant_id 不能访问其他 Tenant；
- Tenant Resource 查询缺少 Scope 时被 Repository/RLS 阻断；
- Fallback 不能选择 Policy-ineligible Model；
- Policy Engine 故障时 Side Effect Fail Closed；
- Expired Approval 不能执行；
- Expired Credential 不能执行；
- 同一 Idempotency Key 的 Tool Invocation 不产生重复副作用；
- Unknown Tool Risk Class 默认 Deny；
- Sandbox 不能访问未授权 Network/Host Resource；
- Provider Timeout 服从 Task Total Deadline；
- Workflow Retry 不重复 CostEvent；
- Trace/Audit 不包含配置为 Secret 的字段。

## 12. Release Gate
Model/Provider Version 只有在 Provider Contract Test、Routing Compatibility、Safety/Policy Check、Evaluation Threshold 全部通过后才能 Promote。

Application Release 只有在 Migration、Roll-forward/Rollback Strategy、Smoke Test、Tenant Isolation Suite、Reference Scenario 全部通过后才能 Promote。

## 13. Backup / Recovery
PostgreSQL 与 Temporal 必须有 Backup 和 Restore Drill；Artifact 按 Retention Policy 保存；Redis 丢失最多影响性能，不能破坏 Source-of-truth State；Workflow Replay 必须保持原 Tenant Context 和 Idempotency。

## 14. Production-ready 定义
至少满足：2 个 API Replica、Durable Workflow/Persistence、Tenant Isolation Test、Fail-closed Policy Boundary、Bounded Retry/Fallback、Reference Scenario 的完整 Audit/Cost/Trace、Health/Readiness Probe、Backup/Restore Procedure、已验证 Upgrade Path。