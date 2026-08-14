# AI Cloud Execution Plan Graph 契约

> 状态：S0 之后演进的架构基线

## 1. 目的

AI Cloud 的运行时抽象从 `prompt -> model` 演进为：

```text
task -> execution plan -> model / deployment / tool / subagent graph
```

Task 继续作为业务执行聚合根。ExecutionPlan 是版本化的运行时意图，用于把 Task 分解为可执行的有向图。图中的节点可以是模型推理、具体 Deployment 绑定、Tool 调用、Subagent 委派、审批/等待以及控制节点。

## 2. 核心不变量

1. Task 拥有业务意图和最终业务结果；ExecutionPlan 不替代 Task。
2. ExecutionPlan 一旦开始执行即不可变。Replan 必须生成新版本，并与上一版本建立因果关联。
3. 每个可执行节点在执行前都必须通过 Policy 检查。
4. Model Identity 与 Deployment Identity 必须保持分离。Model Node 可以表达能力需求，真正执行时再由 Router 绑定到 Eligible Deployment。
5. Tool 的 Side Effect 仍只能通过 Tool Gateway，并必须生成 Evidence。
6. Subagent 不获得 Ambient Authority。委派的 Identity、Scope、Budget、Tools 和 Expiry 必须显式定义。
7. 每次 Node Attempt 都必须有稳定 Idempotency Identity、Bounded Retry 和 Trace Context。
8. 成本按 Node/Attempt 归集，最终 Roll-up 到 Task。
9. Route-time 的 Pricing、Policy、Evaluation 和 Signal Version 必须固化到 Decision Evidence。
10. Eval 必须衡量 Graph-level Task Success 与 Cost per Success，而不是只看 Model Benchmark。

## 3. Graph Model

```yaml
execution_plan:
  execution_plan_id: string
  task_id: string
  version: integer
  planner_version: string
  policy_version: string
  status: proposed | approved | executing | superseded | completed | failed
  nodes: []ExecutionNode
  edges: []ExecutionEdge
  budget_envelope: object
  created_at: timestamp

execution_node:
  node_id: string
  kind: model | tool | subagent | approval | wait | transform | branch | join
  requirements: object
  binding: object?
  input_refs: []string
  output_contract: object
  retry_policy: object
  budget_limit: object?
  risk_class: string

execution_edge:
  from_node_id: string
  to_node_id: string
  condition: object?
```

第一阶段实现只支持 DAG。带循环的 Autonomous Loop 需要后续单独冻结契约，因为它涉及独立的终止条件、预算上限和安全语义。

## 4. Planning 与 Binding 分层

Planning 与 Routing 必须分离：

```text
Task
  -> Planner：能力拆解与 Graph 构建
  -> Policy：Node / Action Eligibility
  -> Router：Eligible Model / Deployment Binding + Fallback
  -> Executor：Graph Execution
  -> Evidence / Eval / Cost：Attempt 与 Task Outcome 记录
```

Planner 可以表达 `coding`、`vision`、`long-context`、`low-latency` 等能力需求；除非 Task 明确要求，否则不得在 Domain 层硬编码 Provider-specific Deployment ID。

## 5. RouteDecision 作用域

RouteDecision 从 Task-level 单次选择升级为 Node-scoped Decision。一个 Task 因此可以包含多个 RouteDecision。

每个 Model/Subagent Node 至少记录：

```yaml
route_binding:
  execution_plan_id: string
  node_id: string
  route_decision_id: string
  model_version_id: string
  deployment_id: string
  fallback_chain: []object
  policy_version: string
  evaluation_version: string?
  pricing_snapshot: object
  signal_version: string?
```

`pricing_snapshot` 是路由当时用于经济决策和事后成本重建的证据，执行结束后不能再用“当前 Catalog Price”反推历史成本。

## 6. Subagent Delegation

Subagent Node 必须收到受限的 Delegation Envelope：

```yaml
delegation:
  parent_task_id: string
  parent_node_id: string
  delegated_principal: string
  allowed_capabilities: []string
  allowed_tools: []string
  data_scope: object
  max_cost: object
  max_duration: duration
  expires_at: timestamp
```

Subagent 可以在该 Envelope 内创建 Child Plan，但不能自行扩大权限、Tool Set、Data Scope 或 Budget。

## 7. Evidence 与 Evaluation

Evidence 必须能够重建：

```text
Task Intent
-> Plan Version
-> Node Graph
-> Policy Decisions
-> Route Bindings + Pricing Snapshots
-> Tool / Subagent Actions
-> Retry / Fallback
-> Node Outputs
-> Final Outcome
-> Total Cost + Latency
```

生产 Eval 的核心指标应至少包括 `task_success`、`cost_per_success`、`end_to_end_latency`、`retry_rate`、`fallback_rate`、`tool_failure_rate`，以及适用场景下的 `autonomous_duration`。

## 8. 验收条件

- 一个 Task 可以引用一个 Active ExecutionPlan Version；
- Plan 内 Node / Edge Identity 稳定；
- Router 面向每个 Routable Node 工作，而不是只在 Task 开始时路由一次；
- 每次 Route Binding 保存 ModelVersion + Deployment + Route-time Pricing Snapshot；
- Tool 与 Subagent Node 均具备 Policy Scope 与 Audit Evidence；
- Replan 生成新的 Immutable Version，并保留因果链；
- Node Cost / Attempt 可以确定性 Roll-up 为 Task Cost；
- Eval 可以基于持久化 Execution Evidence 计算 Cost per Successful Task；
- 第一阶段 Executor 必须拒绝无边界循环 Graph。
