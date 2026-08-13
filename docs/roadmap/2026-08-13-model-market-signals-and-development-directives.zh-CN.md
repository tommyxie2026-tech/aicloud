# 2026-08-13 模型市场信号与后续开发约束

## 目的

本文将 2026-08-13 商业模型与开放权重模型市场信号转换为 AI Cloud 的工程约束。本文延续 R5-R10 架构参考，不引入任何厂商绑定。

## 核心结论

当前有三类市场变化已经值得直接进入工程契约：

1. 开放权重与商业使用权正在成为两个不同维度；
2. 模型价格与可用性越来越依赖 Deployment、Region、Cache、Batch、Service Tier、Context 和 Inference Effort 等动态条件；
3. 托管模型具有明确的 Preview、Deprecated、Replacement、Quota Reduction、Retired 等生命周期事件。

因此稳定的执行链仍然是：

```text
Task
  -> Policy / Requirements
  -> Capability
  -> ModelVersion
  -> Deployment
  -> Pricing / Capacity / Region / Service Tier
  -> Execution
  -> Evaluation
  -> CostEvent
  -> RouteOutcome
  -> Policy-bounded Feedback
```

## 1. License Economics 必须成为路由约束

开放权重不代表商业使用无限制，也不代表商业成本为零。模型的商业条款可能受到收入门槛、托管服务、再分发、衍生作品、署名等条件限制。

### 开发要求

扩展 License Evidence，使生产准入能够被机器判定。

最小字段应支持：

- 权重是否公开以及权重 License；
- 商业使用：允许 / 有条件允许 / 禁止；
- 托管服务：允许 / 有条件允许 / 禁止；
- 再分发：允许 / 有条件允许 / 禁止；
- 衍生作品限制；
- 署名与 Notice 要求；
- 收入或使用量阈值；
- Revenue Sharing 或额外费用义务的证据引用；
- 地域或客户类型限制；
- 生效时间、过期/复核时间与权威证据引用；
- Reviewer、Approval State 与 Evidence Digest。

Router 不应直接理解厂商特有 License 文本，而应消费归一化的、可由 Policy Engine 判定的商业约束字段，同时保留不可变证据引用。

### 验收方向

即使某个模型技术上可用且 Capability 匹配，只要当前 Tenant/Task 的使用方式违反商业条款，该候选路由必须在执行前被拒绝。

## 2. Pricing 必须版本化并归属于 Deployment

单个 `price_per_token` 字段已经不足以描述现代推理经济性。

价格可能受以下条件影响：

- Deployment / Provider；
- Region；
- Input / Output；
- Cache Hit / Miss；
- Context 区间；
- Batch / Async 模式；
- Service Tier；
- Inference / Reasoning Effort；
- 时间窗口或促销时段；
- Reserved / Dedicated Capacity；
- Self-hosted GPU 分摊与利用率。

### 开发要求

增加版本化的 `PricingPolicy` 或等价 Deployment-scoped Cost Model，由 Deployment 引用。

RouteDecision 使用的 Pricing Snapshot 必须可回放；最终 CostEvent 必须记录实际使用量，并绑定用于结算的具体 PricingPolicyVersion。

### 验收方向

当同一 ModelVersion 存在两个不同 Deployment，且 Region、Cache 或 Service Tier 成本不同，Router 能解释为什么选择其中一个，并可使用当时保存的 Pricing Evidence 回放决策。

## 3. Model 与 Deployment Lifecycle 必须成为一等对象

托管模型 ID 并不是永久基础设施。Provider 可能将模型从 Preview 推向 Stable，也可能 Deprecated、降低 Quota、指定 Replacement 并最终 Retire。

### 开发要求

ModelVersion Lifecycle 与 Deployment Lifecycle 必须分离。

建议 ModelVersion 状态：

```text
DRAFT -> ACTIVE -> DEPRECATED -> RETIRED
                  \-> REVOKED
```

建议 Deployment 状态：

```text
DISCOVERED -> READY -> DEGRADED -> DRAINING -> RETIRED
                     \-> BLOCKED
```

Lifecycle Event 至少记录：

- announced-at；
- effective-at；
- Replacement ModelVersion / Deployment 引用；
- Provider Notice / Evidence；
- Quota / Rate Limit 变化；
- Routing Eligibility；
- Migration State。

### 验收方向

Deprecated Deployment 可以在策略允许下继续承载已有流量，但不能无提示继续作为默认首选；Retired、Revoked、Blocked 对象不得进入新路由。历史 Task 始终保持对实际 ModelVersion 和 Deployment 的精确引用。

## 4. 增加可控 Model Migration Workflow

生命周期事件意味着 Control Plane 需要迁移工作流，而不仅是一个状态字段。

建议迁移路径：

```text
Provider Notice / Internal Decision
  -> Lifecycle Event
  -> Replacement Candidate Discovery
  -> Shadow / Regression Evaluation
  -> Policy + License Check
  -> Canary Traffic
  -> Traffic Shift
  -> Observation Window
  -> Rollback or Complete
  -> Old Deployment Drain / Retire
```

### 开发要求

禁止仅依靠模型 Alias 自动替换底层 ModelVersion。迁移必须可验证、可回滚、有证据。

最小 Migration Record：

- Source / Target ModelVersion 与 Deployment；
- 迁移原因；
- Evaluation Evidence；
- Policy / License Evidence；
- Canary Percentage 或 Cohort；
- 开始/结束时间；
- Success Gates；
- Rollback Target；
- 最终 Decision 与 Actor。

## 5. Public Benchmark 只能作为参考证据

企业模型选择还受到 Trust、Supply Chain、Residency、Commercial Rights、Runtime Reliability 与 Deployment Freedom 等因素影响。

### 开发要求

Public Benchmark 继续作为 Evidence Source，但生产路由应优先使用 Task-Class-specific Execution Evaluation 和 RouteOutcome Evidence。

任何公开综合分数都不得绕过 Policy、License、Residency、Lifecycle、Capacity 等硬约束。

## 6. Open Weight + Managed Inference 必须天然可表达

同一个开放权重 ModelVersion 可以同时被 Vendor API、Managed Inference Provider、Enterprise Private Endpoint 或 Self-hosted Runtime 提供服务。

### 开发要求

Provider 与 Deployment 继续作为可替换运行资源。Open-weight 状态属于 ModelVersion Evidence；Managed Service 的 Price、Health、Capacity 属于 Deployment。

业务 Workflow 不得通过硬编码 Provider 名称判断模型是“开源”还是“商业”。

## 7. Router 的硬约束与软目标

### Hard Constraints

优化前必须先判定：

- Capability / Context Fit；
- Tenant Model / Provider Allow-Deny Policy；
- Lifecycle / Routing Eligibility；
- License / Commercial-use Eligibility；
- Residency / Region Requirements；
- Security / Provenance Approval；
- Safe Capacity；
- 需要 Tool 时的 Credential / Tool Policy。

### Soft Objectives

只在合格 Candidate 集合中优化：

- Predicted Task Success；
- Predicted Cost per Successful Task；
- Expected Latency / Queue Time；
- Historical Reliability；
- Retry / Fallback Probability；
- Human Intervention Probability；
- Deployment Preference 与 Operational Cost。

## 8. Schema 演进方向

下一阶段实现应逐步收敛到以下关系：

```text
Model
  1 -> N ModelVersion

ModelVersion
  1 -> N Deployment
  1 -> N LicenseEvidence
  1 -> N EvaluationEvidence

Deployment
  1 -> N PricingPolicyVersion
  1 -> N RuntimeSignalWindow
  1 -> N LifecycleEvent

Task
  1 -> N RouteDecision
  1 -> N CostEvent
  1 -> N RouteOutcome

ModelMigration
  source ModelVersion/Deployment
  target ModelVersion/Deployment
  evaluation + policy + rollout evidence
```

## 9. 对开发优先级的影响

这些信号不需要新建架构层，而是进一步细化 R5、R6、R7 与 Model Supply-Chain Governance。

### Immediate

1. 完成 ModelVersion 与 Deployment 分离；
2. 增加 Deployment Lifecycle 与 Routing Eligibility；
3. 定义版本化、Deployment-scoped PricingPolicy；
4. 扩展 License Evidence，使商业约束机器可判定；
5. RouteDecision 必须保存 Pricing、License、Lifecycle 与 Evidence Version。

### Next

1. 增加 Model / Deployment Migration Workflow，以及 Canary / Rollback Evidence；
2. 将动态价格维度纳入 Predicted Task Cost；
3. 增加 Lifecycle-triggered Evaluation 与 Replacement Recommendation；
4. 使用 RouteOutcome 比较 Replacement Candidate 的真实生产表现。

### Deferred

在 Replay、Rollback、Policy Governance 与 Evidence Quality 尚未验证前，不实现自动价格套利或完全自动模型迁移。

## 10. Engineering Gates

以下任一情况存在时，Routing 实现不能视为完成：

- ModelVersion 中仍保存可变 Endpoint Health / Capacity；
- 使用单个静态 Token Price 作为权威成本；
- License 仅由自由文本标签表示；
- Deprecated / Retired Deployment 可以在没有显式策略的情况下继续参与路由；
- Model Alias 可以静默切换底层 ModelVersion；
- RouteDecision 无法基于当时使用的 Pricing / License / Lifecycle Evidence 精确回放。

生产模型升级只有满足以下条件后才可视为完成：

- Replacement Evaluation Evidence；
- License / Policy Admission；
- Controlled Traffic Shift；
- Rollback Capability；
- Shift 后的 RouteOutcome Observation。

## 与现有契约的关系

本文细化但不替代：

- R5 Deployment Registry；
- R6 Capability / Economics / Runtime-Aware Router；
- R7 Execution Evaluation；
- R8 Route Outcome Feedback Loop；
- Evidence-based Model Supply-Chain Governance。

核心原则保持不变：

> AI Cloud 优化的是受治理的完整执行路径，而不是追逐单个模型发布。
