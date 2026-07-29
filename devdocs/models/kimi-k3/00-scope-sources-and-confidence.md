# 00. Research Scope, Sources, and Confidence

## 1. Research objective

This study answers five engineering questions:

1. What is the end-to-end technical architecture of Kimi K3?
2. Which parts are visible in released weights, configuration, source, and technical documentation?
3. Which reported mechanisms are essential to long context, sparse scaling, multimodality, and Agent behavior?
4. What infrastructure is required to train, serve, evaluate, and operate the model?
5. How should AI Cloud represent and govern Kimi K3 as a provider and model version?

## 2. Primary sources

The study prioritizes the following official sources:

| Source | Use in this study |
|---|---|
| MoonshotAI/Kimi-K3 GitHub repository | Official model summary, deployment recommendations, usage behavior, license, and technical report |
| Kimi K3 technical report, arXiv:2607.24653 | Architecture, training, post-training, infrastructure, evaluation, and limitations |
| moonshotai/Kimi-K3 on Hugging Face | Released weight inventory, configuration, custom Transformers implementation, processor, and quantization metadata |
| Kimi K3 License | Rights, Model-as-a-Service revenue threshold, attribution threshold, exemptions, and warranty terms |
| Official Kimi platform documentation | API compatibility, reasoning effort, preserved thinking history, tool use, structured output, and context behavior |

Secondary benchmarks or media summaries are not treated as architectural authorities.

## 3. Evidence labels

Each claim should be read using these labels.

### Published fact

A claim directly stated by Moonshot AI in the official report, model card, repository, or license.

Examples:

- 2.8T total parameters;
- 104B activated parameters;
- 93 layers;
- 896 routed experts;
- 16 routed experts selected per token;
- 1,048,576-token context length.

### Code-verifiable fact

A claim directly visible in released configuration or implementation files.

Examples:

- `KimiK3ForConditionalGeneration` is mapped through custom Transformers code;
- the vocabulary size is configured around 160K;
- the model exposes a MoonViT-based visual pathway;
- the public checkpoint is partitioned into 96 Safetensors shards;
- the repository declares MXFP4/MXFP8 compressed-tensor metadata.

### Engineering inference

A conclusion that follows from the published architecture but is not itself a vendor guarantee.

Examples:

- a 1.56 TB checkpoint makes ordinary single-node deployment impractical;
- hybrid KDA and MLA serving requires two different cache lifecycles;
- preserved reasoning history increases trace sensitivity and data-governance requirements;
- extreme expert sparsity shifts difficulty from arithmetic alone to communication, scheduling, and expert placement.

### Unknown or unverified

A claim that cannot be established from released material.

Examples:

- exact full pre-training corpus;
- complete data licensing chain;
- exact sample weights and curriculum ratios;
- complete training source code and orchestration configuration;
- optimizer checkpoints and intermediate model checkpoints;
- independently reproduced end-to-end benchmark results;
- exact production hardware topology used by Moonshot AI.

## 4. Public-material inventory

| Artifact | Public status | Notes |
|---|---|---|
| Full model weights | Public | Approximately 1.56 TB on Hugging Face, split into 96 Safetensors files |
| Model configuration | Public | Architecture, token IDs, text and vision configuration, quantization metadata |
| Transformers model implementation | Public | Custom configuration, text model, multimodal model, processors, and media utilities |
| Technical report | Public | Detailed architecture and systems disclosure |
| License | Public | Custom Kimi K3 License, not Apache-2.0 |
| Inference recipes | Partially public | Official recommendations and external engine recipes for vLLM, SGLang, and TokenSpeed |
| Full pre-training source code | Not established | The report describes methods, but the Kimi K3 repository is not a complete training stack |
| Complete training dataset | Not public | Data strategy is described at a category and method level |
| SFT and RL datasets | Not public | Environment classes and synthesis methods are described, not released as a complete corpus |
| Reward models and all reward functions | Not public | Verifiability and reward design are discussed, but not fully reproducible |
| Intermediate checkpoints | Not public | The released artifact is the final model checkpoint |
| Full official evaluation harness | Not fully public | Many benchmarks and harnesses are external; some internal suites are unavailable |

## 5. Source-version discipline

This study is a snapshot as of **2026-07-29**. Kimi K3 is newly released and the following may change quickly:

- model repository files;
- inference engine compatibility;
- quantization kernels;
- API pricing and limits;
- license interpretation guidance;
- independent evaluation results;
- hardware recipes.

Before production admission, AI Cloud should pin:

```text
model repository
+ exact revision
+ weight manifest
+ SHA-256 or stronger artifact digest
+ custom-code revision
+ inference-engine version
+ kernel version
+ license revision
+ evaluation-suite revision
```

## 6. Research limitations

This documentation is an architecture analysis, not a legal opinion, independent benchmark report, or hardware sizing guarantee.

Where the technical report presents a result as an improvement over Kimi K2, this study records it as a vendor-reported result unless independent reproduction is available.
