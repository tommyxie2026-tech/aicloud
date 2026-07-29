# Kimi K3 Technical Architecture Study

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

1. [Research scope, sources, and confidence](00-scope-sources-and-confidence.md)
2. [Architecture overview](01-architecture-overview.md)
3. [Hybrid attention and depth mixing](02-attention-and-depth-mixing.md)
4. [Stable LatentMoE and native multimodality](03-moe-and-native-multimodal.md)
5. [Pre-training, post-training, and Agentic RL](04-training-posttraining-and-agentic-rl.md)
6. [Systems, inference, and deployment](05-systems-inference-and-deployment.md)
7. [Evaluation and limitations](06-evaluation-and-limitations.md)
8. [License, openness, and supply chain](07-license-openness-and-supply-chain.md)
9. [AI Cloud integration blueprint](08-aicloud-integration-blueprint.md)
10. [References](references.md)

## Directory structure

```text
kimi-k3/
├── README.md
├── 00-scope-sources-and-confidence.md
├── 01-architecture-overview.md
├── 02-attention-and-depth-mixing.md
├── 03-moe-and-native-multimodal.md
├── 04-training-posttraining-and-agentic-rl.md
├── 05-systems-inference-and-deployment.md
├── 06-evaluation-and-limitations.md
├── 07-license-openness-and-supply-chain.md
├── 08-aicloud-integration-blueprint.md
└── references.md
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
