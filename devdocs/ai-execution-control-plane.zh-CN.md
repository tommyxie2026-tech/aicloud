# AI Execution Control Plane：从模型平台到任务执行控制面的演进

[English](ai-execution-control-plane.md) | **简体中文**

> 状态：研究输入（devdocs），不是正式产品承诺  
> 观察日期：2026-09-16

## 1. 核心判断

AI Cloud 不应停留在“模型管理平台”或“Agent Platform”。下一阶段建议将核心抽象提升为 **AI Task Execution Control Plane**：系统治理的一级对象不是 Model，也不是 Agent，而是 **Task**。

目标是把四个目前相互割裂的世界连接起来：

1. AI：Model / Agent / Memory / MCP / Evaluation；
2. 企业：Identity / Domain Data / API / Workflow / Governance；
3. 计算：GPU / NPU / PPU / K8s / VM / Bare Metal；
4. 持续学习：Task → Execution → Outcome → Evaluation → Organization Memory。

核心原则：**Model proposes; Policy permits; Runtime executes; Evaluation learns.** Prompt 不是权限，模型也不拥有最终执行权。

## 2. 目标架构

```text
                         TASK
                           │
                     Task Planner
                           │
                     Policy Engine
                           │
                  Intelligent Router
             ┌─────────────┼─────────────┐
             ↓             ↓             ↓
          Model          Agent         Human
             │             │
             └──────┬──────┘
                    ↓
               Tool Registry
                    ↓
             Execution Runtime
        ┌───────────┼───────────┐
        ↓           ↓           ↓
       K8s          VM       Bare Metal
        │           │           │
        └───────────┼───────────┘
                    ↓
             GPU / NPU / PPU
                    │
                    ↓
             Execution Trace
                    │
                    ↓
                Evaluation
                    │
                    ↓
           Organization Memory
                    │
                    └──────────→ Next Task
```

控制面负责声明“允许什么、由谁完成、使用什么资源、预算多少、何时升级给人”；执行面只负责在约束内可靠执行。

## 3. 一级资源模型

建议逐步形成以下平台 CRD/API 级对象：

- `TaskDefinition`：目标、输入、SLA、预算、风险等级、成功条件；
- `ModelProfile`：Provider、能力、成本、延迟、KV Cache、硬件兼容性、许可证、数据边界；
- `AgentProfile`：Agent 能力、工具集合、运行时、委托边界；
- `ToolDefinition`：MCP/API/Function、凭据、读写能力、风险等级；
- `Policy`：主体、任务、模型、工具、数据、预算和审批约束；
- `ExecutionPlan`：Planner/Router 生成的可审计执行 DAG；
- `ExecutionRun`：实际执行实例、状态、资源和 checkpoint；
- `MemoryObject`：Session/User/Procedural/Organization Memory；
- `EvaluationProfile`：任务成功、质量、安全、成本、人工介入等评价规则；
- `ResourceProfile`：GPU/NPU/PPU、K8s/VM/BM、地域、容量和调度约束。

## 4. Model Registry 2.0

现有 Model Registry 应从静态模型目录升级为“可执行能力注册表”。除参数量、上下文和 Benchmark 外，增加：

```text
identity / provider / license
capability / modality / tool-use / reasoning
quality / task-success-rate
input-price / output-price / cache-price
TTFT / prefill-throughput / decode-throughput
KV-cache-per-token / HBM-requirement
quantization / tensor-parallel / speculative-decoding
GPU-NPU-PPU compatibility
vLLM / SGLang / TensorRT-LLM compatibility
data-retention / residency / private-deployment
task-cost / reliability / observed-failure-rate
```

长期核心指标从 `$ / token` 转向 **Cost per Successful Task**。

## 5. Intelligent Router 2.0

Router 不应只做 Provider failover 或价格路由，而应面向 Task 做多目标决策：

```text
Route = f(
  capability,
  modality,
  quality,
  latency,
  task_state,
  privacy,
  jurisdiction,
  license,
  hardware,
  cost,
  reliability,
  risk
)
```

支持四类路由：模型路由、Agent 路由、工具路由、计算资源路由。Planner 生成候选执行计划，Router 选择执行路径，Policy Engine 拥有最终许可权。

## 6. Policy Engine 2.0

Policy 从内容过滤升级为 Execution Authorization。至少控制：

- 哪个主体可以发起什么 Task；
- Task 可以访问哪些 Domain Data；
- 哪些模型可以处理该数据等级；
- Agent 可以调用哪些 Tool；
- Tool 是否允许 read/write/delete/execute；
- 是否需要 Human Approval；
- 最大 token / GPU-time / monetary budget；
- 最大委托深度和 sub-agent 数；
- 网络、文件系统、secret 和 credential 边界；
- 地域、法域、数据驻留和日志保留要求。

关键原则：凭据永远由 Runtime 注入，模型只看到 capability，不直接拥有 secret。

## 7. Execution Runtime

Execution Runtime 是与 LangGraph/Dify 等上层框架拉开差异的核心。

需要具备：Durable Execution、Checkpoint/Resume、Sandbox、Credential Broker、Network Policy、Filesystem Policy、Human-in-the-loop、Timeout、Budget Enforcement、Kill/Suspend、Replay、Immutable Audit。

Runtime 与 Agent Framework 解耦，允许 LangGraph、CrewAI、自研 Agent、Codex-like harness 等作为 workload 运行。

## 8. 异构计算执行面

AI Cloud 已有 K8s / KubeVirt / 物理机与异构 GPU 调度方向，应统一进入 Resource Plane：

```text
Task Requirement
      ↓
Execution Plan
      ↓
Resource Scheduler
 ┌────┼─────┐
 K8s  VM    Bare Metal
  ↓    ↓       ↓
 GPU / NPU / PPU
```

调度输入不仅包含显存和卡数，还应包含模型 Runtime、量化方式、TP/PP、KV Cache、数据地域、启动时间、成本和 SLA。

## 9. Memory 与持续学习

Memory 分层：

- Session Memory：当前执行上下文；
- User/Team Memory：稳定偏好和业务上下文；
- Procedural Memory：任务如何成功完成；
- Organization Memory：跨任务沉淀的组织级可复用经验。

重点保存的不只是对话，而是经过脱敏和评价后的 Execution Trajectory：

```text
Problem
 → Plan
 → Model/Agent decisions
 → Tool calls
 → Execution
 → Feedback
 → Outcome
 → Evaluation
 → Reusable Procedure
```

只有通过 Evaluation、Policy 和治理门槛的轨迹才能进入 Organization Memory，避免错误经验自我强化。

## 10. Evaluation 2.0

从模型 Benchmark 扩展为 Task Evaluation。核心指标：

- Task Success Rate；
- Cost per Successful Task；
- Human Intervention Rate；
- Effective Autonomous Horizon；
- Recovery Rate；
- Tool Success Rate；
- Policy Violation Rate；
- P50/P95 Completion Time；
- Resource Efficiency；
- Memory Reuse Gain。

Evaluation 结果反向进入 Router、Model Registry 和 Organization Memory，形成闭环，但生产策略变更必须可审计、可回滚，不能由模型自行直接修改权限策略。

## 11. Domain AI

Domain AI 不应被定义为简单 Fine-tune 或 RAG：

```text
Domain AI =
General Intelligence
+ Domain Data
+ Domain Knowledge
+ Domain Tools
+ Domain Evaluation
+ Workflow
```

Domain Pack 可作为未来产品扩展单位，包含数据连接器、工具、策略、评价集、Prompt/Procedure 和 Agent 模板，而底层模型可替换。

## 12. 外部产品研究映射

截至观察日期，值得持续跟踪的方向包括 Microsoft Foundry Agent Service、AWS Bedrock AgentCore、Google Vertex AI Agent Engine、OpenAI Agents/agent runtime、LangGraph、Dify，以及新兴的 security/runtime-first 开源项目。

研究重点不是复制功能，而是验证以下假设：

- Runtime 是否可以真正 framework-neutral；
- Model Registry 是否可以 provider-neutral；
- Resource Plane 是否可以 hardware-neutral；
- Policy 是否能独立于 prompt/model；
- Memory 是否能形成受治理的持续学习闭环；
- Task 是否适合作为统一一级抽象。

外部产品事实、版本和能力应单独建立 research notes，并按 devdocs 证据等级规则持续验证。

## 13. 建议迭代路线

### R1：Task Contract

先定义 `TaskDefinition / ExecutionPlan / ExecutionRun` 契约，不急于开发复杂 UI。让现有推理调用能够被统一包装成 Task，并完整记录成本、模型、资源和结果。

### R2：Registry + Router

升级 Model Registry；实现 Provider-neutral Router；接入本地 vLLM/SGLang 与至少两个商业 Provider；建立 Task-level cost/quality telemetry。

### R3：Tool + Policy + Runtime

建立 Tool Registry、Credential Broker、Policy Engine；加入 sandbox、checkpoint、approval、kill、budget 和 audit。此阶段开始具备真正 Agent Execution Control Plane 特征。

### R4：Heterogeneous Resource Plane

统一 K8s / VM / Bare Metal 和 GPU/NPU/PPU 调度；把模型运行画像与资源画像连接起来，实现 Task → Model → Runtime → Hardware 的联合调度。

### R5：Evaluation + Organization Memory

建立 Task Evaluation，沉淀经过验证的 Procedural Memory；让 Router 根据历史成功率、成本和可靠性优化，但策略发布必须经过治理流程。

### R6：Domain Packs

选择 1–2 个真实企业场景验证 Domain Data + Tool + Evaluation + Workflow 的组合，证明平台价值不是“部署更多模型”，而是“以更低成本和更高可靠性完成业务任务”。

## 14. 产品边界

暂时不把以下内容作为核心差异化：

- 再造一个通用 Agent Framework；
- 再造一个低代码 Dify；
- 单纯做 OpenAI-compatible Gateway；
- 单纯做 GPU Scheduler；
- 单纯做 Model Serving UI。

AI Cloud 的差异化应落在：**Task-centric、Execution-first、Policy-native、Model/Framework/Cloud/Hardware-neutral、Continuous-learning-ready。**

## 15. 下一步工程输出

建议从本文进一步派生正式设计，而不是直接把研究结论当作产品承诺：

1. ADR：`Task` 是否升级为 AI Cloud 一级资源；
2. API Spec：TaskDefinition / ExecutionPlan / ExecutionRun；
3. Architecture：Control Plane / Execution Plane / Resource Plane；
4. Policy Spec：Execution Authorization；
5. Registry Spec：ModelProfile 2.0；
6. Roadmap：R1–R6 里程碑与验收指标；
7. PoC：一个真实 Coding/Operations Task 跑通 Task → Router → Agent → Tool → Runtime → Evaluation → Memory 闭环。
