# 2026-08-11 AI 行业信号与架构影响

## 目的

本文记录对 AI Cloud 架构具有长期价值的外部 AI 模型、Agent Runtime 与工具生态信号。本文是架构决策输入，不代表绑定任何具体厂商或模型。

AI Cloud 不应追逐每一次新模型发布，而应从反复出现的产业变化中抽取稳定趋势，并仅把具有长期意义的变化固化为平台契约。

## 核心结论

行业竞争正在从单一模型竞争转向完整执行系统竞争。

未来更稳定的优化单位是：

```text
Task
  -> 能力需求
  -> Model
  -> Deployment
  -> Inference Effort / Service Tier
  -> Harness / Tools
  -> 受治理的执行
  -> Trace / Evaluation / Cost Evidence
  -> Routing Feedback
```

因此，AI Cloud 不应寻找一个静态的“最佳模型”，而应针对每个任务，在质量、可靠性、成本、时延、容量、数据驻留、许可和风险约束下选择最佳的合规执行路径。

## 值得转化为产品契约的行业信号

### 1. 开放权重与托管 API 正在成为并存模式

同一个模型家族可能同时通过公有 API、企业专属 Endpoint、自托管和本地运行提供。因此 Model Identity 与 Deployment Identity 必须拆分。

架构影响：

- 分离 Model、ModelVersion 和 Deployment；
- Provider 与 Endpoint 配置不得进入不可变的模型身份定义；
- 一个 ModelVersion 可以对应多个 Deployment；
- 每个 Deployment 独立维护健康、成本、容量、区域和策略状态。

### 2. 高效模型与 Specialist Model 正在承担大量 Agent 工作负载

最强 Frontier Model 并不适合所有任务。高效、小型、专业化或本地模型可以承担确定性、窄领域、隐私敏感和高吞吐任务，而 Frontier Model 更适合作为复杂任务升级路径。

架构影响：

- Router 必须理解 Task Class 与 Capability；
- Specialist Capability 必须成为 Registry 一等元数据；
- 保留 deterministic / efficient / specialist / flagship 四类路由；
- Specialist Route 并不是降级路径，在有证据支持时可以成为首选路径。

### 3. Token 单价不足以代表真实经济性

模型价格越来越受到缓存、上下文长度、Service Tier、Batch、Region、Reasoning Effort 和部署形态影响。自托管模型还包含 GPU、排队、资源预留等成本。

因此平台应继续冻结：

```text
核心经济指标 = Cost per Successful Task
```

Router 应根据预测的完整任务成本，而不是单纯 Token Price 做选择。

### 4. Agent 能力属于 Model-Harness-Environment 系统

近期研究越来越明确地把 Runtime Harness 视为 Agent 性能的重要决定因素。Prompt、上下文管理、Tool、State、权限、Recovery、Verification 和 Trace 即使在同一底层模型下也可能明显改变任务成功率。

因此 Evaluation Target 应扩展为：

```text
MODEL
MODEL_DEPLOYMENT
MODEL_HARNESS
AGENT
WORKFLOW
ROUTE_POLICY
```

生产环境 Agent 能力不能仅归因于基础模型。

### 5. Tool 生态正在逐步标准化和可发现化

Official MCP Registry 说明外部 Tool、Data Source 和 Context Provider 正逐步进入统一发现机制。

架构影响：

- Protocol Adapter 必须独立于 Provider Adapter；
- MCP 应作为 Tool Gateway 后的一种协议，而不是企业治理边界；
- Registry Discovery 不能绕过 Tool Owner、Policy、Credential、Sandbox、Provenance 和 Approval。

### 6. Agent 自主性提升后，安全重点转向 Execution Governance

Agent 获得更多 Tool Use、长期状态和自主执行能力后，风险不再只来自模型输出，还来自权限、凭据、网络访问、工具副作用和失败恢复。

因此继续保持强制执行路径：

```text
Agent
  -> Tool Gateway
  -> Policy Engine
  -> 必要时 Approval
  -> Credential Broker
  -> Sandbox 或 Approved Resource
```

广泛 Agent Autonomy 必须建立在确定性控制和完整 Audit Evidence 之上。

### 7. Routing 应演进为基于证据的反馈闭环

静态规则例如 `coding -> model-x` 不能成为最终架构。真实生产 Trace 会形成任务成功率、重试、Fallback、成本和人工介入等结果证据。

应形成：

```text
Registry
  -> Router
  -> Execution
  -> Trace
  -> Evaluation
  -> CostEvent
  -> RouteOutcome
  -> Router Policy Update
```

未来即使引入学习型 Router，也必须受 Policy 约束，并保留可回放的 Routing Evidence。

## Router 目标函数

v0.1 不要求立即实现一个统一数学优化器，但 Contract 必须允许多目标路由。

概念上：

```text
Route Utility
~
(Task Success Probability
 * Quality
 * Reliability
 * Policy Compliance)
/
(Cost * Latency * Risk)
```

其中首先执行 Hard Constraints，再优化 Soft Objectives。

Hard Constraints：

- Capability Fit；
- Tenant Allow/Deny Policy；
- Data Residency；
- Model Approval 与 License Status；
- Credential 与 Tool Permission；
- Context Requirement；
- Safety Requirement；
- 无法安全排队时的 Capacity Requirement。

Soft Objectives：

- Predicted Task Success；
- Expected Total Cost；
- Latency；
- Queue Time；
- Historical Reliability；
- Retry Probability；
- Human Intervention Probability。

## R5-R10 架构扩展

R5-R10 是现有 Roadmap 的下一阶段架构契约，不替换也不重新编号 v0.1 Milestone 1-9。

### R5 Deployment Registry

新增独立于 Model Registry Identity 的 Deployment Source of Truth。

最小契约：

- Deployment ID 与不可变 ModelVersion 引用；
- Provider 与 Endpoint Class；
- Deployment Mode：public API、enterprise API、private endpoint、self-hosted、local、edge；
- Region 与 Data Residency；
- Runtime 与 Quantization Metadata；
- Pricing / Cost Model Reference；
- Health、Latency、Quota、Concurrency、Queue、Signal Freshness；
- Lifecycle 与 Routing Eligibility；
- Owner 与 Policy Reference。

验收方向：同一个不可变 ModelVersion 可以拥有多个可路由 Deployment，而不复制 Model Identity 和 Evaluation Provenance。

### R6 Capability / Economics / Runtime-Aware Router

Router 从“选模型”升级为联合选择：

```text
Model
+ Deployment
+ Inference Effort
+ Service Tier
```

RouteDecision 必须记录：

- Task Classification 与 Requirements；
- Candidate Set；
- Hard Filter Reject Reason；
- Predicted Quality / Success Evidence；
- Predicted Total Task Cost；
- Runtime Health / Capacity Evidence；
- Selected Deployment / Effort / Tier；
- Fallback Chain；
- Policy Version 与 Evidence Version。

### R7 Execution Evaluation

Evaluation 从 Model-centric 扩展为完整 Execution Configuration。

Evaluation Identity 至少绑定：

- Model 与 Deployment；
- System / Prompt Package；
- Harness Configuration；
- Tool；
- Permission 与 Policy；
- Workflow；
- Sandbox / Runtime；
- Budget 与 Retry Policy；
- Dataset 与 Validator。

核心生产指标：

- task_success_rate；
- cost_per_successful_task；
- p50/p95_task_latency；
- retry_count / fallback_count；
- human_intervention_rate；
- tool_failure_rate；
- policy_rejection_rate；
- unsafe_action_or_side_effect_violation_rate。

### R8 Route Outcome Feedback Loop

新增 `RouteOutcome`，关联 Task、RouteDecision、Trace 与 Evaluation Evidence。

应区分：

- Selected Route Success / Failure；
- Fallback Success / Failure；
- Retry Count；
- Final Cost；
- Final Latency；
- Quality / Validator Result；
- Human Intervention；
- Policy / Safety Outcome。

RouteOutcome 可以用于改进 Routing Policy，但在 Replay、Rollback 和 Governance 契约完善前，不允许自动修改生产策略。

### R9 Agent Harness Registry

将 Harness 从隐藏在应用代码中的实现细节提升为 Versioned Runtime Configuration。

最小契约：

- Harness ID / Version；
- Prompt / Context Strategy；
- Memory / State Strategy；
- Tool Set；
- Control Flow；
- Retry / Recovery Strategy；
- Verification Strategy；
- Permission Profile；
- Trace Configuration；
- Compatible Task / Model Capability Constraints。

这使 Model-Harness Evaluation 可以复现，也避免把 Harness 带来的收益错误归因给 Model。

### R10 Tool Execution Governance

即使未来支持 MCP 或其他 Tool Protocol，Tool Gateway 仍必须是强制执行入口。

生产最小控制：

- Tool Registry Owner 与 Risk Metadata；
- Protocol Adapter Isolation；
- Invocation 前 Deterministic Policy Decision；
- Short-lived Task-scoped Credential；
- 必要时 Sandbox / Network Boundary；
- 高风险 Side Effect Approval；
- Input / Output Digest 与 Audit Trail；
- Timeout、Idempotency、Compensation / Rollback Metadata（如 Tool 支持）。

## 更新后的 Control Plane

```text
                   AI Cloud Control Plane

 Model Registry              Deployment Registry
      |                              |
      +--------------+---------------+
                     |
              Intelligent Router
                     |
       +-------------+-------------+
       |             |             |
  Capability      Economics      Runtime
  Quality         Task Cost      Health
  Context         Budget         Capacity
  Specialist      Cache/Tier     Quota/Latency
       +-------------+-------------+
                     |
                Agent Runtime
                     |
          Harness / Memory / Tools
                     |
                Tool Gateway
                     |
      Policy -> Credential -> Sandbox
                     |
                  Execution

---------------- Evidence Plane ----------------
Trace / Evaluation / CostEvent / RouteOutcome / Audit
License / Provenance / Security Evidence
```

## 实施顺序建议

下一阶段不以“继续增加 Provider 数量”为首要目标，而保持当前 v0.1 Milestone 不变，并按以下顺序冻结新 Contract：

```text
R5 Deployment Registry
  -> R6 Joint Router
  -> R7 Execution Evaluation
  -> R8 RouteOutcome Feedback
  -> R9 Harness Registry
  -> R10 Tool Execution Governance Hardening
```

其中 R10 可以与 R5-R9 并行，因为安全 Tool Execution 始终是扩大 Agent 自主性的前置条件。

## 外部参考

以下来源仅作为 Evidence Input，不构成产品依赖：

- Official MCP Registry: https://registry.modelcontextprotocol.io/
- AI Harness Engineering: A Runtime Substrate for Foundation-Model Software Agents (2026): https://arxiv.org/abs/2605.13357
- HarnessX: A Composable, Adaptive, and Evolvable Agent Harness Foundry (2026): https://arxiv.org/abs/2606.14249
- Harness-Bench: Measuring Harness Effects across Models in Realistic Agent Workflows (2026): https://arxiv.org/abs/2605.27922
- EvoHarness-RL: Learning Self-Evolving Runtime Harness for Long-Horizon LLM Agents (2026): https://arxiv.org/abs/2608.05446

## 后续行业动态的架构变更准入规则

未来新的模型或生态新闻，仅在至少满足以下一项时才修改 AI Cloud Stable Architecture：

1. 出现新的长期 Deployment Class；
2. Routing Economics 发生实质变化；
3. 出现企业任务需要的新 Capability Class；
4. Trust、License、Residency 或 Execution Boundary 发生变化；
5. 多次证据表明当前 Registry / Evaluation / Routing Contract 无法表达新的产业现实。

否则只记录进 Research Note 或 Provider / Model Metadata，不修改 Control Plane 架构。
