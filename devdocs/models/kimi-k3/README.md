# Kimi K3 Technical Architecture Study

**English** | [简体中文](README.zh-CN.md)

**Observation date:** 2026-07-29  
**Status:** Initial primary-source study  
**Scope:** Architecture, training disclosures, systems design, serving, evaluation, openness, license, and AI Cloud integration

## Executive view

Kimi K3 is a 2.8-trillion-parameter native multimodal Mixture-of-Experts model with approximately 104 billion activated parameters per token and a context window of 1,048,576 tokens.

Its overall design can be understood as scaling information flow along three axes:

```text
Sequence length  -> Kimi Delta Attention + periodic Gated MLA
Network depth    -> Attention Residuals
Model width      -> Stable LatentMoE
Native modality  -> MoonViT-V2 + lightweight projector
Post-training    -> SFT + multi-domain, multi-effort RL + policy consolidation
System support   -> KDA kernels + expert parallelism + persistent Agent RL state
Serving          -> hybrid cache management + specialized kernels + budget-aware scheduling
```

The most important architectural observation is that Kimi K3 is not merely a larger Transformer. It combines:

- mostly linear/recurrent long-sequence mixing through Kimi Delta Attention;
- periodic global attention through Gated Multi-head Latent Attention;
- selective retrieval across model depth through Attention Residuals;
- extreme sparse width through 896 routed experts, with 16 selected per token and two shared experts;
- native visual encoding through MoonViT-V2;
- deployment-aware quantization-aware post-training using MXFP4 expert weights and MXFP8 activations.

## Reading order

| No. | English | 简体中文 |
|---:|---|---|
| 0 | [Research scope, sources, and confidence](00-scope-sources-and-confidence.md) | [研究范围、来源与可信度](00-scope-sources-and-confidence.zh-CN.md) |
| 1 | [Architecture overview](01-architecture-overview.md) | [总体架构](01-architecture-overview.zh-CN.md) |
| 2 | [Hybrid attention and depth mixing](02-attention-and-depth-mixing.md) | [混合注意力与深度信息流](02-attention-and-depth-mixing.zh-CN.md) |
| 3 | [Stable LatentMoE and native multimodality](03-moe-and-native-multimodal.md) | [Stable LatentMoE 与原生多模态](03-moe-and-native-multimodal.zh-CN.md) |
| 4 | [Pre-training, post-training, and Agentic RL](04-training-posttraining-and-agentic-rl.md) | [预训练、后训练与 Agentic RL](04-training-posttraining-and-agentic-rl.zh-CN.md) |
| 5 | [Systems, inference, and deployment](05-systems-inference-and-deployment.md) | [系统、推理与部署](05-systems-inference-and-deployment.zh-CN.md) |
| 6 | [Evaluation and limitations](06-evaluation-and-limitations.md) | [评测与局限](06-evaluation-and-limitations.zh-CN.md) |
| 7 | [License, openness, and supply chain](07-license-openness-and-supply-chain.md) | [许可证、开放程度与供应链](07-license-openness-and-supply-chain.zh-CN.md) |
| 8 | [AI Cloud integration blueprint](08-aicloud-integration-blueprint.md) | [AI Cloud 接入蓝图](08-aicloud-integration-blueprint.zh-CN.md) |
| — | [References](references.md) | [参考资料](references.zh-CN.md) |

## Directory structure

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

## What this study does not claim

This study does not claim that:

- vendor-reported benchmarks have been independently reproduced;
- the released materials are sufficient to reproduce pre-training;
- public weights imply unrestricted commercial use;
- a 1M-token context window guarantees reliable recall over all positions;
- model capability makes direct infrastructure execution safe;
- Kimi K3 is already approved for AI Cloud production routing.

Production use must pass AI Cloud's version-specific admission, evaluation, security, capacity, cost, and ownership controls.
