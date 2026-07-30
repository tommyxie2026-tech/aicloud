# 05. Checkpoints, Evaluation, and Reproducibility

**English** | [简体中文](05-checkpoints-evaluation-and-reproducibility.zh-CN.md)

## 1. Checkpoints are experimental evidence

OLMo 3 publishes checkpoints from major stages rather than treating only the final model as meaningful.

```text
pre-training checkpoint
→ mid-training checkpoint
→ long-context checkpoint
→ SFT checkpoint
→ DPO checkpoint
→ RLVR checkpoint
→ extended OLMo 3.1 checkpoint
```

This enables causal questions such as:

- Did code capability improve during mid-training or RLVR?
- Did instruction following improve while factual calibration regressed?
- Did long-context extension affect short-context quality?
- Did extended RL improve reasoning at the cost of latency or verbosity?

## 2. Checkpoint identity

A checkpoint identity should include:

```yaml
checkpoint:
  modelRepository: allenai/Olmo-3.1-32B-Think
  revision: <immutable-revision>
  stage: rlvr-extended
  parentRevision: <parent>
  trainingCodeCommit: <commit>
  dataRevision: <revision>
  configDigest: <sha256>
  weightManifestDigest: <sha256>
```

The Hugging Face branch or tag is useful, but the immutable commit and artifact digest are the production authority.

## 3. Evaluation layers

OLMo research uses multiple evaluation tools and task suites. AI Cloud should separate:

| Layer | Purpose |
|---|---|
| Training-time evaluation | Detect divergence and capability trends |
| Standard academic benchmarks | Compare public checkpoints under a declared harness |
| Stage-delta evaluation | Attribute gains and regressions to one training transition |
| System evaluation | Measure serving, tool use, structured output, and long-context behavior |
| Business workload evaluation | Measure successful-task cost and operational value |
| Security evaluation | Test misuse, prompt injection, data leakage, and tool-boundary behavior |

## 4. Reproducible evaluation record

```yaml
evaluationRun:
  modelRevision: <pinned>
  harness:
    name: olmes
    commit: <pinned>
  tasks:
    manifestDigest: <sha256>
  promptTemplateDigest: <sha256>
  generation:
    temperature: 0
    maxTokens: 4096
    reasoningMode: think
  tools:
    enabled: false
  environment:
    imageDigest: <sha256>
    accelerator: <type>
  resultDigest: <sha256>
```

Without this configuration, two scores with the same benchmark name may not be comparable.

## 5. OLMo 3.1 benchmark interpretation

Ai2 reports that extended RL improved OLMo 3.1 32B Think on math, logic, instruction-following, coding, and multi-step tasks. These results support a hypothesis that longer RL continued to produce gains.

They do not establish:

- equal gains on enterprise workloads;
- lower cost per successful task;
- improved safety;
- improved factuality outside tested domains;
- identical results under another inference stack.

## 6. Reproducibility levels

| Level | Evidence |
|---|---|
| L0 | Final weights only |
| L1 | Weights plus architecture and model card |
| L2 | Training code and high-level data description |
| L3 | Pinned data, config, stage checkpoints, and evaluation harness |
| L4 | Logs, environment, data order, optimizer state, and independently repeated run |

OLMo 3 targets a much higher level than ordinary open-weight releases, but individual branches should be assessed separately. “Logs coming soon” or an announced artifact must remain `pending` until verified.

## 7. Stage-gate policy

A new OLMo checkpoint should not replace an existing production route automatically. Required gates include:

- no critical safety regression;
- workload success improvement or justified specialization;
- bounded P95 latency and memory;
- acceptable cost per successful task;
- compatible tokenizer and chat template;
- verified artifact and license evidence;
- fallback route retained until a soak period completes.

## 8. Research value

The principal value of OLMo 3 for AI Cloud is the ability to build evaluation methods against a visible model lineage, then apply those methods to less transparent models such as Kimi K3.