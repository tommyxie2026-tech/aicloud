# AI Cloud Policy 与 Routing 边界

> 状态：S0 Contract Freeze + ExecutionPlan 演进

## 1. 目标

严格分离 Authorization/Eligibility Decision 与 Optimization Decision。Policy 决定“能不能用”，Router 只在允许的候选中决定“用哪个更好”。

运行时目标不再只是 `prompt -> model`，Routing 必须支持：

```text
Task -> ExecutionPlan -> model / deployment / tool / subagent graph
```

## 2. 基本原则

```text
Planner
  回答：需要哪些能力与执行步骤？

Policy Engine
  回答：这个 Node / Candidate / Action 是否允许？

Router
  回答：在允许候选中应绑定哪个执行目标？

Executor
  回答：已授权 Graph 如何执行、恢复与重试？
```

Routing Score 永远不能覆盖 Policy Deny；Planner 也不能通过“创建一个 Graph Node”来凭空授予权限。

## 3. Routable Node Pipeline

```text
Task Requirements
  -> ExecutionPlan / Node Requirements
  -> Registry Candidate Discovery
  -> Admission Filter
  -> Tenant/Project/Delegation Policy Filter
  -> License Filter
  -> Data Residency Filter
  -> Security/Risk Filter
  -> Health/Quota/Capacity Filter
  -> Budget Hard Limit
  -> Eligible Deployments / Execution Targets
  -> Router Scoring
  -> Node Binding + Fallback Chain
```

在 `Eligible Deployments / Execution Targets` 之前全部属于 Hard Constraint；Router Scoring 只对 Eligible Candidate 生效。

一个 Task 可以包含多个 Routable Node，因此也可以产生多个 RouteDecision。

## 4. Policy Input

Policy 可以评估：

```text
Principal / roles / capabilities
Tenant / Project
Task goal/type
ExecutionPlan ID/version
Node ID/kind/risk class
Delegation envelope
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
  execution_plan_id: string?
  node_id: string?
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
historical task/node success rate
latency
estimated cost
provider reliability
queue/capacity pressure
service tier
cache locality
preference weight
```

Router 不得自行重新解释 License、Residency、Tenant Isolation、Delegation Scope 等 Hard Policy。

## 7. RouteDecision Contract

```yaml
route_decision:
  route_decision_id: string
  tenant_id: string
  project_id: string
  task_id: string
  execution_plan_id: string
  node_id: string
  selected:
    model_version_id: string?
    deployment_id: string?
    execution_target_id: string?
  eligible_candidates: []object
  rejected_candidates: []object
  fallback_chain: []object
  scores: object
  constraints: object
  policy_version: string
  evaluation_version: string?
  pricing_snapshot: object?
  signal_version: string?
  created_at: timestamp
```

Rejected Candidate 需要保存稳定 Reason Code，但不能泄露 Secret。

`pricing_snapshot` 保存路由当时真正用于选择的经济输入。历史成本重建不能依赖当前 Catalog Price。

## 8. Fallback

Fallback Candidate 应尽可能在 Route Decision 阶段预先授权。Runtime Fallback 仍需重新检查 Health、Quota、Capacity、Delegation Expiry、Policy Expiry 等动态 Hard Constraint。

如果 Candidate 已不满足 Hard Constraint，即使 Score 很高也必须跳过。

## 9. Tool 与 Subagent Policy

Tool/Subagent 同样采用：

```text
Policy -> allow / deny / require approval
Router / Execution Strategy -> 在允许后决定在哪里、如何执行
```

Subagent 必须收到显式 Bounded Delegation Envelope，不能选择超出 Parent Delegation 的更高权限 Execution Path、Tool Set、Data Scope 或 Budget。

## 10. Budget Semantics

Hard Budget 属于 Policy/Admission；Cost Optimization 属于 Router。

例如：

```text
monthly budget exceeded -> deny
node max cost exceeded -> candidate 必须被拒绝
在允许候选中 -> Router 在质量/SLA 约束下优化预期 Task Cost
```

Budget 同时存在于 Task Envelope 与 Node Envelope。Node Attempt Cost 最终 Roll-up 到 Task。

## 11. Explainability

每次 Route 必须可解释：

```text
哪个 Plan / Node 触发了 Routing
考虑了哪些 Candidate
哪些被拒绝以及原因
用了哪些 Hard Constraint
用了哪些 Soft Score
为什么 Selected Binding 胜出
路由当时采用了什么 Pricing Context
准备了什么 Fallback Chain
```

## 12. Failure Semantics

对于 Mutation/High-risk Request，Policy Unavailable 必须 Fail Closed。Router Unavailable 只有在显式配置了 Policy-approved Deterministic Default 时才能降级，否则安全失败。

如果 Node Failure 需要 Replan，必须生成新的 Immutable ExecutionPlan Version，不能静默修改当前 Plan。

## 13. 验收条件

- Router 只接收 Policy-eligible Candidate；
- High Score 永远不能恢复被 Deny 的 Candidate；
- RouteDecision 作用域为 ExecutionPlan + Node；
- 一个 Task 可以包含多个 RouteDecision；
- 选择模型执行时，RouteDecision 同时保存 ModelVersion + Deployment；
- Route-time Pricing Context 必须持久化为 Decision Evidence；
- Denied/Rejected Reason Code 可追溯；
- Hard Budget 与 Cost Optimization 分层；
- Fallback 重新检查动态 Eligibility；
- Tool/Subagent Node 遵守显式 Policy / Delegation Boundary；
- 每个 Task / Node 可追溯 Policy/Route Version 与 Signal Version；
- Deterministic Test 证明 Hard Constraint 始终优先于 Soft Score。