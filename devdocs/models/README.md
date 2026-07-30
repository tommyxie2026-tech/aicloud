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

## Studies

| Model | Status | Observation date | English entry | 中文入口 |
|---|---|---:|---|---|
| Kimi K3 | Initial architecture study with bilingual structure | 2026-07-29 | [Open English study](kimi-k3/README.md) | [打开中文研究](kimi-k3/README.zh-CN.md) |

## Suggested future structure

```text
models/
├── kimi-k3/
├── deepseek-*/
├── glm-*/
├── qwen-*/
├── mistral-*/
└── commercial-api-models/
```

A model study must not be interpreted as production approval. Production admission requires version-specific license, provenance, security, evaluation, deployment, and ownership evidence in the AI Cloud Model Registry.
