# Apertus Technical Architecture and Openness Study

**English** | [简体中文](README.zh-CN.md)

**Observation date:** 2026-07-30  
**Research role:** End-to-end open, multilingual, compliance-oriented model reference

## 1. Executive view

Apertus is a family of 8B and 70B decoder-only Transformer models developed by the Swiss AI Initiative. Official model cards describe:

- pre-training on approximately 15T tokens;
- a staged web, code, and mathematics curriculum;
- more than 1,000 supported languages, with current model cards stating 1,811 native languages;
- 65,536-token context;
- xIELU activation;
- AdEMAMix optimizer;
- supervised fine-tuning and QRPO-based alignment;
- open weights, open data reconstruction, training recipes, and intermediate checkpoints;
- Apache-2.0 model license for the inspected checkpoints.

## 2. Architecture

The 70B configuration exposes:

| Property | Value |
|---|---:|
| Layers | 80 |
| Hidden size | 8192 |
| Intermediate size | 43,008 |
| Attention heads | 64 |
| KV heads | 8 |
| Context | 65,536 |
| Vocabulary | 131,072 |
| Activation | xIELU |
| Normalization | RMSNorm with QK normalization |
| Position strategy | Llama-3-style RoPE scaling |

The architecture is dense rather than MoE. Grouped-query attention reduces KV-cache requirements relative to full multi-head KV storage.

## 3. Training and openness

```text
open/compliant source reconstruction
→ staged web/code/math pre-training
→ intermediate checkpoints
→ supervised fine-tuning
→ QRPO alignment
→ Base and Instruct releases
```

Apertus is especially valuable because the project links technical openness with data-rights and transparency documentation. The official materials reference:

- training-data reconstruction scripts;
- intermediate checkpoints exposed through repository branches;
- technical report and evaluation evidence;
- memorization analysis tooling;
- EU AI Act public-summary and code-of-practice documents.

## 4. Compliance design

The project states that its data process respects publisher opt-out signals and includes measures intended to reduce memorization. This is a stronger governance posture than simply publishing a final dataset name.

AI Cloud should nevertheless independently verify:

- the exact data reconstruction revision;
- source-license categories;
- personal-data handling and deletion updates;
- output-filter revisions where applicable;
- regional and sector-specific requirements.

## 5. Deployment

Official materials describe compatibility with:

- Transformers 4.56 or later;
- vLLM;
- SGLang;
- MLX for supported on-device scenarios;
- OpenAI-compatible serving interfaces.

The 8B variant is a practical multilingual research and restricted-deployment candidate. The 70B variant requires substantial accelerator memory and distributed serving.

## 6. Comparison with Kimi K3 and OLMo 3

| Dimension | Apertus | OLMo 3 | Kimi K3 |
|---|---|---|---|
| Main research value | Open multilingual and compliance engineering | Complete model-flow research | Frontier sparse multimodal systems |
| Scale | 8B / 70B dense | 7B / 32B dense | 2.8T sparse MoE |
| Data transparency | High | High | Low to limited |
| Post-training transparency | Moderate to high | High | Limited |
| Multilingual focus | Very high | Primarily English | Broad capability, not principally an openness/compliance project |
| Self-hosting accessibility | Moderate for 8B, high cost for 70B | Moderate for 7B, high for 32B | Extreme |

## 7. AI Cloud integration priorities

1. Register Base and Instruct as distinct immutable versions.
2. Pin data-reconstruction, training-code, and checkpoint revisions.
3. Add language-level evaluation and routing metadata.
4. Track deletion/output-filter evidence as renewable governance artifacts.
5. Test 65K retrieval, multilingual safety, and language-specific quality.
6. Treat 8B as the first practical deployment target; use 70B for controlled higher-capability evaluation.

## 8. Primary references

- https://huggingface.co/swiss-ai/Apertus-8B-2509
- https://huggingface.co/swiss-ai/Apertus-70B-2509
- https://huggingface.co/swiss-ai/Apertus-70B-Instruct-2509
- https://github.com/swiss-ai/pretrain-data
- https://github.com/swiss-ai/apertus-memorization
- arXiv `2509.14233`

A model study is not production approval. Exact revisions, artifacts, licenses, security, quality, capacity, and ownership must pass AI Cloud admission.