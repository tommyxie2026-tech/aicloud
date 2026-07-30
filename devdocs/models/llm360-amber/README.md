# LLM360 Amber Technical Architecture and Pre-training Trace Study

**English** | [简体中文](README.zh-CN.md)

**Observation date:** 2026-07-30  
**Research role:** Fine-grained single-run pre-training trace, data-sequence mapping, and checkpoint analysis

## 1. Executive view

Amber is a 6.7B-parameter English causal language model from LLM360. It uses a LLaMA-7B-style dense architecture and is intentionally released as a transparent training record rather than a state-of-the-art final model.

The project publishes:

- 360 model checkpoints;
- the fully prepared data sequence in 360 tokenized chunks;
- approximately 1.259T training tokens;
- training code and configurations;
- data preparation code;
- W&B logs and evaluation trajectories;
- Apache-2.0 licensing for the inspected model and data artifacts.

## 2. Architecture

| Property | Value |
|---|---:|
| Parameters | 6.7B |
| Hidden size | 4096 |
| MLP intermediate size | 11,008 |
| Layers | 32 |
| Attention heads | 32 |
| Maximum sequence length | 2048 |
| Vocabulary | 32,000 |
| Normalization | RMSNorm |
| Architecture lineage | LLaMA-7B style |

Amber is a conventional dense model. Its research value comes from evidence density, not architecture novelty.

## 3. Data mixture

Official model material lists the following approximate token totals:

| Source | Tokens |
|---|---:|
| Arxiv | 30.00B |
| Books | 28.86B |
| C4 | 197.67B |
| RefinedWeb | 665.01B |
| StarCoder | 291.92B |
| StackExchange | 21.75B |
| Wikipedia | 23.90B |
| **Total** | **1,259.13B** |

The preparation pipeline downloads source datasets, tokenizes them with the LLaMA tokenizer, concatenates sequences to 2049 tokens for shifted next-token labels, and divides the final sequence into 360 chunks.

## 4. Checkpoint-to-data mapping

```text
data chunk 000 → checkpoint 000
...
data chunk N → checkpoint N
...
data chunk 359 → final checkpoint
```

The precise implementation should be verified from the training repository, but the project's core design is to preserve both the full data sequence and dense checkpoints so researchers can connect model state with consumed data.

This supports:

- memorization timing analysis;
- source-domain influence;
- capability-curve reconstruction;
- anomaly detection;
- loss-spike investigation;
- benchmark change attribution.

## 5. Comparison with Pythia

| Dimension | Amber | Pythia |
|---|---|---|
| Experiment shape | One principal 7B run | Many scales and dedup variants |
| Checkpoints | 360 | 154 per model |
| Data trace | 360 prepared chunks, full sequence | Reconstructable shared token order |
| Best question | What changed during this run? | How does training behavior scale? |
| Architecture | LLaMA-7B style | GPT-NeoX family |

## 6. Limitations

- 2048-token context does not represent modern long-context architectures.
- The model is English-centric.
- Modern SFT, DPO, RLVR, tool use, and Agent safety are not the primary focus.
- The source corpus reflects an earlier generation of web and code data.
- Full openness does not remove upstream license and privacy review requirements.

## 7. AI Cloud research use

Amber should be used as a training-evidence benchmark, not a default production route.

Recommended uses:

1. Test storage and indexing of hundreds of checkpoints.
2. Model exact data-chunk and checkpoint relationships.
3. Validate training-trace visualization.
4. Correlate source-domain exposure with evaluation movement.
5. Test anomaly and regression detection across a continuous run.
6. Compare artifact completeness against Kimi K3 and OLMo 3.

## 8. Recommended registry shape

```yaml
modelFamily: llm360-amber
run:
  id: amber-7b-main
  dataSequence:
    dataset: LLM360/AmberDatasets
    chunks: 360
    totalTokens: 1259.13B
checkpoints:
  pattern: ckpt_<000-359>
  parentRelation: sequential
  trainingCode: https://github.com/LLM360/amber-train
  dataPrepCode: https://github.com/LLM360/amber-data-prep
```

## 9. Primary references

- https://huggingface.co/LLM360/Amber
- https://huggingface.co/datasets/LLM360/AmberDatasets
- https://github.com/LLM360/amber-train
- https://github.com/LLM360/amber-data-prep
- https://github.com/LLM360/Analysis360
- arXiv `2312.06550`

The study does not constitute production approval.