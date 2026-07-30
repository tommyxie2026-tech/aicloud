# Kimi K3 技术架构研究

[English](README.md) | **简体中文**

**观察日期：** 2026-07-29  
**状态：** 初始一手来源研究  
**范围：** 架构、训练披露、系统设计、推理服务、评测、开放程度、许可证及 AI Cloud 接入

## 执行摘要

Kimi K3 是一个总参数约 2.8 万亿、每个 Token 激活约 1040 亿参数、上下文窗口为 1,048,576 Token 的原生多模态混合专家模型（Mixture-of-Experts，MoE）。

其整体设计可以理解为沿多个维度同时扩展信息流：

```text
序列长度  → Kimi Delta Attention + 周期性 Gated MLA
网络深度  → Attention Residuals
模型宽度  → Stable LatentMoE
原生模态  → MoonViT-V2 + 轻量投影器
后训练    → SFT + 多领域/多推理强度 RL + 策略整合
系统支撑  → KDA 内核 + 专家并行 + 持久化 Agent RL 状态
在线服务  → 混合缓存管理 + 专用内核 + 预算感知调度
```

最重要的架构判断是：Kimi K3 并非简单地放大标准 Transformer，而是组合了：

- 以 Kimi Delta Attention 为主的线性/递归式长序列混合；
- 周期性 Gated Multi-head Latent Attention 全局注意力；
- 通过 Attention Residuals 对网络深度中的早期表示进行选择性检索；
- 896 个路由专家、每 Token 选择 16 个专家和 2 个共享专家形成的极端稀疏宽度；
- MoonViT-V2 原生视觉编码；
- 面向部署的量化感知后训练，专家权重采用 MXFP4、激活采用 MXFP8。

## 阅读顺序

1. [研究范围、来源与可信度](00-scope-sources-and-confidence.zh-CN.md)
2. [总体架构](01-architecture-overview.zh-CN.md)
3. [混合注意力与深度信息流](02-attention-and-depth-mixing.zh-CN.md)
4. [Stable LatentMoE 与原生多模态](03-moe-and-native-multimodal.zh-CN.md)
5. [预训练、后训练与 Agentic RL](04-training-posttraining-and-agentic-rl.zh-CN.md)
6. [系统、推理与部署](05-systems-inference-and-deployment.zh-CN.md)
7. [评测与局限](06-evaluation-and-limitations.zh-CN.md)
8. [许可证、开放程度与供应链](07-license-openness-and-supply-chain.zh-CN.md)
9. [AI Cloud 接入蓝图](08-aicloud-integration-blueprint.zh-CN.md)
10. [参考资料](references.zh-CN.md)

## 双语目录

```text
kimi-k3/
├── README.md
├── README.zh-CN.md
├── 00-scope-sources-and-confidence.md
├── 00-scope-sources-and-confidence.zh-CN.md
├── 01-architecture-overview.md
├── 01-architecture-overview.zh-CN.md
├── 02-attention-and-depth-mixing.md
├── 02-attention-and-depth-mixing.zh-CN.md
├── 03-moe-and-native-multimodal.md
├── 03-moe-and-native-multimodal.zh-CN.md
├── 04-training-posttraining-and-agentic-rl.md
├── 04-training-posttraining-and-agentic-rl.zh-CN.md
├── 05-systems-inference-and-deployment.md
├── 05-systems-inference-and-deployment.zh-CN.md
├── 06-evaluation-and-limitations.md
├── 06-evaluation-and-limitations.zh-CN.md
├── 07-license-openness-and-supply-chain.md
├── 07-license-openness-and-supply-chain.zh-CN.md
├── 08-aicloud-integration-blueprint.md
├── 08-aicloud-integration-blueprint.zh-CN.md
├── references.md
└── references.zh-CN.md
```

## 本研究不作出的声明

本研究不声称：

- 厂商报告的 Benchmark 已经得到独立复现；
- 公开材料足以复现完整预训练；
- 公开权重意味着不受限制的商业使用；
- 100 万 Token 上下文保证模型在所有位置都能可靠召回和推理；
- 模型能力足以证明直接执行基础设施操作是安全的；
- Kimi K3 已经获得 AI Cloud 生产路由批准。

生产使用仍必须通过 AI Cloud 针对精确版本的准入、评测、安全、容量、成本和责任归属控制。
