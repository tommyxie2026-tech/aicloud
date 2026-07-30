# Model Architecture Research

**English** | [简体中文](README.zh-CN.md)

This directory contains engineering studies of commercial, open-weight, and self-hosted AI models that may be connected to AI Cloud.

The purpose is not to reproduce vendor marketing. Each study examines:

- model architecture;
- modality and context behavior;
- training and post-training disclosures;
- inference and deployment architecture;
- benchmark methodology;
- license and supply-chain evidence;
- operational risks;
- compatibility with AI Cloud's Model Registry, routing, evaluation, FinOps, Tool Gateway, and Sandbox.

## Bilingual structure

Each model-study directory should contain paired files:

```text
README.md          English entry
README.zh-CN.md    Chinese entry
<topic>.md         English chapter
<topic>.zh-CN.md   Chinese companion
```

The two versions must preserve matching section numbers, evidence boundaries, references, and code examples.

## Comparison entry

- [Open Model Research Matrix](comparisons/open-model-research-matrix.md)
- [开放模型研究矩阵](comparisons/open-model-research-matrix.zh-CN.md)

The matrix compares architecture, data, training code, checkpoints, post-training, licensing, deployment barriers, and recommended AI Cloud research roles.

## Studies

| Model / project | Research role | Status | Observation date | English entry | 中文入口 |
|---|---|---|---:|---|---|
| Kimi K3 | Frontier sparse multimodal and long-context systems | Detailed bilingual architecture study | 2026-07-30 | [Open English study](kimi-k3/README.md) | [打开中文研究](kimi-k3/README.zh-CN.md) |
| OLMo 3 / OLMo 3.1 | End-to-end open Model Flow | **Detailed multi-chapter bilingual study** | 2026-07-30 | [Open English study](olmo-3/README.md) | [打开中文研究](olmo-3/README.zh-CN.md) |
| Apertus | Open multilingual and compliance-oriented model engineering | Bilingual technical study | 2026-07-30 | [Open English study](apertus/README.md) | [打开中文研究](apertus/README.zh-CN.md) |
| Pythia | Cross-scale pre-training dynamics and interpretability | Bilingual technical study | 2026-07-30 | [Open English study](pythia/README.md) | [打开中文研究](pythia/README.zh-CN.md) |
| LLM360 Amber | Dense checkpoint and data-sequence trace | Bilingual technical study | 2026-07-30 | [Open English study](llm360-amber/README.md) | [打开中文研究](llm360-amber/README.zh-CN.md) |
| Tülu 3 | Modern open SFT, DPO, and RLVR | Bilingual technical study | 2026-07-30 | [Open English study](tulu-3/README.md) | [打开中文研究](tulu-3/README.zh-CN.md) |
| Marin | Open model-laboratory operations and experiment provenance | Bilingual technical study | 2026-07-30 | [Open English study](marin/README.md) | [打开中文研究](marin/README.zh-CN.md) |

## Research classification

```text
Frontier architecture and serving
└── Kimi K3

End-to-end open model flow
├── OLMo 3 / 3.1
└── Apertus

Pre-training dynamics
├── Pythia
└── LLM360 Amber

Post-training
└── Tülu 3

Open research operations
└── Marin
```

## Promotion boundary

`devdocs/models/` contains implementation-oriented external technology research. A finding becomes a stable AI Cloud commitment only after it is reviewed and promoted into an ADR, design document, API specification, production policy, or roadmap item under `docs/`.

A model study must not be interpreted as production approval. Production admission requires version-specific license, provenance, security, evaluation, deployment, capacity, cost, and ownership evidence in the AI Cloud Model Registry.