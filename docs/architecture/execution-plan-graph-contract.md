# AI Cloud Execution Plan Graph Contract

> Status: Architecture baseline for post-S0 evolution

## 1. Purpose

AI Cloud evolves the runtime abstraction from `prompt -> model` into:

```text
task -> execution plan -> model / deployment / tool / subagent graph
```

A Task remains the business execution aggregate. An ExecutionPlan is the versioned runtime intent that decomposes the Task into an executable directed graph. The graph may contain model inference, concrete deployment selection, tool invocation, subagent delegation, approval/wait, and control nodes.

## 2. Core Invariants

1. Task owns business intent and final business outcome; ExecutionPlan does not replace Task.
2. ExecutionPlan is immutable once execution starts. Replanning creates a new plan version linked to the prior version.
3. Every executable node is policy checked before execution.
4. Model identity and Deployment identity remain separate. A model node may express capability intent; route-time binding selects an eligible Deployment.
5. Tool side effects still cross only the Tool Gateway and must produce Evidence.
6. Subagents do not receive ambient authority. Delegated identity, scope, budget, tools and expiry are explicit.
7. Every node attempt has stable idempotency identity, bounded retries and trace context.
8. Cost is accounted at node/attempt level and rolls up to the Task.
9. Route-time pricing, policy, evaluation and signal versions are snapshotted into decision evidence.
10. Eval measures graph-level task success and cost-per-success, not only model benchmark scores.

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

The first implementation should support a DAG. Cyclic autonomous loops require a later explicit contract because they need separate termination, budget and safety semantics.

## 4. Planning and Binding

Planning and routing are distinct phases:

```text
Task
  -> Planner: capability decomposition and graph construction
  -> Policy: node/action eligibility
  -> Router: eligible model/deployment binding and fallback
  -> Executor: graph execution
  -> Evidence/Eval/Cost: attempt and task outcome recording
```

The Planner may request capabilities such as `coding`, `vision`, `long-context`, or `low-latency`; it must not hard-code provider-specific deployment identifiers unless the Task explicitly requires one.

## 5. RouteDecision Scope

RouteDecision becomes node-scoped. One Task may therefore contain multiple RouteDecisions.

Each model/subagent node records at least:

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

`pricing_snapshot` is the route-time economic evidence used for later cost reconstruction; it must not be inferred from the current catalog price after execution.

## 6. Subagent Delegation

A subagent node receives a bounded delegation envelope:

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

A subagent may create a child plan only within this envelope. It cannot expand its own privileges, tool set, data scope or budget.

## 7. Evidence and Evaluation

Evidence must reconstruct:

```text
Task intent
-> plan version
-> node graph
-> policy decisions
-> route bindings and pricing snapshots
-> tool/subagent actions
-> retries/fallbacks
-> node outputs
-> final outcome
-> total cost and latency
```

Primary production evaluation should include `task_success`, `cost_per_success`, `end_to_end_latency`, `retry_rate`, `fallback_rate`, `tool_failure_rate`, and `autonomous_duration` where applicable.

## 8. Acceptance Criteria

- A Task can reference one active ExecutionPlan version.
- A plan contains stable node and edge identities.
- Router operates per routable node, not only once per Task.
- Every route binding stores ModelVersion + Deployment + route-time pricing snapshot.
- Tool and subagent nodes are policy-scoped and auditable.
- Replanning produces a new immutable version and preserves causal linkage.
- Node costs and attempts roll up deterministically to Task cost.
- Eval can compute cost per successful Task from persisted execution evidence.
- The initial executor rejects unbounded cyclic graphs.
