# Model Architecture Research

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

## Studies

| Model | Status | Observation date | Entry |
|---|---|---:|---|
| Kimi K3 | Initial architecture study | 2026-07-29 | [Open study](kimi-k3/README.md) |

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
