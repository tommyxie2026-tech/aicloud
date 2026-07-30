# 00. Scope, Sources, and Confidence

**English** | [简体中文](00-scope-sources-and-confidence.zh-CN.md)

## 1. Research question

This study asks four separate questions:

1. What is the OLMo 3 / 3.1 model architecture?
2. Which stages of model creation are externally inspectable?
3. Which published artifacts support reproducible experiments?
4. How should AI Cloud represent and govern this model flow?

## 2. Primary sources

The study prioritizes:

- Ai2 OLMo 3 release and OLMo 3.1 update;
- official OLMo 3 and OLMo 3.1 Hugging Face model cards;
- `allenai/OLMo-core` for distributed training;
- `allenai/open-instruct` for SFT, DPO, and RLVR;
- Dolma 3, Dolmino, Longmino, and Dolci dataset repositories;
- OLMES and official evaluation configurations;
- exact model and dataset revisions where available.

Secondary commentary is not used to establish architecture or openness claims when a primary source exists.

## 3. Confidence labels

| Label | Meaning |
|---|---|
| Confirmed | Directly stated in official documentation or visible in a released artifact |
| Strong inference | Supported by multiple official artifacts but not stated as one explicit claim |
| Limited | Announced or partially documented, but the complete artifact set was not verified |
| Unknown | No sufficient public evidence was found |

## 4. Stable facts used in this study

Official model cards identify:

- 7B and 32B base families;
- decoder-only Transformer architecture;
- 65,536-token context;
- staged pre-training;
- Dolma 3 base data, Dolmino mid-training, and Longmino long-context data;
- Dolci post-training datasets;
- Instruct and Think branches using SFT, DPO, and RLVR;
- Apache-2.0 model-card licensing for the inspected final checkpoints;
- explicit releases of stage checkpoints and training details.

The OLMo 3.1 update identifies two additional 32B final branches: Think and Instruct.

## 5. Important caveats

“Open” must be decomposed into independent dimensions:

```text
weight access
training-code access
data access
recipe visibility
checkpoint coverage
log availability
evaluation reproducibility
license clarity
production-system transparency
```

A project can score highly on scientific reproducibility while still withholding internal cluster credentials, private operational records, security incidents, or production control-plane details.

## 6. Revision discipline

AI Cloud research should pin:

- model repository revision;
- dataset revision;
- training-code commit;
- post-training-code commit;
- evaluation harness commit;
- container and dependency digests.

A floating model name such as `Olmo-3.1-32B-Think` is insufficient production evidence.