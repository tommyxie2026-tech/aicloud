# Tülu 3 Post-training Architecture Study

**English** | [简体中文](README.zh-CN.md)

**Observation date:** 2026-07-30  
**Research role:** Modern open post-training reference for SFT, preference optimization, RLVR, decontamination, and evaluation

## 1. Correct scope

Tülu 3 is not an end-to-end open pre-training project. Its principal models are post-trained from Llama 3.1 Base checkpoints, whose pre-training data and manufacturing process are not fully open.

Tülu 3's major contribution is opening the post-training layer:

```text
Llama 3.1 Base
→ curated and synthetic prompts
→ supervised fine-tuning
→ DPO with off-policy and on-policy preferences
→ RLVR with deterministic verifiers
→ final Tülu 3 model
→ standardized evaluation and decontamination
```

## 2. Model family

Official collections include 8B, 70B, and 405B families with visible stage checkpoints:

| Stage | 8B | 70B | 405B |
|---|---|---|---|
| Base | Llama 3.1 Base | Llama 3.1 Base | Llama 3.1 Base |
| SFT | Public Tülu SFT checkpoint | Public Tülu SFT checkpoint | Public Tülu SFT checkpoint |
| DPO | Public Tülu DPO checkpoint | Public Tülu DPO checkpoint | Public Tülu DPO checkpoint |
| Final | RLVR model | RLVR model | RLVR model |

Model licensing inherits the Llama 3.1 Community License rather than Apache-2.0.

## 3. Data architecture

Tülu 3 reports 939,344 curated prompts, with approximately 57% sourced from public resources and 43% synthetically generated.

Data design emphasizes:

- provenance and clear licenses;
- skill coverage across general tasks, math, code, and precise instruction following;
- persona-driven synthetic generation;
- evaluation decontamination;
- separate SFT, preference, and RLVR mixtures;
- on-policy generations for preference data.

This makes the data pipeline an explicit system component rather than an undocumented fine-tuning set.

## 4. SFT

SFT establishes instruction format and core skills. Reproduction evidence includes:

- prompt and completion datasets;
- mixture composition;
- chat templates;
- training commands and DeepSpeed configurations;
- data curation and decontamination scripts;
- stage checkpoint.

## 5. DPO

Tülu 3 investigated several preference methods and selected length-normalized DPO for the main recipe. Important conclusions reported by Ai2 include:

- more unique prompts improve preference training;
- new DPO prompts can outperform reusing only SFT prompts;
- on-policy preference data improves results;
- hyperparameter and length normalization choices materially affect comparisons.

## 6. RLVR

Reinforcement Learning with Verifiable Rewards replaces a learned reward model for selected tasks with a deterministic or programmatic verifier.

```text
rollout
→ answer matching / constraint verification / test execution
→ binary or scalar verifiable reward
→ policy update
```

Target domains include mathematics, instruction constraints, and other tasks with checkable outcomes. At 405B scale, official materials describe a distributed loop combining vLLM inference, weight synchronization, and large-scale training.

## 7. Evaluation and decontamination

Tülu 3 releases a standardized evaluation suite and decontamination approach. This is essential because post-training development can overfit public benchmarks through prompt selection or synthetic-data generation.

AI Cloud should preserve:

- development and unseen-test separation;
- prompt-template versions;
- contamination thresholds;
- stage-by-stage evaluation;
- generation and verifier configuration;
- negative results and rejected recipes.

## 8. Comparison with OLMo 3

| Dimension | Tülu 3 | OLMo 3 |
|---|---|---|
| Base pre-training | Inherited from Llama 3.1, not fully open | OLMo base flow openly documented |
| Post-training | Highly open | Highly open and integrated with full model flow |
| Scales | 8B, 70B, 405B | 7B, 32B |
| Main value | Post-training recipe research | End-to-end model manufacturing research |
| License | Llama 3.1 Community License | Inspected checkpoints Apache-2.0 |

## 9. AI Cloud research priorities

1. Reproduce an 8B SFT smoke run.
2. Register Base, SFT, DPO, and RLVR as separate lineage nodes.
3. Implement a verifier registry for math, code, and constraint tasks.
4. Compare DPO and RLVR stage deltas on enterprise workloads.
5. Track synthetic-data generator and teacher-model provenance.
6. Measure quality, safety, verbosity, latency, and successful-task cost at every stage.
7. Prevent production approval from being inherited automatically from the Base model.

## 10. Primary references

- https://allenai.org/blog/tulu-3-technical
- https://allenai.org/tulu
- https://github.com/allenai/open-instruct
- https://github.com/allenai/open-instruct/blob/main/docs/tulu3.md
- https://huggingface.co/collections/allenai/tulu-3-models
- https://github.com/allenai/olmes
- arXiv `2411.15124`

This study evaluates the post-training system, not the completeness of Llama 3.1 pre-training disclosure.