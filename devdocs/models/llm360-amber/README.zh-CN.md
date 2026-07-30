# LLM360 Amber 技术架构与预训练轨迹研究

[English](README.md) | **简体中文**

**观察日期：** 2026-07-30  
**研究角色：** 单次预训练运行的细粒度轨迹、数据序列映射与 Checkpoint 分析

## 1. 执行摘要

Amber 是 LLM360 发布的 6.7B 参数英语因果语言模型，采用 LLaMA-7B 风格 Dense 架构。它的主要目的不是成为 SOTA 最终模型，而是公开一条可分析的训练记录。

项目发布：

- 360 个模型 Checkpoint；
- 由 360 个 Tokenized Chunk 组成的完整训练数据序列；
- 约 1.259T 训练 Token；
- 训练代码和配置；
- 数据准备代码；
- W&B 日志和评测轨迹；
- 本研究检查的模型与数据 Artifact 使用 Apache-2.0。

## 2. 架构

|属性|数值|
|---|---:|
|参数量|6.7B|
|隐藏维度|4096|
|MLP Intermediate Size|11,008|
|层数|32|
|Attention Heads|32|
|最大序列长度|2048|
|词表|32,000|
|Normalization|RMSNorm|
|架构谱系|LLaMA-7B 风格|

Amber 是常规 Dense 模型。它的研究价值来自证据密度，而不是架构创新。

## 3. 数据混合

官方模型资料列出近似 Token 规模：

|来源|Token|
|---|---:|
|Arxiv|30.00B|
|Books|28.86B|
|C4|197.67B|
|RefinedWeb|665.01B|
|StarCoder|291.92B|
|StackExchange|21.75B|
|Wikipedia|23.90B|
|**总计**|**1,259.13B**|

数据准备流程下载来源数据，使用 LLaMA Tokenizer 进行 Tokenize，将序列拼接为 2049 Token 以生成移位的 Next-token Label，并将最终序列拆成 360 个 Chunk。

## 4. Checkpoint 与数据映射

```text
数据 Chunk 000 → Checkpoint 000
...
数据 Chunk N → Checkpoint N
...
数据 Chunk 359 → 最终 Checkpoint
```

精确实现仍需根据训练仓库核验，但项目核心设计是同时保留完整数据 Sequence 与密集 Checkpoint，使研究者可以把模型状态与已消费数据对应起来。

这支持：

- 记忆发生时点分析；
- 来源领域影响；
- 能力曲线重建；
- 异常检测；
- Loss Spike 调查；
- Benchmark 变化归因。

## 5. 与 Pythia 对比

|维度|Amber|Pythia|
|---|---|---|
|实验形态|一条主要 7B 训练运行|多个规模和去重版本|
|Checkpoint|360 个|每模型 154 个|
|数据轨迹|360 个准备完成的 Chunk 与完整 Sequence|可重建的共享 Token Order|
|最适合回答|这次训练中发生了什么？|训练行为如何随规模变化？|
|架构|LLaMA-7B 风格|GPT-NeoX 家族|

## 6. 局限

- 2048 Token 上下文不能代表现代长上下文架构。
- 模型以英语为中心。
- 现代 SFT、DPO、RLVR、Tool Use 与 Agent 安全不是重点。
- 来源语料代表较早一代 Web 与 Code 数据。
- 高开放度不能免除上游许可证和隐私审查。

## 7. AI Cloud 研究用途

Amber 应作为训练证据基准，而不是默认生产路由。

推荐用途：

1. 测试数百个 Checkpoint 的存储和索引。
2. 表达精确数据 Chunk 与 Checkpoint 关系。
3. 验证训练 Trace 可视化。
4. 关联来源领域暴露和评测变化。
5. 测试连续训练运行中的异常和回归检测。
6. 对比 Kimi K3 与 OLMo 3 的 Artifact 完整度。

## 8. 推荐 Registry 结构

```yaml
modelFamily: llm360-amber
run:
  id: amber-7b-main
  dataSequence:
    dataset: LLM360/AmberDatasets
    chunks: 360
    totalTokens: 1259.13B
checkpoints:
  pattern: ckpt_<000-359>
  parentRelation: sequential
  trainingCode: https://github.com/LLM360/amber-train
  dataPrepCode: https://github.com/LLM360/amber-data-prep
```

## 9. 一手来源

- https://huggingface.co/LLM360/Amber
- https://huggingface.co/datasets/LLM360/AmberDatasets
- https://github.com/LLM360/amber-train
- https://github.com/LLM360/amber-data-prep
- https://github.com/LLM360/Analysis360
- arXiv `2312.06550`

本研究不代表生产批准。