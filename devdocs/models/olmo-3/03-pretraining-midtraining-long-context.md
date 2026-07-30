# 03. Pre-training, Mid-training, and Long Context

**English** | [简体中文](03-pretraining-midtraining-long-context.zh-CN.md)

## 1. Three-stage base-model training

OLMo 3 separates base-model development into three stages:

```text
Stage A: broad large-scale pre-training
Stage B: targeted mid-training
Stage C: long-context extension
```

This separation creates explicit intervention points. Researchers can compare a broad base model with a skill-focused model and a long-context model without conflating every change.

## 2. Stage A: broad pre-training

The initial stage builds general language capability from the Dolma 3 mixture. The primary outputs are:

- a base checkpoint lineage;
- training configurations;
- token-count evidence;
- loss and evaluation trajectories where available;
- data-mixture and processing references.

The 7B and 32B models do not report identical total token counts, so scale comparisons must account for both parameter count and training compute.

## 3. Stage B: mid-training

Mid-training continues causal language modeling on a harder, more capability-focused distribution. It is designed to improve areas such as math, science, code, instruction comprehension, and reasoning.

The engineering rationale is:

```text
broad general representation
+ concentrated high-value data
→ stronger capability before instruction tuning
```

This can be more stable than attempting to create every capability only during SFT or RL.

## 4. Stage C: long-context extension

The long-context phase extends the model to 65,536 tokens using long documents and relevant mid-training data.

Long-context training affects more than maximum sequence length:

- positional behavior;
- attention memory use;
- sequence packing;
- training batch composition;
- document boundary handling;
- long-range retrieval and reasoning.

A model may accept 65K tokens while still showing uneven recall or reasoning quality across positions. Context support and context quality are separate registry fields.

## 5. Checkpoint boundaries as governance boundaries

Each stage should produce an immutable registry identity.

```yaml
stages:
  - id: olmo3-32b-pretrain
    stage: broad-pretraining
  - id: olmo3-32b-midtrain
    parent: olmo3-32b-pretrain
    stage: targeted-midtraining
  - id: olmo3-32b-longcontext
    parent: olmo3-32b-midtrain
    stage: long-context-extension
```

This enables:

- targeted regression testing;
- rollback to the last acceptable stage;
- domain-specific branching;
- attribution of capability changes;
- separate safety and license evidence.

## 6. Reproduction requirements

A meaningful reproduction requires more than architecture and final weights:

- exact training code revision;
- optimizer and scheduler configuration;
- global batch and sequence packing;
- tokenizer revision;
- dataset and mixture revision;
- data order or sampling policy;
- precision and distributed strategy;
- checkpoint conversion process;
- evaluation schedule and harness.

## 7. AI Cloud evaluation plan

The stages should be compared on the same workload suite:

| Stage | Required evaluation focus |
|---|---|
| Broad pre-training | knowledge, language modeling, toxicity baseline, memorization |
| Mid-training | math, code, reading comprehension, instruction sensitivity |
| Long-context | retrieval by position, multi-document synthesis, latency, memory, lost-in-the-middle |

This turns the open Model Flow into actionable evidence rather than documentation only.