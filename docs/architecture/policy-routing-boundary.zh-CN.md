# AI Cloud Policy 与 Routing 边界

> 状态：S0 Contract Freeze

## 1. 目标

严格分离 Authorization/Eligibility Decision 与 Optimization Decision。Policy 决定“能不能用”，Router 只在允许的候选中决定“用哪个更好”。

## 2. 基本原则

```text
Policy Engine
  回答：这个 Candidate/Action 是否允许？

Router
  回答：在允许的 Candidate 中哪个最优？
```

Routing Score 永远不能覆盖 Policy Deny。

## 3. Model Routing Pipeline

```text
Task Requirements
  -> Registry Candidate Discovery
  -> Admission Filter
  -> Tenant/Project Policy Filter
  -> License Filter
  -> Data Residency Filter
  -> Security/Risk Filter
  -> Health/Quota/Capacity Filter
  -> Budget Hard Limit
  -> Eligible Deployments
  -> Router Scoring
  -> Selected Deployment + Fallback Chain
```

在 `Eligible Deployments` 之前全部属于 Hard Constraint；Router Scoring 只对 Eligible Candidate 生效。

## 4. Policy Input

Policy 可以评估：

```text
Principal / roles / capabilities
Tenant / Project
Task goal/type
Data classification
ModelVersion admission state
Model license / usage restriction
Deployment region/residency
Tool/action risk
Environment(dev/stage/prod)
Budget hard limit
Approval requirement
Time/risk window
```

会随时间变化的 Policy Input 必须 Versioned 或固化进 Decision Evidence。

## 5. PolicyDecision Contract

```yaml
policy_decision:
  policy_decision_id: string
  tenant_id: string
  project_id: string
  task_id: string?
  subject_id: string
  action: string
  resource: string
  decision: allow | deny | require_approval
  reasons: []string
  policy_version: string
  input_digest: string
  created_at: timestamp
```

High-risk Decision 必须持久化，Deny 同样属于 Evidence。

## 6. Router Input

Router 只优化 Soft Signal，例如：

```text
quality score
historical task success rate
latency
estimated cost
provider reliability
queue/capacity pressure
service tier
cache locality
preference weight
```

Router 不得自行重新解释 License、Residency、Tenant Isolation 等 Hard Policy。

## 7. RouteDecision Contract

```yaml
route_decision:
  route_decision_id: string
  tenant_id: string
  project_id: string
  task_id: string
  selected:
    model_version_id: string
    deployment_id: string
  eligible_candidates: []object
  rejected_candidates: []object
  fallback_chain: []object
  scores: object
  constraints: object
  policy_version: string
  evaluation_version: string?
  pricing_version: string?
  signal_version: string?
  created_at: timestamp
```

Rejected Candidate 需要保存稳定 Reason Code，但不能泄露 Secret。

## 8. Fallback

Fallback Candidate 应尽可能在 Route Decision 阶段预先授权。Runtime Fallback 仍需重新检查 Health、Quota、Capacity、Policy Expiry 等动态 Hard Constraint。

如果 Candidate 已不满足 Hard Constraint，即使 Score 很高也必须跳过。

## 9. Tool Policy

Tool 同样采用：

```text
Policy -> allow / deny / require approval
Execution Strategy -> 在允许后决定如何执行
```

Agent 不能通过选择更高权限的 Execution Path 绕过 Policy。

## 10. Budget Semantics

Hard Budget 属于 Policy/Admission；Cost Optimization 属于 Router。

例如：

```text
monthly budget exceeded -> deny
per-task max $1 -> candidate 必须满足硬预算
在允许候选中 -> Router 再优化成本/质量/SLA
```

## 11. Explainability

每次 Route 必须可解释：

```text
考虑了哪些 Candidate
哪些被拒绝以及原因
用了哪些 Hard Constraint
用了哪些 Soft Score
为什么 Selected Deployment 胜出
准备了什么 Fallback Chain
```

## 12. Failure Semantics

对于 Mutation/High-risk Request，Policy Unavailable 必须 Fail Closed。Router Unavailable 只有在显式配置了 Policy-approved Deterministic Default 时才能降级，否则安全失败。

## 13. 验收条件

- Router 只接收 Policy-eligible Candidate；
- High Score 永远不能恢复被 Deny 的 Candidate；
- RouteDecision 同时保存 ModelVersion + Deployment；
- Denied/Rejected Reason Code 可追溯；
- Hard Budget 与 Cost Optimization 分层；
- Fallback 重新检查动态 Eligibility；
- 每个 Task 可追溯 Policy/Route Version 与 Signal Version；
- Deterministic Test 证明 Hard Constraint 始终优先于 Soft Score。