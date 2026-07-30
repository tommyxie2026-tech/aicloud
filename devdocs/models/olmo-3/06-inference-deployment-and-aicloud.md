# 06. Inference, Deployment, and AI Cloud

**English** | [简体中文](06-inference-deployment-and-aicloud.zh-CN.md)

## 1. Deployment profiles

OLMo 3 supports several operational profiles:

| Profile | Typical use |
|---|---|
| 7B Base | Research, continued training, controlled domain adaptation |
| 7B Instruct | Lower-cost chat, extraction, tool workflows |
| 7B Think | Cost-bounded reasoning experiments |
| 32B Base | Higher-capability research and domain adaptation |
| 32B / 3.1 Instruct | Broad enterprise instruction following |
| 32B / 3.1 Think | High-value reasoning, math, code, complex planning |

The correct route depends on task value, latency, memory, output-token budget, and safety—not model size alone.

## 2. Inference stack

Official model cards support standard Transformers loading. Production deployments may use optimized runtimes, but every runtime becomes part of the evaluated artifact set.

```text
model revision
+ tokenizer revision
+ chat template
+ runtime version
+ kernel version
+ quantization method
+ container digest
+ serving configuration
```

A change to quantization or runtime can change quality, latency, memory, and numerical behavior even when the model ID is unchanged.

## 3. Memory and capacity planning

A 32B BF16 checkpoint requires substantial accelerator memory before KV cache and runtime overhead. Quantization can reduce weight memory, but production sizing must include:

- model weights;
- KV cache for up to 65K context;
- temporary activations;
- tensor-parallel communication;
- batching and fragmentation;
- runtime reserve;
- concurrent requests;
- long Think outputs.

Capacity should be measured under the intended context and concurrency distribution, not a one-request smoke test.

## 4. Joint routing

Recommended routing dimensions:

```yaml
routingRequest:
  taskClass: code-reasoning
  modelFamily: olmo-3
  variantPreference:
    - instruct
    - think
  inferenceEffort: medium
  maxContext: 32000
  maxOutputTokens: 4000
  serviceTier: standard
  maxSuccessfulTaskCost: <budget>
```

The router should choose jointly:

- model scale;
- Instruct versus Think pathway;
- reasoning/output budget;
- quantized or full-precision endpoint;
- latency tier;
- fallback chain.

## 5. Provider and registry objects

```yaml
modelVersion:
  id: allenai/Olmo-3.1-32B-Think
  revision: <pinned>
  family: olmo-3
  parameters: 32B
  pathway: think
  contextWindow: 65536
  license: Apache-2.0
  parentModel: allenai/Olmo-3-32B-Think
  trainingEvidence:
    pretraining: <refs>
    midtraining: <refs>
    longContext: <refs>
    sft: <refs>
    dpo: <refs>
    rlvr: <refs>
endpoint:
  runtime: vllm-or-other
  runtimeVersion: <pinned>
  quantization: <none-or-scheme>
  health: healthy
  capacity:
    maxConcurrentRequests: <measured>
    maxContextAtTargetSLO: <measured>
```

## 6. Think-route controls

Think models require explicit controls:

- maximum output tokens;
- reasoning timeout;
- early-stop and verifier behavior;
- trace retention policy;
- sensitive reasoning redaction;
- retry limit;
- task-level cost ceiling.

A reasoning trace should not automatically be exposed to end users or stored indefinitely.

## 7. Fallback design

A representative chain is:

```text
OLMo 3.1 32B Think
→ OLMo 3.1 32B Instruct
→ OLMo 3 7B Instruct
→ commercial API or deterministic workflow
```

Fallback is allowed for capacity, timeout, transient runtime, and rate-limit failures. It should not silently bypass policy, data-residency, license, or safety requirements.

## 8. FinOps

Cost accounting should include:

- accelerator-seconds;
- reserved versus utilized capacity;
- input and output tokens;
- KV-cache occupancy;
- runtime startup and model loading;
- failed attempts and fallback attempts;
- tool and sandbox cost;
- evaluation cost;
- human review.

The primary metric is cost per successful task, not cost per generated token.

## 9. Production admission

Open code and data improve confidence, but production still requires:

- artifact digest and signature;
- independent workload evaluation;
- security and Agent boundary tests;
- license and dataset review;
- capacity and failure testing;
- observability and rollback;
- named operational owner.