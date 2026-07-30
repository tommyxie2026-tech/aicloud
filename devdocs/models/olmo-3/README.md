# OLMo 3 / OLMo 3.1 Technical Architecture Study

**English** | [简体中文](README.zh-CN.md)

**Observation date:** 2026-07-30  
**Status:** Primary-source architecture and openness study  
**Scope:** Model architecture, complete model flow, Dolma 3 and Dolci data, training, post-training, checkpoints, evaluation, deployment, and AI Cloud integration

## Executive view

OLMo 3 is best understood not as one checkpoint but as an openly documented model-development flow.

```text
Dolma 3 large-scale pre-training
        ↓
Dolmino targeted mid-training
        ↓
Longmino long-context extension
        ↓
Base checkpoints
        ├── Instruct SFT → DPO → RLVR
        ├── Think SFT → DPO → RLVR
        └── RL-Zero experiments
```

The family includes 7B and 32B decoder-only Transformer models. Official model cards report:

| Model | Training tokens | Layers | Hidden size | Query heads | KV heads | Context |
|---|---:|---:|---:|---:|---:|---:|
| OLMo 3 7B | 5.93T | 32 | 4096 | 32 | 32 | 65,536 |
| OLMo 3 32B | 5.50T | 64 | 5120 | 40 | 8 | 65,536 |

OLMo 3.1 extends the 32B post-training branches:

- **OLMo 3.1 32B Think** continues the strongest OLMo 3 Think RL run for a substantially longer schedule;
- **OLMo 3.1 32B Instruct** applies the Instruct recipe to the 32B base family;
- the base pre-training lineage remains OLMo 3 32B.

## Why this project matters to AI Cloud

OLMo 3 provides a strong reference for a model registry that tracks a model as a graph of immutable artifacts rather than one mutable name.

```text
model family
+ exact base checkpoint
+ training-stage checkpoint
+ data-mix revision
+ code revision
+ evaluation configuration
+ post-training branch
+ license and provenance evidence
```

It is therefore the primary comparison model for the Kimi K3 study:

- Kimi K3 demonstrates frontier-scale sparse multimodal architecture and deployment complexity;
- OLMo 3 demonstrates high transparency across the model manufacturing flow;
- neither should be reduced to a single openness score.

## Reading order

| No. | English | 简体中文 |
|---:|---|---|
| 0 | [Scope, sources, and confidence](00-scope-sources-and-confidence.md) | [范围、来源与可信度](00-scope-sources-and-confidence.zh-CN.md) |
| 1 | [Architecture and model family](01-architecture-and-model-family.md) | [架构与模型家族](01-architecture-and-model-family.zh-CN.md) |
| 2 | [Open model flow and data](02-open-model-flow-and-data.md) | [开放 Model Flow 与数据](02-open-model-flow-and-data.zh-CN.md) |
| 3 | [Pre-training, mid-training, and long context](03-pretraining-midtraining-long-context.md) | [预训练、中期训练与长上下文](03-pretraining-midtraining-long-context.zh-CN.md) |
| 4 | [Post-training: SFT, DPO, and RLVR](04-posttraining-sft-dpo-rlvr.md) | [后训练：SFT、DPO 与 RLVR](04-posttraining-sft-dpo-rlvr.zh-CN.md) |
| 5 | [Checkpoints, evaluation, and reproducibility](05-checkpoints-evaluation-and-reproducibility.md) | [Checkpoint、评测与可复现性](05-checkpoints-evaluation-and-reproducibility.zh-CN.md) |
| 6 | [Inference, deployment, and AI Cloud](06-inference-deployment-and-aicloud.md) | [推理、部署与 AI Cloud](06-inference-deployment-and-aicloud.zh-CN.md) |
| 7 | [Comparison with Kimi K3](07-kimi-k3-comparison.md) | [与 Kimi K3 的对比](07-kimi-k3-comparison.zh-CN.md) |
| — | [References](references.md) | [参考资料](references.zh-CN.md) |

## Evidence boundary

This study does not claim that:

- every announced log or intermediate artifact is already available for every branch;
- public training data removes all upstream copyright or privacy questions;
- published checkpoints reproduce internal infrastructure behavior without environment matching;
- vendor benchmark improvements directly transfer to AI Cloud workloads;
- open training evidence automatically qualifies a model for production use.

Production routing still requires version-specific security, quality, capacity, cost, license, and ownership approval.