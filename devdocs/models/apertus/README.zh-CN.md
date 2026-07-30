# Apertus 技术架构与开放度研究

[English](README.md) | **简体中文**

**观察日期：** 2026-07-30  
**研究角色：** 端到端开放、多语言、合规导向模型参照

## 1. 执行摘要

Apertus 是 Swiss AI Initiative 开发的 8B 与 70B decoder-only Transformer 家族。官方模型卡披露：

- 预训练约 15T Token；
- Web、Code 和 Mathematics 分阶段课程；
- 支持超过 1000 种语言，当前模型卡称原生支持 1811 种语言；
- 65,536 Token 上下文；
- xIELU Activation；
- AdEMAMix Optimizer；
- 监督微调和基于 QRPO 的 Alignment；
- 开放权重、数据重建、训练配方与中间 Checkpoint；
- 本研究检查的 Checkpoint 使用 Apache-2.0。

## 2. 架构

70B 配置披露：

|属性|数值|
|---|---:|
|层数|80|
|隐藏维度|8192|
|Intermediate Size|43,008|
|Attention Heads|64|
|KV Heads|8|
|上下文|65,536|
|词表|131,072|
|Activation|xIELU|
|Normalization|RMSNorm 与 QK Normalization|
|位置策略|Llama-3 风格 RoPE Scaling|

该架构为 Dense 模型而不是 MoE。Grouped-Query Attention 相比完整多头 KV 存储可以降低 KV Cache 需求。

## 3. 训练与开放流程

```text
开放/合规来源重建
→ Web/Code/Math 分阶段预训练
→ 中间 Checkpoint
→ 监督微调
→ QRPO Alignment
→ Base 与 Instruct 发布
```

Apertus 的重要价值是把技术开放与数据权利、透明度文档结合起来。官方资料包含或引用：

- 训练数据重建脚本；
- 通过仓库 Branch 提供的中间 Checkpoint；
- 技术报告与评测证据；
- Memorization Analysis 工具；
- EU AI Act Public Summary 和 Code of Practice 文档。

## 4. 合规设计

项目声明数据流程尊重发布者 Opt-out 信号，并采用措施降低记忆风险。这比只发布最终数据集名称具有更强的治理意义。

AI Cloud 仍需独立验证：

- 精确数据重建 Revision；
- 来源许可证分类；
- 个人数据处理和删除更新；
- 适用时的 Output Filter Revision；
- 区域与行业要求。

## 5. 部署

官方资料说明兼容：

- Transformers 4.56 或更新版本；
- vLLM；
- SGLang；
- 支持场景下的 MLX；
- OpenAI-compatible Serving Interface。

8B 是现实的多语言研究和受限部署候选；70B 需要大量加速器内存和分布式服务。

## 6. 与 Kimi K3、OLMo 3 对比

|维度|Apertus|OLMo 3|Kimi K3|
|---|---|---|---|
|主要研究价值|开放多语言与合规工程|完整 Model Flow 研究|前沿稀疏多模态系统|
|规模|8B / 70B Dense|7B / 32B Dense|2.8T Sparse MoE|
|数据透明度|高|高|低至有限|
|后训练透明度|中高|高|有限|
|多语言重点|很高|主要为英语|能力广泛，但不是以开放合规为核心的项目|
|自托管可达性|8B 中等、70B 成本高|7B 中等、32B 较高|极高门槛|

## 7. AI Cloud 接入重点

1. 将 Base 与 Instruct 注册为不同不可变版本。
2. 固定数据重建、训练代码和 Checkpoint Revision。
3. 增加语言级评测与路由元数据。
4. 将删除与 Output Filter 证据作为需要更新的治理 Artifact。
5. 测试 65K 召回、多语言安全和各语言质量。
6. 先以 8B 作为现实部署目标，70B 用于受控高能力评测。

## 8. 一手来源

- https://huggingface.co/swiss-ai/Apertus-8B-2509
- https://huggingface.co/swiss-ai/Apertus-70B-2509
- https://huggingface.co/swiss-ai/Apertus-70B-Instruct-2509
- https://github.com/swiss-ai/pretrain-data
- https://github.com/swiss-ai/apertus-memorization
- arXiv `2509.14233`

模型研究不代表生产批准。精确 Revision、Artifact、许可证、安全、质量、容量和责任归属必须通过 AI Cloud 准入。