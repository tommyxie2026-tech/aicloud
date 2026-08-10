# AI Cloud 成本核算契约

> 状态：S0 Contract Freeze

## 1. 目标

定义不可变、可归属于 Task 的 AI Cost Evidence，使 Routing、Budget、Chargeback、ROI Analysis 使用同一个 Accounting Model。

## 2. 核心原则

AI 成本是 Task Cost，不只是 Token 单价。

```text
Task Cost
 = Model Input/Output/Reasoning
 + Tool Execution
 + Workflow/Retry
 + Sandbox/Compute
 + Storage/Network
 + Human Review
 + 其他 Metered Platform Activity
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

CostEvent 必须 Append-only。

## 4. Pricing Version

历史成本必须可以复现。PricingProfile 一旦用于生产计费后应视为不可变 Version。

CostEvent 保存执行时使用的 Pricing Version。未来 Provider 调价不能改写历史 Event。

## 5. Estimate 与 Actual

Routing 可以使用 Estimated Cost；Billing/FinOps 使用 Metered 或 Reconciled Actual Cost。

```text
EstimatedCost -> Routing/Admission
MeteredCost   -> Runtime Evidence
ReconciledCost -> 最终 Task Accounting
```

差异必须保留，不能静默覆盖。

## 6. Retry/Fallback

每一次物理 Provider/Tool Attempt 即使失败，也可能产生真实成本，因此必须保留对应 CostEvent。

Fallback 成功不能抹掉 Primary Attempt 的成本。

## 7. Budget Enforcement

Budget Policy 可以作用于：

```text
Tenant
Project
Application/Agent
Task
```

Hard Limit 属于 Policy Constraint；Router 只能在通过 Hard Eligibility 后再用 Cost 做 Optimization。

## 8. Cost per Successful Task

推荐主效率指标：

```text
Cost per Successful Task
 = total reconciled cost / successful task count
```

Model Token Unit Price 只是辅助指标，不是 AI Cloud 的最终 KPI。

## 9. Chargeback

每个 Task 必须能归属于 Tenant/Project，并可选带 Cost Center/Application Tag。Shared Infrastructure Cost 通过显式、版本化 Allocation Rule 分摊。

## 10. Currency

CostEvent 保存 Original Currency。跨币种报表使用独立 Versioned FX Reference，不能修改原始 CostEvent。

## 11. Reconciliation

Reconciler 将 AI Cloud Meter Event 与 Provider Invoice/GPU Meter 做比对。差异通过新的 Adjustment Event 记录，而不是修改历史 Event。

## 12. 验收条件

- 每个 Production Task 都有可复现的 Cost Total；
- Failed/Retry Attempt 成本保留；
- Provider Pricing Change 不改变历史 Task Cost；
- Estimated 与 Actual/Reconciled Cost 可区分；
- Hard Budget 不能被 Router Score 绕过；
- Cost 可按 Tenant、Project、Agent、ModelVersion、Deployment 聚合；
- Reconciliation Adjustment 采用 Append-only 并可审计。