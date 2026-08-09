# AI Cloud 安全边界模型

> 状态：S0 Contract Freeze

## 1. 目标

明确 Trust 在哪里建立、Authorization 在哪里执行、Side Effect 在哪里允许发生。安全能力必须是分层且独立于模型行为的，不能把“模型按 Prompt 行事”当成安全边界。

## 2. Trust Boundary

```text
External Client
  -> Authn Boundary
  -> Principal / Tenant Boundary
  -> API Authorization Boundary
  -> Domain / Policy Boundary
  -> Workflow Boundary
  -> Tool Gateway Boundary
  -> Credential Boundary
  -> Sandbox / Adapter Boundary
  -> Enterprise Resource
```

Model、Prompt 或 Agent Output 即使来自平台内部，也不自动可信。

## 3. Authorization Pipeline

```text
Authenticate
  -> Resolve Principal
  -> Resolve Resource Scope
  -> RBAC
  -> ABAC / Policy
  -> Domain Invariant
  -> Execute
```

任一层都可以 Deny。后续层不能覆盖 Platform Hard Constraint 的 Deny。

## 4. Hard Constraint 与 Contextual Policy

Platform Hard Constraint 包括：

- Tenant Isolation；
- Explicit Identity Requirement；
- Model License/Admission Restriction；
- Data Residency；
- Prohibited Tool/Resource Class；
- Sandbox Baseline；
- Credential TTL 与 Max Scope。

Tenant/Project Policy 可以更严格，例如增加 Approval、Budget Threshold、Environment Restriction，但不能削弱 Platform Hard Constraint。

## 5. Agent Boundary

Agent 不持有长期生产凭证，也不直接访问企业系统。

禁止：

```text
Agent -> kubectl
Agent -> SSH key
Agent -> database password
Agent -> cloud admin token
Agent -> production filesystem
```

要求：

```text
Agent Proposal
  -> Tool Gateway
  -> Schema Validation
  -> Policy Decision
  -> Human Approval when required
  -> Credential Broker
  -> Sandbox / Adapter
  -> Target Resource
```

## 6. Credential Boundary

Credential Broker 只在 Policy 允许后签发 Task/Tool Scoped Short-lived Credential。

每个 Grant 至少记录：

```text
grant_id
tenant_id
project_id
task_id
tool_invocation_id
resource
permissions
expires_at
policy_decision_id
```

Credential 只交给 Execution Adapter/Sandbox，不能写入 LLM Context。

## 7. Sandbox Baseline

生产默认执行 Profile：

```text
non-root
read-only root filesystem
drop Linux capabilities
seccomp RuntimeDefault
resource limits
execution timeout
ephemeral workspace
default-deny network
no hostPath
no Docker socket
no cluster-admin token
```

任何例外都必须使用更高 Risk Profile，并产生显式 Policy/Approval Evidence。

## 8. Data Boundary

数据在进入 Model/Tool 前先做 Classification。是否允许送出 Tenant-controlled Environment 或进入 Commercial Provider，由 Policy 决定：

```text
Data Classification
  + Tenant Policy
  + Provider/Deployment Residency
  + Model License/Admission
  -> Access Decision
```

敏感数据不能仅因为某模型 Quality Score 更高就被 Router 选中。

## 9. Model Boundary

Model Output 对下游 Execution 来说属于 Untrusted Input。Structured Output 必须通过 Schema Validation 与 Semantic Validation 后才能触发 Side Effect。

高风险 Operation 中，Model 只能生成 Proposal/Change Plan，而不是直接生成可执行命令。

## 10. Approval Boundary

Human Approval 必须绑定：

```text
proposal_digest
task_id
policy_decision_id
expires_at
```

如果 Approval 后 Proposal 发生实质变化，原 Approval 立即失效。

## 11. Audit Boundary

所有 Security-relevant Decision，包括 Deny，都必须记录：

```text
principal
resource scope
policy version
input/proposal digest
decision
reason
trace_id
task_id
```

## 12. Failure Behavior

对 Mutating/High-risk Operation，安全基础设施一律 Fail Closed。Identity、Policy Engine、Credential Broker 故障不能被转换成 Allow。

如果某些 Read-only API 支持 Degraded Mode，必须逐 Endpoint 显式定义。

## 13. Secrets

Secret 禁止：

- 写入 Prompt Template；
- 写入 TaskEvent Payload；
- 原样写入 Trace/Audit Log；
- 未经授权返回 Tool Result。

优先存储 Secret Reference，而不是 Secret Value。

## 14. 验收条件

- 访问 Protected Resource 前完成 Identity 与 Tenant Check；
- Agent 无法直接获得 Production Credential；
- Tool Side Effect 必须经过 Tool Gateway + Policy；
- High-risk Approval 与 Proposal Digest 强绑定；
- Security Subsystem Failure 对 Mutation Fail Closed；
- Sensitive Data Routing 同时覆盖 Allow/Deny Case；
- Sandbox Policy 防止 Privileged Execution；
- 所有 High-risk Allow/Deny 都产生 Audit Evidence。