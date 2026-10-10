# AI Cloud Policy and Routing Boundary

> Status: S0 Contract Freeze + ExecutionPlan evolution

## 1. Purpose

Separate authorization/eligibility decisions from optimization decisions. Policy determines what is allowed; Router determines the best choice among what remains allowed.

The runtime target is no longer only `prompt -> model`. Routing must support:

```text
Task -> ExecutionPlan -> model / deployment / tool / subagent graph
```

## 2. Fundamental Rule

```text
Planner
  answers: WHAT capabilities and steps are required?

Policy Engine
  answers: MAY this node/candidate/action be used?

Router
  answers: WHICH eligible binding should be selected?

Executor
  answers: HOW is the approved graph executed and recovered?
```

A routing score can never override a policy denial, and the Planner cannot grant authority by constructing a graph node.

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

Everything before `Eligible Deployments / Execution Targets` is a hard-constraint stage. Router scoring only operates on eligible candidates.

One Task may contain multiple routable nodes and therefore multiple RouteDecisions.

## 4. Policy Inputs

Policy may evaluate:

```text
Principal / roles / capabilities
Tenant and Project
Task goal/type
ExecutionPlan ID/version
Node ID/kind/risk class
Delegation envelope
Data classification
ModelVersion admission state
Model license and usage restrictions
Deployment region/residency
Tool/action risk
Environment (dev/stage/prod)
Budget hard limits
Approval requirements
Time/risk windows
```

Policy inputs that may change over time must be versioned or captured as decision evidence.

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

High-risk decisions must be persisted. Denials are evidence too.

## 6. Router Inputs

Router may optimize on soft signals such as:

```text
quality score
historical task/node success rate
latency
estimated cost
provider reliability
queue/capacity pressure
service tier
cache locality
preference weights
```

Router must not independently reinterpret license, residency, tenant isolation, delegation scope or other hard policy constraints.

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

Rejected candidates include stable reason codes without leaking secrets.

`pricing_snapshot` captures the route-time economic inputs actually used for selection. Historical cost reconstruction must not depend on the current catalog price.

## 8. Fallback

Fallback candidates are pre-authorized at decision time when possible. Runtime fallback still rechecks time-sensitive constraints such as health, quota, capacity, delegation expiry and policy expiry.

A fallback candidate that no longer satisfies a hard constraint is skipped, regardless of score.

## 9. Tool and Subagent Policy

The same separation applies to tools and subagents:

```text
Policy -> allow / deny / require approval
Router/Execution Strategy -> where/how to execute an allowed node
```

A subagent receives an explicit bounded delegation envelope. It cannot choose a more privileged execution path, tool set, data scope or budget than its parent delegation permits.

## 10. Budget Semantics

Hard budget constraints belong to Policy/Admission. Cost optimization belongs to Router.

Example:

```text
monthly budget exceeded -> deny
node max cost exceeded -> candidate must be rejected
within allowed candidates -> Router minimizes expected task cost subject to quality/SLA
```

Budget applies at both Task envelope and Node envelope. Node attempt costs roll up to the Task.

## 11. Explainability

Every route must be explainable from persisted evidence:

```text
which plan/node required routing
which candidates were considered
which were rejected and why
which hard constraints applied
which soft scores were used
why the selected binding won
what route-time price context applied
what fallback chain was prepared
```

## 12. Failure Semantics

If Policy is unavailable for a mutation/high-risk request, fail closed. Router unavailability may use a deterministic policy-approved default only when explicitly configured; otherwise fail safely.

If replanning is required after a node failure, a new immutable ExecutionPlan version must be created; the current plan is not silently mutated.

## 13. Acceptance Criteria

- Router receives only policy-eligible candidates.
- A high score never resurrects a denied candidate.
- RouteDecision is scoped to ExecutionPlan + Node.
- A Task can contain multiple RouteDecisions.
- RouteDecision records ModelVersion + Deployment when model execution is selected.
- Route-time pricing context is persisted with decision evidence.
- Denied/rejected reason codes are persisted.
- Budget hard limits and optimization are implemented in separate layers.
- Fallback rechecks time-sensitive eligibility.
- Tool and subagent nodes respect explicit policy/delegation boundaries.
- Policy and routing versions/signals are traceable per Task and Node.
- Deterministic tests prove hard constraints always dominate soft scores.