# References

## Primary sources

### Official repository

- MoonshotAI/Kimi-K3  
  https://github.com/MoonshotAI/Kimi-K3

Use for:

- official model introduction;
- published model dimensions;
- deployment recommendations;
- usage behavior;
- reasoning-effort fields;
- preserved thinking-history guidance;
- license link;
- technical report.

### Technical report

- Kimi K3: Open Frontier Intelligence, arXiv:2607.24653  
  https://arxiv.org/abs/2607.24653  
  https://arxiv.org/pdf/2607.24653

Key sections:

| Topic | Report section |
|---|---|
| Architecture overview | Section 2 and Figure 2 |
| Kimi Delta Attention | Section 2.1.1 |
| Gated MLA | Section 2.1.2 |
| Attention Residuals | Section 2.2 |
| Stable LatentMoE | Section 2.3 |
| Native Vision | Section 2.4 |
| Pre-training data and curriculum | Section 3 |
| SFT, RL, and policy consolidation | Section 4 |
| Deployment-aware QAT | Section 4.1.4 |
| KDA system co-design | Section 5.1 |
| MoE pre-training infrastructure | Section 5.2 |
| Million-token Agentic RL infrastructure | Section 5.3 |
| Inference and online serving | Section 5.4 |
| Public evaluation | Section 6.1 |
| Internal evaluation | Section 6.2 |

### Hugging Face model repository

- moonshotai/Kimi-K3  
  https://huggingface.co/moonshotai/Kimi-K3

Use for:

- weight-shard inventory;
- repository size;
- `config.json`;
- generation configuration;
- custom Transformers code;
- processors and media utilities;
- released checkpoint metadata;
- model-card usage examples.

Important files include:

```text
config.json
generation_config.json
configuration_kimi_k3.py
modeling_kimi_k3.py
modeling_kimi_linear.py
kimi_k3_processor.py
kimi_k3_vision_processing.py
media_utils.py
encoding_k3.py
model.safetensors.index.json
model-00001-of-000096.safetensors
...
model-00096-of-000096.safetensors
```

The exact file inventory may change; production records must pin a repository revision.

### License

- Kimi K3 License  
  https://github.com/MoonshotAI/Kimi-K3/blob/main/LICENSE

Use for:

- granted rights;
- copyright and notice requirements;
- Model-as-a-Service definition;
- US$20 million consecutive-12-month revenue condition;
- 100 million MAU and US$20 million monthly-revenue attribution conditions;
- internal-use and certified-partner exemptions;
- warranty and liability disclaimer.

### Official API and usage documentation

- Kimi platform  
  https://platform.kimi.ai

Use for the exact current API contract, including:

- OpenAI-compatible access;
- Anthropic-compatible access;
- `reasoning_effort` values;
- multimodal messages;
- structured output;
- tool choice and dynamic tools;
- context caching;
- current pricing, quotas, and regions.

API behavior and price are time-sensitive and must be checked at implementation time.

## Related open projects referenced by the report

### AgentENV

- https://github.com/kvcache-ai/AgentENV

The technical report references AgentENV as a microVM-oriented sandbox system for Agent workloads. Its public repository does not mean Moonshot AI's complete internal RL orchestration is open.

### vLLM

- https://github.com/vllm-project/vllm
- https://docs.vllm.ai

### SGLang

- https://github.com/sgl-project/sglang
- https://docs.sglang.ai

### TokenSpeed

- https://lightseek.org

Engine support must be verified against an exact version and Kimi K3 recipe.

## Source-quality hierarchy

Use sources in this order:

1. exact released files and configuration;
2. Kimi K3 License;
3. official technical report;
4. official model card and API documentation;
5. inference-engine official documentation;
6. independent benchmark reproduction;
7. reputable secondary reporting;
8. community comments and unverified deployment claims.

## Citation practice for future updates

When updating these documents:

- link the exact source;
- include the observation date;
- separate vendor claims from independent results;
- avoid copying large blocks of source text;
- pin model and code revisions for production statements;
- document contradictions between the report, model card, configuration, and observed runtime;
- move superseded claims into a change log rather than silently rewriting history.

## Snapshot note

This reference list reflects public material available on **2026-07-29**. Kimi K3 is newly released, so repository contents, inference recipes, documentation, and ecosystem support should be expected to evolve.
