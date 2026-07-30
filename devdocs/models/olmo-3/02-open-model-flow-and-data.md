# 02. Open Model Flow and Data

**English** | [简体中文](02-open-model-flow-and-data.zh-CN.md)

## 1. Model Flow as the primary artifact

OLMo 3 treats the model as a sequence of inspectable transformations.

```text
raw and curated sources
→ Dolma 3 processing and mixture
→ large-scale pre-training
→ Dolmino mid-training
→ Longmino long-context extension
→ Base checkpoint
→ Dolci SFT
→ Dolci DPO
→ Dolci RLVR
→ Instruct / Think final checkpoints
```

A reproducibility claim is strongest when each arrow has:

- input dataset revision;
- transformation code revision;
- training configuration;
- parent and child checkpoint IDs;
- metrics and evaluation configuration;
- license and provenance evidence.

## 2. Dolma 3

Dolma 3 is the large-scale pre-training data foundation. The important engineering value is not only dataset access but the ability to inspect data construction, filtering, mixing, and later-stage specialization.

AI Cloud should distinguish:

```text
source corpus
processed corpus
training mixture
sample order or sampling policy
model-consumed token stream
```

These are not interchangeable provenance objects.

## 3. Dolmino

Ai2 describes Dolmino as a mid-training mixture built from a roughly 2.2T-token pool, with approximately 100B tokens sampled for targeted training. It emphasizes harder material such as:

- mathematics;
- science;
- code;
- instruction following;
- reading comprehension;
- reasoning traces.

Mid-training is therefore not ordinary fine-tuning. It continues next-token training while deliberately shifting the distribution toward capabilities that are underrepresented in broad web pre-training.

## 4. Longmino

Longmino extends context capability using approximately 50B training tokens sampled from a larger long-document pool and mixed with mid-training material.

Its purpose is not merely to change a configuration field. It trains the model to operate over:

- long reports;
- logs;
- multi-chapter documents;
- dispersed evidence;
- long-range dependencies.

AI Cloud must still independently test retrieval, reasoning, and positional robustness across the full context range.

## 5. Dolci

Dolci is the post-training suite. Separate mixtures support:

- supervised fine-tuning;
- preference optimization;
- reinforcement learning with verifiable rewards;
- reasoning, tool use, instruction following, math, coding, and conversation.

The separation of datasets by stage is important. A single `trainingData` field cannot describe the flow.

## 6. Recommended data-lineage object

```yaml
dataLineage:
  pretraining:
    dataset: allenai/Dolma3
    revision: <pinned>
    mixtureRevision: <pinned>
  midtraining:
    dataset: allenai/Dolmino
    tokenBudget: 100B
    sourcePoolApprox: 2.2T
  longContext:
    dataset: allenai/Longmino
    tokenBudgetApprox: 50B
    sourcePoolApprox: 639B
  posttraining:
    sft: <dolci-sft-revision>
    dpo: <dolci-dpo-revision>
    rlvr: <dolci-rl-revision>
```

## 7. Openness does not remove governance

Open data improves inspection but does not eliminate:

- upstream license variation;
- privacy and personal-data questions;
- source removal and revision drift;
- benchmark contamination;
- malicious or low-quality samples;
- regional data restrictions.

AI Cloud should store both the published data evidence and its own legal, security, and quality decision.