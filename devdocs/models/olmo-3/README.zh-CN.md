# OLMo 3 / OLMo 3.1 技术架构研究

[English](README.md) | **简体中文**

**观察日期：** 2026-07-30  
**状态：** 基于一手资料的架构与开放度研究  
**范围：** 模型架构、完整 Model Flow、Dolma 3 与 Dolci 数据、训练、后训练、Checkpoint、评测、部署及 AI Cloud 接入

## 执行摘要

OLMo 3 最适合被理解为一条公开记录的模型研发流程，而不只是一个最终 Checkpoint。

```text
Dolma 3 大规模预训练
        ↓
Dolmino 定向中期训练
        ↓
Longmino 长上下文扩展
        ↓
Base Checkpoint
        ├── Instruct SFT → DPO → RLVR
        ├── Think SFT → DPO → RLVR
        └── RL-Zero 实验
```

该家族包含 7B 和 32B 的 decoder-only Transformer。官方模型卡披露：

|模型|训练 Token|层数|隐藏维度|Query Heads|KV Heads|上下文|
|---|---:|---:|---:|---:|---:|---:|
| OLMo 3 7B | 5.93T | 32 | 4096 | 32 | 32 | 65,536 |
| OLMo 3 32B | 5.50T | 64 | 5120 | 40 | 8 | 65,536 |

OLMo 3.1 主要扩展 32B 后训练分支：

- **OLMo 3.1 32B Think** 在 OLMo 3 最强 Think RL 运行基础上继续更长时间训练；
- **OLMo 3.1 32B Instruct** 将 Instruct 配方应用到 32B 基础模型；
- 两者的基础预训练谱系仍来自 OLMo 3 32B。

## 为什么它对 AI Cloud 重要

OLMo 3 为 Model Registry 提供了一个重要参照：模型不应只是一个可变名称，而应是一张不可变 Artifact 图。

```text
模型家族
+ 精确基础 Checkpoint
+ 训练阶段 Checkpoint
+ 数据混合版本
+ 代码 Revision
+ 评测配置
+ 后训练分支
+ 许可证与来源证据
```

因此，它是 Kimi K3 研究最重要的对照模型：

- Kimi K3 展示前沿超大稀疏多模态架构和复杂部署；
- OLMo 3 展示跨模型制造流程的高度透明；
- 两者都不应被压缩为一个简单的“开放度分数”。

## 阅读顺序

|编号|中文|English|
|---:|---|---|
| 0 | [范围、来源与可信度](00-scope-sources-and-confidence.zh-CN.md) | [Scope, sources, and confidence](00-scope-sources-and-confidence.md) |
| 1 | [架构与模型家族](01-architecture-and-model-family.zh-CN.md) | [Architecture and model family](01-architecture-and-model-family.md) |
| 2 | [开放 Model Flow 与数据](02-open-model-flow-and-data.zh-CN.md) | [Open model flow and data](02-open-model-flow-and-data.md) |
| 3 | [预训练、中期训练与长上下文](03-pretraining-midtraining-long-context.zh-CN.md) | [Pre-training, mid-training, and long context](03-pretraining-midtraining-long-context.md) |
| 4 | [后训练：SFT、DPO 与 RLVR](04-posttraining-sft-dpo-rlvr.zh-CN.md) | [Post-training: SFT, DPO, and RLVR](04-posttraining-sft-dpo-rlvr.md) |
| 5 | [Checkpoint、评测与可复现性](05-checkpoints-evaluation-and-reproducibility.zh-CN.md) | [Checkpoints, evaluation, and reproducibility](05-checkpoints-evaluation-and-reproducibility.md) |
| 6 | [推理、部署与 AI Cloud](06-inference-deployment-and-aicloud.zh-CN.md) | [Inference, deployment, and AI Cloud](06-inference-deployment-and-aicloud.md) |
| 7 | [与 Kimi K3 的对比](07-kimi-k3-comparison.zh-CN.md) | [Comparison with Kimi K3](07-kimi-k3-comparison.md) |
| — | [参考资料](references.zh-CN.md) | [References](references.md) |

## 证据边界

本研究不声称：

- 每个分支宣布的日志和中间 Artifact 都已全部发布；
- 训练数据公开即可消除所有上游版权与隐私问题；
- 公开 Checkpoint 在不匹配环境时能够复现内部基础设施行为；
- 厂商 Benchmark 提升会直接转化为 AI Cloud 任务提升；
- 训练证据开放即可自动获得生产批准。

进入生产路由仍需完成精确版本的安全、质量、容量、成本、许可证和责任归属审查。