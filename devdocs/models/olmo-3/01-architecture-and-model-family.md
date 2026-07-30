# 01. Architecture and Model Family

**English** | [简体中文](01-architecture-and-model-family.zh-CN.md)

## 1. Family topology

OLMo 3 separates base-model scale from post-training purpose.

```text
OLMo 3 7B Base
├── 7B Instruct
├── 7B Think
└── RL-Zero research branches

OLMo 3 32B Base
├── 32B Think
├── OLMo 3.1 32B Think
└── OLMo 3.1 32B Instruct
```

A final name therefore encodes at least:

- base parameter scale;
- pre-training lineage;
- post-training pathway;
- post-training duration or generation;
- checkpoint revision.

## 2. Core architecture

OLMo 3 uses a decoder-only Transformer. Official model cards report the following stable dimensions.

| Property | 7B | 32B |
|---|---:|---:|
| Layers | 32 | 64 |
| Hidden size | 4096 | 5120 |
| Query heads | 32 | 40 |
| KV heads | 32 | 8 |
| Context length | 65,536 | 65,536 |
| Pre-training tokens | 5.93T | 5.50T |

The 32B model's 40 query heads and 8 KV heads indicate grouped-query attention behavior, reducing KV-cache growth compared with a full one-to-one Q/KV-head design.

## 3. Architectural interpretation

OLMo 3 is not primarily differentiated by a novel sparse architecture. Its engineering contribution is the coordination of:

```text
conventional dense Transformer
+ carefully staged data curriculum
+ long-context extension
+ open post-training branches
+ dense checkpoint evidence
+ reproducible evaluation tooling
```

This matters because model performance is treated as an outcome of the complete system rather than architecture alone.

## 4. Base, Instruct, and Think are different products

### Base

The Base checkpoint is the foundation for continued training, domain adaptation, interpretability work, and controlled post-training experiments. It should not be routed as a chat model without an explicit prompt and safety design.

### Instruct

The Instruct path prioritizes broad instruction following, conversational usefulness, tool-oriented tasks, and controlled response behavior.

### Think

The Think path is trained to emit longer reasoning traces and improve math, coding, logic, and multi-step tasks. Its operational profile differs from Instruct:

- longer output sequences;
- higher latency variance;
- higher cost per successful task;
- more sensitivity to token budgets and stopping rules;
- additional privacy considerations if reasoning traces are stored.

## 5. OLMo 3.1 is a post-training evolution

OLMo 3.1 should not be represented as an unrelated base architecture. Official release material describes:

- 32B Think as an extension of the strongest prior RL run;
- 32B Instruct as the larger-scale application of the Instruct recipe;
- the same OLMo 3 32B base lineage.

AI Cloud should model this as lineage edges rather than flattening all names into independent models.

## 6. Recommended registry relationship

```yaml
modelFamily: olmo-3
baseModel:
  id: allenai/Olmo-3-1125-32B
  stage: long-context-base
variants:
  - id: allenai/Olmo-3-32B-Think
    pathway: think
    parentStage: rlvr
  - id: allenai/Olmo-3.1-32B-Think
    pathway: think
    parent: allenai/Olmo-3-32B-Think
    changeType: extended-rl
  - id: allenai/Olmo-3.1-32B-Instruct
    pathway: instruct
    parent: allenai/Olmo-3-32B
    changeType: full-32b-instruct-recipe
```

The exact repository revision must be stored beside every ID.