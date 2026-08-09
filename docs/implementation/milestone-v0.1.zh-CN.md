# AI Cloud v0.1 可直接开发里程碑

## 目标
v0.1 只证明一件事：**一条完整、安全、Provider-independent、可审计、可恢复、可计算成本的 Task 执行链真正跑通。** 当前阶段优先保证边界正确，不追求接入最多模型或最多 Agent。

## Slice 0：工程与构建基础
实现 API/Worker Entrypoint、Typed Config、Structured Logging、Makefile、Migration、Docker Compose、Helm、CI。

验收：Main 上 `go test ./...`、`go vet ./...`、Build、Migration Validation、Helm Template 全部通过。

## Slice 1：Tenant Context 与 Persistence Foundation
实现 Tenant、Project、Subject、RequestContext Middleware、PostgreSQL Transaction/Repository Base、Tenant-scoped Query Helper、首批 RLS Policy、Idempotency Store。

验收：Two-tenant Integration Test 证明不能跨租户读写；Tenant Context 缺失时 Fail Closed。

## Slice 2：Model Domain 与 MockProvider
实现 Provider-neutral GenerateRequest/Response、Normalized ProviderError、Deterministic MockProvider、Model/ModelVersion/ProviderEndpoint Repository。

验收：Domain Request 可通过 MockProvider 执行；Domain 不出现 Provider SDK Type；Model Admission/Lifecycle Test 通过。

## Slice 3：Router v1
实现 Candidate Filter、Deterministic Score、RouteDecision Persistence、Fallback Chain、Health/Quota Input Interface。

验收：Table-driven Test 覆盖 Capability、Policy、Residency、License、Health、Budget 淘汰；Selected/Rejected Reason 全部可查询。

## Slice 4：Task API 与 Durable State
实现 POST/GET/Cancel Task、TaskEvent Append Log、Optimistic Version、Temporal Workflow Adapter、Restart-safe State Machine。

验收：API/Worker Restart 后 Task 可继续；相同 Idempotency-Key 重复 Create 返回同一 Task；Cancel 正确进入 Terminal State。

## Slice 5：Agent Plan 与 Validation
实现首个 Agent Version：请求模型生成 Structured ChangePlan，并校验 Target、Expected Current Value、Desired Value、Risk、Rollback、Allowed Operation Type。

验收：Malformed/Unsafe Plan 永远不能进入 Execution；MockProvider Golden Test Deterministic。

## Slice 6：Policy 与 Approval
实现 PolicyEngine Port、OPA Adapter、PolicyDecision Persistence、WAITING_APPROVAL、Approve API。

验收：Deny 阻止执行；RequireApproval 可持久暂停 Workflow；Expired/Mismatched Approval 不能恢复 Action。

## Slice 7：Tool Gateway 与 Credential Boundary
实现 Tool Registry、ToolGateway Pipeline、CredentialBroker Interface + Fake/Local Adapter、Kubernetes Scale Tool Adapter Interface、Audit Record。

验收：Agent Package 中不存在直接 Kubernetes Client；每次 Tool Invocation 都有关联 PolicyDecision 和 Task-scoped Credential Reference；重复 Invocation 幂等。

## Slice 8：Sandbox Foundation
实现 Kubernetes Job Sandbox Adapter，用于需要执行生成代码/脚本的场景；配置 Restricted SecurityContext、Resource/Timeout/Network Control、Artifact Collection。

验收：Negative Test 证明禁止 hostPath、Privileged、默认公网访问；CPU/Memory/Runtime 有界。

## Slice 9：Cost、Audit、Trace
实现 Append-only CostEvent/AuditEvent、OpenTelemetry Trace Hierarchy、Model/Tool/Sandbox Usage、Task Cost Reconciliation。

验收：通过一个 trace_id 可以重建完整 Task；Retry/Failure 后仍能得到不重复的 Reconciled Cost。

## Slice 10：第一个端到端场景
完整运行：

```text
POST Task
 -> Workflow
 -> Router
 -> MockProvider
 -> Structured ChangePlan
 -> Validator
 -> Policy
 -> Optional Approval
 -> Tool Gateway
 -> Fake Kubernetes Adapter
 -> Read-back Validation
 -> COMPLETED
```

Reference Goal：

```text
scale dev-gpu-cluster gpu-workers from 3 to 6
```

验收：不存在绕过 Policy/Tool Boundary 的直接执行路径；TaskEvents、RouteDecision、AuditEvents、CostEvents、Trace 全部存在并相互关联。

## Slice 11：真实 Commercial Provider + Private Model Adapter
增加一个商业 Provider Adapter，以及一个 OpenAI-compatible Private/Self-hosted Adapter（兼容 vLLM/SGLang）。

验收：两个 Adapter 通过相同 Provider Contract Suite；Router 可以切换/Fallback，Agent/Task 代码无需修改；Credential 仍只存在于 Adapter Boundary。

## Slice 12：Release Hardening
增加 Load/Smoke Test、Backup/Restore Drill、Migration Roll-forward Test、Provider Outage Simulation、Quota Exhaustion、Policy Outage、Tenant Isolation Regression、Helm Production Values 文档。

验收：满足 `deployment-testing.zh-CN.md` 的 Production-ready Definition。

## v0.1 明确不做
Marketplace、Autonomous Multi-agent Swarm、ML Router、Per-tenant Kubernetes Cluster、Model Training/Fine-tuning、大规模 MCP Ecosystem、复杂 Billing Settlement、Multi-region Active-active 均不属于 v0.1。只保留未来可扩展 Interface，不提前实现。

## Issue 拆分模板
每个 Slice 再拆 Issue 时必须包含：Objective、Affected Package、Contract Reference、Schema/API Change、Security Impact、Telemetry/Cost Impact、Tests、Migration Plan、Acceptance Criteria。

任何改变 Public Contract 的 PR 必须同步更新 Implementation Doc 和中英文版本。

## 推荐立即编码顺序
先完成 Slice 0/1，不再继续无限扩展抽象架构；然后严格按：

```text
2 -> 3 -> 4 -> 5 -> 6 -> 7 -> 9 -> 10
```

推进。Sandbox 在 Tool Gateway Interface 稳定后可并行；真实 Provider 必须等 MockProvider E2E 全绿后再接入。