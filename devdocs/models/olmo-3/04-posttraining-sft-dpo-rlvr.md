# 04. Post-training: SFT, DPO, and RLVR

**English** | [简体中文](04-posttraining-sft-dpo-rlvr.zh-CN.md)

## 1. Post-training branch structure

OLMo 3 exposes post-training as multiple inspectable stages instead of publishing only one final chat checkpoint.

```text
Long-context Base
        ↓
SFT checkpoint
        ↓
DPO checkpoint
        ↓
RLVR checkpoint
        ↓
Instruct or Think final model
```

The Instruct and Think paths share the stage vocabulary but use different data mixtures, objectives, output behavior, and evaluation priorities.

## 2. Supervised fine-tuning

SFT teaches response format, instruction following, reasoning demonstrations, tool conventions, and conversational behavior from curated examples.

Important evidence includes:

- exact SFT mixture revision;
- example formatting and chat template;
- sequence-length policy;
- sample weighting;
- loss masking;
- learning-rate and epoch configuration;
- parent Base checkpoint.

SFT changes model behavior significantly even when the architecture is unchanged.

## 3. Direct Preference Optimization

DPO uses preferred and rejected responses to shape behavior without requiring a separate online RL loop.

Its effects can include:

- improved instruction adherence;
- response-style changes;
- stronger preference alignment;
- regressions caused by preference-data bias;
- altered verbosity and refusal behavior.

AI Cloud must not assume that a DPO checkpoint has identical security and cost characteristics to its SFT parent.

## 4. Reinforcement Learning with Verifiable Rewards

RLVR is especially suitable where correctness can be checked deterministically or by a reliable verifier, such as:

- mathematics;
- code execution and tests;
- constrained instruction following;
- structured-output requirements;
- logic tasks with checkable answers.

The general loop is:

```text
prompt
→ model rollout
→ verifier or reward function
→ accepted/rejected or scalar reward
→ policy update
```

Open reproduction requires the prompt set, rollout configuration, verifier implementation, reward aggregation, training code, and intermediate checkpoints.

## 5. Think pathway

The Think pathway explicitly trains long reasoning behavior. Operational implications include:

- higher average output tokens;
- variable reasoning depth;
- more expensive failed attempts;
- sensitivity to maximum token limits;
- possible exposure of sensitive intermediate reasoning;
- need to distinguish reasoning content from final answer quality.

AI Cloud should route Think only when expected task value justifies the extra cost and latency.

## 6. Instruct pathway

The Instruct pathway targets broad usability and instruction compliance. It is generally a better default for:

- standard chat;
- extraction and transformation;
- ordinary tool orchestration;
- low-to-medium complexity enterprise tasks;
- workflows with strict latency budgets.

This is a routing hypothesis, not a production conclusion; it must be validated on local workloads.

## 7. OLMo 3.1 extension

Official release material states that OLMo 3.1 32B Think extends the strongest OLMo 3 RL run for an additional 21 days on 224 GPUs with additional passes over the Think RL dataset. This should be represented as an extended-RL descendant, not a new pre-training lineage.

OLMo 3.1 32B Instruct applies the Instruct pathway at 32B scale and preserves its own SFT, DPO, and final RL checkpoints.

## 8. Recommended AI Cloud route metadata

```yaml
postTraining:
  pathway: think
  stages:
    - sft
    - dpo
    - rlvr
  reasoningTrace: explicit
  expectedOutputTokens: high
  verifierDomains:
    - math
    - code
    - instruction-following
  parentCheckpoint: <pinned-base>
  datasetRevisions:
    sft: <pinned>
    dpo: <pinned>
    rlvr: <pinned>
```

## 9. Required comparative evaluation

Every stage should be evaluated against its parent for:

- task success;
- calibration and hallucination;
- safety and refusal behavior;
- output length and cost;
- latency distribution;
- tool-call validity;
- formatting reliability;
- regression by domain.

Final-model evaluation alone hides which stage introduced a gain or failure.