# AI Cloud Cost Accounting Contract

> Status: S0 Contract Freeze

## 1. Purpose

Define immutable, task-attributable AI cost evidence so routing, budgeting, chargeback and ROI analysis use the same accounting model.

## 2. Core Principle

AI cost is task cost, not token price alone.

```text
Task Cost
 = Model Input/Output/Reasoning
 + Tool Execution
 + Workflow/Retry
 + Sandbox/Compute
 + Storage/Network
 + Human Review
 + Other metered platform activities
```

## 3. CostEvent

```yaml
cost_event:
  cost_event_id: string
  tenant_id: string
  project_id: string
  task_id: string
  trace_id: string
  logical_operation_id: string?
  attempt_id: string?
  component: string
  provider_id: string?
  model_version_id: string?
  deployment_id: string?
  tool_id: string?
  quantity: decimal
  unit: string
  unit_price: decimal
  amount: decimal
  currency: string
  pricing_version: string
  source: estimated | metered | reconciled
  occurred_at: timestamp
  created_at: timestamp
```

CostEvent is append-only.

## 4. Pricing Version

Historical cost must be reproducible. A PricingProfile is versioned and immutable after use.

CostEvent stores the pricing version used at execution time. Later provider price changes must not rewrite historical events.

## 5. Estimate vs Actual

Routing may use estimated cost. Billing/FinOps uses metered or reconciled actuals.

```text
EstimatedCost -> routing/admission
MeteredCost   -> runtime evidence
ReconciledCost -> final authoritative Task accounting
```

Differences are retained, not overwritten silently.

## 6. Retry/Fallback

Every physical provider/tool attempt can generate cost even if the attempt fails. Cost is attributed to the same Task and logical operation with a distinct attempt identity.

Successful fallback does not erase primary-attempt cost.

## 7. Budget Enforcement

Budget policy has multiple scopes:

```text
Tenant
Project
Application/Agent
Task
```

Hard limits are Policy constraints. Router uses cost only as optimization after hard eligibility checks.

## 8. Cost per Successful Task

Primary efficiency metric:

```text
Cost per Successful Task
 = total reconciled cost / successful task count
```

Model token unit price is a supporting metric, not the platform KPI.

## 9. Chargeback

Every Task must be attributable to tenant/project and optionally cost center/application tags. Shared infrastructure cost may be allocated by explicit allocation rules with versioned methodology.

## 10. Currency

CostEvent stores original currency. Cross-currency reporting uses a separately versioned FX reference and must not mutate original events.

## 11. Reconciliation

A reconciler compares AI Cloud meter events with provider invoices/GPU meters where available and emits reconciliation adjustments as new events rather than editing history.

## 12. Acceptance Criteria

- Every production Task has zero or more immutable CostEvents and a reproducible total.
- Failed/retried attempts retain their cost.
- Historical Task cost does not change when provider pricing changes.
- Estimated and actual/reconciled costs are distinguishable.
- Hard budget rules cannot be bypassed by Router scoring.
- Cost can be aggregated by tenant, project, agent, model version and deployment.
- Reconciliation adjustments are append-only and auditable.