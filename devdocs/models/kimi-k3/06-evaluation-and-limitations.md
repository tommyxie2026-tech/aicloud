# 06. Evaluation and Limitations

## 1. Evaluation posture

Kimi K3's official report presents broad results across reasoning, coding, Agent, knowledge-work, and vision benchmarks. These results are useful evidence of intended capability, but they should be interpreted as **vendor-reported measurements under specific harnesses and inference settings**.

They are not automatically equivalent to:

- independent reproduction;
- enterprise workload performance;
- identical results under a different serving engine;
- identical results at lower reasoning effort;
- identical results without tools;
- cost-effective production performance.

## 2. Published evaluation configuration

The official model summary states that Kimi K3 benchmark results generally use:

- reasoning effort set to `max`;
- temperature set to `1.0`;
- top-p `0.95` for selected single-step tasks;
- top-p `1.0` for Agent tasks;
- tool augmentation for some benchmarks;
- benchmark-specific Agent harnesses;
- multiple-run averaging for selected multimodal tasks.

This configuration is part of the result. A score without the configuration is incomplete evidence.

## 3. Selected vendor-reported results

The official release reports strong results in areas such as:

- long-horizon software engineering;
- terminal and repository tasks;
- web research and browsing;
- MCP and tool-use tasks;
- office and spreadsheet tasks;
- visual documents and multimodal reasoning;
- finance, legal, and professional knowledge work.

The report also states that Kimi K3 trails the strongest proprietary systems overall in its comparison while outperforming other models on many evaluated tasks.

This study intentionally avoids copying the full leaderboard. The official release and report should remain the source of record for exact values.

## 4. Harness dependence

Model performance can change materially depending on the Agent harness.

The Kimi K3 report uses different harnesses for different models and tasks, including vendor-specific coding frameworks. In several coding benchmarks, Kimi K3 is evaluated with Kimi Code while competing models may use Codex, Claude Code, Terminus, or another framework.

### Consequence

A benchmark measures a system:

```text
base model
+ reasoning effort
+ system prompt
+ Agent harness
+ tool schema
+ retry policy
+ context management
+ sandbox
+ evaluator
```

It does not isolate the base model.

## 5. Tool-augmented versus unaugmented results

Some published cells report both:

- model-only or non-tool performance;
- performance with Python or general tool augmentation.

These must not be combined into one capability label. Tool augmentation adds:

- execution accuracy;
- retrieval or computation ability;
- new failure modes;
- additional cost;
- security requirements;
- dependence on Tool Gateway and Sandbox quality.

AI Cloud should store tool-augmented evaluation as a separate configuration and result.

## 6. Long-context evaluation

A declared context length of 1,048,576 tokens should be evaluated across several distinct properties.

| Property | Example question |
|---|---|
| Acceptance | Can the endpoint ingest the context without failure? |
| Retrieval | Can it find an exact fact at different positions? |
| Multi-hop reasoning | Can it combine distant evidence? |
| Instruction retention | Does an early instruction remain effective? |
| Recency bias | Does the model over-weight recent content? |
| Context compaction | What quality changes when history is compacted? |
| Tool-history stability | Can it preserve state after hundreds of calls? |
| Cost and latency | Is the task economically usable? |

The official report mentions context compaction in some evaluations. Results with compaction and results using the full context without management should be treated as separate system configurations.

## 7. Reasoning-effort evaluation

Kimi K3 supports `low`, `high`, and `max` reasoning effort. Production evaluation should measure all levels.

```text
quality
vs.
latency
vs.
output tokens
vs.
tool calls
vs.
actual task cost
```

A model that leads at `max` effort may not be the best route for ordinary tasks.

Recommended metrics:

- success rate by effort level;
- P50/P95 latency;
- output and reasoning-token use;
- tool-call count;
- retry count;
- cost per successful task;
- human intervention rate;
- safety-policy violation rate.

## 8. Agent evaluation

Long-horizon Agent evaluation should measure more than final completion.

### Capability metrics

- task success rate;
- time to first useful artifact;
- total completion time;
- correct tool selection;
- recovery from tool failure;
- verification quality;
- artifact correctness;
- rollback quality.

### Reliability metrics

- repeated-action loops;
- stalled trajectories;
- invalid tool arguments;
- context corruption;
- failure after compaction;
- unnecessary high-cost model calls;
- fallback frequency;
- incomplete finalization.

### Safety metrics

- unauthorized tool-call attempt rate;
- prompt-injection susceptibility;
- credential-handling violations;
- network-policy violations;
- workspace escape attempts;
- policy-bypass attempts;
- human-approval bypass attempts;
- audit completeness.

## 9. Multimodal evaluation

Multimodal results should be separated by task type:

- image perception;
- chart and diagram understanding;
- OCR and document structure;
- mathematical visual reasoning;
- multi-image comparison;
- video understanding;
- vision with Python or other tools.

The public Hugging Face interface and the technical report do not necessarily expose every modality through the same API contract. AI Cloud should evaluate the exact endpoint and processor combination.

## 10. Coding evaluation

For an AI Cloud coding scenario, benchmark success is not sufficient. The evaluation should include:

```text
issue interpretation
→ repository navigation
→ patch generation
→ build and tests
→ security scan
→ policy checks
→ human review
→ PR quality
```

Recommended metrics:

- patch acceptance rate;
- tests passed without weakening tests;
- regression rate;
- dependency-risk introduction;
- secret-exposure rate;
- changed-line efficiency;
- review-comment rate;
- rollback success;
- cost per merged change.

## 11. Reproducible evaluation record

AI Cloud should generate a stable configuration digest from:

```yaml
model:
  id: moonshotai-Kimi-K3
  revision: <pinned-revision>
  engine: <vllm-or-sglang-version>
  endpointMode: self-hosted-or-api
inference:
  reasoningEffort: max
  temperature: 1.0
  topP: 1.0
  maxOutputTokens: 32768
context:
  inputTokens: <measured>
  compactionPolicy: <version>
tools:
  schemaDigest: <digest>
  permissions: <policy-version>
agent:
  harness: <name-and-version>
  maxTurns: <number>
sandbox:
  imageDigest: <digest>
  networkPolicy: deny-by-default
dataset:
  suite: <suite>
  version: <version>
evaluator:
  version: <version>
```

The final result should store both the configuration digest and raw artifact digests.

## 12. Release gates

A Kimi K3 revision should not enter production routing only because its average benchmark score is high.

Suggested gates:

- minimum task success rate;
- maximum critical safety failures of zero;
- maximum cost per successful task;
- maximum P95 latency by context band;
- maximum human-intervention rate;
- maximum retry and fallback rate;
- no license or provenance admission failure;
- no unexplained regression against the previous approved revision.

## 13. Known limitations from public evidence

### Incomplete reproducibility

The full data, training stack, reward system, and internal benchmarks are not released.

### Vendor-reported comparisons

Some comparisons use different Agent harnesses or external leaderboard results collected at different times.

### Maximum-effort emphasis

Most headline results use maximum reasoning effort, which may hide latency and cost trade-offs.

### Internal benchmarks

Some important capability and safety suites are internal and cannot be independently inspected.

### Serving dependence

Performance can change with engine, kernel, quantization support, context management, and endpoint revision.

### Long-context uncertainty

Maximum accepted context does not guarantee uniform effective reasoning across the full window.

### Open-weight governance

Weights are public, but training-data provenance and complete reproducibility remain limited.

## 14. Recommended AI Cloud evaluation suites

```text
K3-CAPABILITY-001  structured reasoning
K3-CODE-001        repository issue-to-patch
K3-LONGCTX-001     1M-context retrieval and multi-hop reasoning
K3-AGENT-001       long-horizon tool workflow
K3-VISION-001      document and chart understanding
K3-SAFETY-001      prompt injection and privilege boundary
K3-COST-001        cost per successful task by effort
K3-FAILOVER-001    endpoint failure and fallback behavior
K3-LICENSE-001     evidence and admission validation
K3-CACHE-001       hybrid prefix-cache isolation and deletion
```

Every suite should be run for the exact production revision and deployment path.
