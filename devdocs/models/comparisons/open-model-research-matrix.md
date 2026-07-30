# Open Model Research Matrix

**English** | [简体中文](open-model-research-matrix.zh-CN.md)

**Observation date:** 2026-07-30

## 1. Purpose

This matrix compares projects by the engineering evidence they expose, not by one benchmark score or an ambiguous `open-source` label.

## 2. Technical and openness matrix

| Project | Architecture / scale | Data | Training code and recipe | Checkpoints | Post-training | License posture | Primary AI Cloud role |
|---|---|---|---|---|---|---|---|
| Kimi K3 | 2.8T sparse MoE, ~104B active, native multimodal, ~1M context | Limited disclosure | Partial architecture and method disclosure | Final release emphasis | Limited reproducibility | Custom license with scale conditions | Frontier MoE, multimodal, long-context, complex serving study |
| OLMo 3 / 3.1 | 7B / 32B dense decoder-only, 65K context | Dolma 3, Dolmino, Longmino, Dolci | OLMo-core, Open Instruct, staged recipes | Major base, mid, long-context, SFT, DPO, RLVR stages | Highly open | Inspected final checkpoints Apache-2.0 | Primary end-to-end open Model Flow reference |
| Apertus | 8B / 70B dense, multilingual, 65K context | Open reconstruction and compliance process | Training details and recipes published | Intermediate checkpoints available | SFT and QRPO disclosed | Apache-2.0 inspected checkpoints | Multilingual, data-rights, compliance, sovereign deployment study |
| Pythia | 14M–12B GPT-NeoX suite | Pile variants, pre-tokenized order reproducible | GPT-NeoX configs and reproduction tools | 154 per model | Not a modern post-training project | Apache-style open research stack by artifact | Cross-scale learning dynamics and interpretability laboratory |
| LLM360 Amber | 6.7B LLaMA-style dense model, 2K context | 1.259T prepared sequence in 360 chunks | Training, data prep, logs, configs | 360 | Limited | Apache-2.0 | Single-run pre-training trace and data/checkpoint mapping |
| Tülu 3 | Llama 3.1 8B / 70B / 405B post-training families | Open SFT, preference, RLVR mixtures | Open Instruct recipes and commands | SFT, DPO, final RLVR stages | Primary focus; highly open | Llama 3.1 Community License | Modern post-training and verifier research |
| Marin | Evolving dense and MoE experiments; 8B/32B published work | Open experiment-specific data artifacts | Experiment declarations, infrastructure, reports, failures | Run-dependent | Evolving SFT/RL/distillation work | Artifact-specific | Open model-lab operations and Experiment Registry reference |

## 3. Best project by research question

| Research question | Preferred project | Reason |
|---|---|---|
| How should an end-to-end open model flow be represented? | OLMo 3 | Clear pre-training, mid-training, long-context, and post-training lineage |
| How can open models incorporate data rights and multilingual governance? | Apertus | Open/compliant data reconstruction and multilingual focus |
| How do behaviors emerge across scale and training time? | Pythia | Shared data order across many scales and 154 checkpoints |
| What happened at each point in one pre-training run? | Amber | Dense checkpoint and data-chunk sequence |
| How should SFT, DPO, and RLVR be implemented and evaluated? | Tülu 3 | Open data, code, recipes, verifiers, and evaluation |
| How should an open model laboratory operate? | Marin | Public hypotheses, PRs, execution, metrics, failures, and follow-ups |
| How should frontier sparse multimodal models be served and governed? | Kimi K3 | Extreme MoE, multimodality, long context, and serving complexity |

## 4. Openness layers

```text
Layer 1 — Product
weights, configuration, tokenizer, inference code

Layer 2 — Recipe
training data, mixtures, configs, SFT/DPO/RL objectives

Layer 3 — Experimental evidence
checkpoints, logs, data order, evaluation traces, negative results

Layer 4 — Factory tooling
training framework, orchestration, data processing, verifier and serving systems

Layer 5 — Production operations
credentials, customer traffic, incident history, capacity and security control plane
```

No project fully opens every Layer 5 detail. The meaningful comparison is how much of Layers 1–4 can be independently inspected and reproduced.

## 5. Recommended AI Cloud research sequence

```text
1. Pythia / Amber
   Validate checkpoint lineage, data mapping, and evaluation sensitivity.

2. Tülu 3
   Validate SFT, DPO, RLVR, verifier, and post-training evidence.

3. OLMo 3
   Integrate the complete Model Flow into Registry, Evaluation, and FinOps.

4. Apertus
   Add multilingual, compliance, deletion, and data-rights governance.

5. Marin
   Add Experiment Registry, negative-result preservation, and open-lab workflow.

6. Kimi K3
   Apply the governance system to a less transparent, much larger serving target.
```

## 6. Registry dimensions

All studies should map into common fields:

```yaml
modelResearch:
  modelFamily: <id>
  exactRevision: <commit-or-tag>
  architectureClass: <dense-or-moe>
  modalities: []
  contextWindow: <tokens>
  openness:
    weights: <level>
    data: <level>
    trainingCode: <level>
    checkpoints: <level>
    postTraining: <level>
    evaluation: <level>
  licenseEvidence: <reference>
  lineageEvidence: []
  deploymentProfiles: []
  unknowns: []
  recommendedUse: research-only
```

## 7. Decision principle

The most open model is not automatically the best production model, and the most capable model is not automatically the best governed model.

AI Cloud should optimize for:

```text
workload success
+ explainable lineage
+ security boundaries
+ recoverability
+ total successful-task cost
+ license and provenance confidence
+ operational ownership
```

Every production decision remains version-specific.