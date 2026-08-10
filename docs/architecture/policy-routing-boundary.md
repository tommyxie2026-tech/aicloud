# AI Cloud Policy and Routing Boundary

> Status: S0 Contract Freeze

## 1. Purpose

Separate authorization/eligibility decisions from optimization decisions. Policy determines what is allowed; Router determines the best choice among what remains allowed.

## 2. Fundamental Rule

```text
Policy Engine
  answers: MAY this candidate/action be used?

Router
  answers: WHICH eligible candidate should be selected?
```

A routing score can never override a policy denial.

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

Everything before `Eligible Deployments` is a hard-constraint stage. Router scoring only operates on eligible candidates.

## 4. Policy Inputs

Policy may evaluate:

```text
Principal / roles / capabilities
Tenant and Project
Task goal/type
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
historical task success rate
latency
estimated cost
provider reliability
queue/capacity pressure
service tier
cache locality
preference weights
```

Router must not independently reinterpret license, residency, tenant isolation or other hard policy constraints.

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

Rejected candidates include stable reason codes without leaking secrets.

## 8. Fallback

Fallback candidates are pre-authorized at decision time when possible. Runtime fallback still rechecks time-sensitive constraints such as health, quota, capacity and policy expiry.

A fallback candidate that no longer satisfies a hard constraint is skipped, regardless of score.

## 9. Tool Policy

The same separation applies to tools:

```text
Policy -> allow / deny / require approval
Execution strategy -> where/how to run an allowed tool
```

The Agent cannot choose a more privileged execution path to bypass policy.

## 10. Budget Semantics

Hard budget constraints belong to Policy/Admission. Cost optimization belongs to Router.

Example:

```text
monthly budget exceeded -> deny
per-task max $1 -> candidate must fit hard estimate
among allowed candidates -> Router minimizes expected cost subject to quality/SLA
```

## 11. Explainability

Every route must be explainable from persisted evidence:

```text
which candidates were considered
which were rejected and why
which hard constraints applied
which soft scores were used
why the selected deployment won
what fallback chain was prepared
```

## 12. Failure Semantics

If Policy is unavailable for a mutation/high-risk request, fail closed. Router unavailability may use a deterministic policy-approved default only when explicitly configured; otherwise fail safely.

## 13. Acceptance Criteria

- Router receives only policy-eligible candidates.
- A high score never resurrects a denied candidate.
- RouteDecision records selected ModelVersion + Deployment.
- Denied/rejected reason codes are persisted.
- Budget hard limits and optimization are implemented in separate layers.
- Fallback rechecks time-sensitive eligibility.
- Policy and routing versions/signals are traceable per Task.
- Deterministic tests prove hard constraints always dominate soft scores.