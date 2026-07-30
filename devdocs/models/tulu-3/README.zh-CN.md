# Tülu 3 后训练架构研究

[English](README.md) | **简体中文**

**观察日期：** 2026-07-30  
**研究角色：** SFT、偏好优化、RLVR、去污染与评测的现代开放后训练参照

## 1. 正确范围

Tülu 3 不是端到端开放预训练项目。其主要模型基于 Llama 3.1 Base Checkpoint 进行后训练，而 Llama 3.1 的预训练数据和制造过程并未完全开放。

Tülu 3 的主要贡献是开放后训练层：

```text
Llama 3.1 Base
→ 精选与合成 Prompt
→ 监督微调
→ 使用 Off-policy 与 On-policy 偏好数据进行 DPO
→ 使用确定性 Verifier 进行 RLVR
→ 最终 Tülu 3 模型
→ 标准化评测与去污染
```

## 2. 模型家族

官方 Collection 包含 8B、70B 和 405B，并公开主要阶段 Checkpoint：

|阶段|8B|70B|405B|
|---|---|---|---|
|Base|Llama 3.1 Base|Llama 3.1 Base|Llama 3.1 Base|
|SFT|公开 Tülu SFT Checkpoint|公开 Tülu SFT Checkpoint|公开 Tülu SFT Checkpoint|
|DPO|公开 Tülu DPO Checkpoint|公开 Tülu DPO Checkpoint|公开 Tülu DPO Checkpoint|
|Final|RLVR 模型|RLVR 模型|RLVR 模型|

模型许可证继承 Llama 3.1 Community License，而不是 Apache-2.0。

## 3. 数据架构

Tülu 3 报告共整理 939,344 个 Prompt，其中约 57% 来自公开资源，43% 为内部合成。

数据设计重点包括：

- 来源和清晰许可证；
- 通用任务、数学、代码、精确指令遵循等技能覆盖；
- Persona-driven Synthetic Generation；
- 评测去污染；
- 独立 SFT、Preference 与 RLVR 混合；
- 使用 On-policy Generation 构建偏好数据。

这使数据流水线成为显式系统组件，而不是一个不可见的 Fine-tuning Dataset。

## 4. SFT

SFT 建立指令格式和核心能力。复现证据包括：

- Prompt 与 Completion 数据；
- 混合组成；
- Chat Template；
- 训练命令与 DeepSpeed 配置；
- 数据整理与去污染脚本；
- 阶段 Checkpoint。

## 5. DPO

Tülu 3 研究了多种偏好方法，并在主要配方中采用 Length-normalized DPO。Ai2 报告的重要发现包括：

- 更多独立 Prompt 有助于偏好训练；
- 新 DPO Prompt 可能优于只复用 SFT Prompt；
- On-policy Preference Data 能提高结果；
- Hyperparameter 和长度归一化显著影响算法比较。

## 6. RLVR

Reinforcement Learning with Verifiable Rewards 在特定任务中使用确定性或程序化 Verifier，代替 Learned Reward Model。

```text
Rollout
→ Answer Matching / Constraint Verification / Test Execution
→ Binary 或 Scalar Verifiable Reward
→ Policy Update
```

目标领域包括数学、指令约束和其他具有可检查结果的任务。在 405B 规模，官方资料描述了结合 vLLM 推理、权重同步和大规模训练的分布式循环。

## 7. 评测与去污染

Tülu 3 发布标准化评测套件和去污染方法。后训练开发可能通过 Prompt 选择或合成数据生成过拟合公开 Benchmark，因此这一层非常重要。

AI Cloud 应保存：

- Development 与 Unseen Test 分离；
- Prompt Template Version；
- 污染阈值；
- 分阶段评测；
- Generation 与 Verifier 配置；
- 负结果和被放弃的配方。

## 8. 与 OLMo 3 对比

|维度|Tülu 3|OLMo 3|
|---|---|---|
|基础预训练|继承 Llama 3.1，未完全开放|OLMo Base Flow 公开说明|
|后训练|高度开放|高度开放并与完整 Model Flow 集成|
|规模|8B、70B、405B|7B、32B|
|主要价值|后训练配方研究|端到端模型制造研究|
|许可证|Llama 3.1 Community License|本研究检查的 Checkpoint 为 Apache-2.0|

## 9. AI Cloud 研究重点

1. 复现一次 8B SFT Smoke Run。
2. 将 Base、SFT、DPO 和 RLVR 注册为独立谱系节点。
3. 建立 Math、Code 和 Constraint Task 的 Verifier Registry。
4. 在企业工作负载上比较 DPO 与 RLVR 阶段差异。
5. 跟踪合成数据 Generator 与 Teacher Model 来源。
6. 在每个阶段测量质量、安全、输出长度、延迟和成功任务成本。
7. 禁止从 Base 模型自动继承生产批准。

## 10. 一手来源

- https://allenai.org/blog/tulu-3-technical
- https://allenai.org/tulu
- https://github.com/allenai/open-instruct
- https://github.com/allenai/open-instruct/blob/main/docs/tulu3.md
- https://huggingface.co/collections/allenai/tulu-3-models
- https://github.com/allenai/olmes
- arXiv `2411.15124`

本研究评估后训练系统，不代表 Llama 3.1 预训练披露完整。