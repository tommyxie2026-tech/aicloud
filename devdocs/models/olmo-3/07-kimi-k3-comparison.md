# 07. Comparison with Kimi K3

**English** | [简体中文](07-kimi-k3-comparison.zh-CN.md)

## 1. Comparison principle

Kimi K3 and OLMo 3 optimize different dimensions. A single label such as “more open” hides the engineering tradeoff.

| Dimension | Kimi K3 | OLMo 3 / 3.1 |
|---|---|---|
| Model scale | 2.8T total / ~104B active sparse MoE | 7B and 32B dense models |
| Modality | Native text and vision | Primarily text |
| Context | About 1M tokens | 65,536 tokens |
| Architecture novelty | KDA, periodic Gated MLA, Attention Residuals, extreme MoE | Dense decoder-only Transformer, staged training flow |
| Final weights | Public | Public |
| Inference/model code | Public custom code | Public standard and project code |
| Pre-training data | Not fully released | Dolma 3 flow published |
| Mid/long-context data | Limited disclosure | Dolmino and Longmino published |
| Post-training data | Limited disclosure | Dolci stage datasets published |
| Stage checkpoints | Limited compared with full manufacturing flow | Major base and post-training stages published |
| Training code and recipe | Partial methods, incomplete end-to-end reproduction | High openness through OLMo-core and Open Instruct |
| License | Custom Kimi K3 License with scale conditions | Inspected final checkpoints use Apache-2.0 |
| Self-hosting barrier | Extremely high | High for 32B, materially lower for 7B |
| Research value | Frontier sparse multimodal and serving architecture | End-to-end model-flow and reproducibility research |

## 2. Product, recipe, and factory comparison

```text
Kimi K3
Product: highly open
Recipe: partially disclosed
Factory: largely retained

OLMo 3
Product: open
Recipe: highly open
Factory tooling: substantially open
Internal production operations: still not fully public
```

OLMo 3 does not publish an entire organization's production environment. Its distinction is that the model-development path is sufficiently exposed for substantially deeper reproduction and branching.

## 3. Capability versus inspectability

Kimi K3 is the more relevant study for:

- extreme-scale sparse inference;
- expert parallelism;
- native multimodality;
- million-token systems;
- specialized attention kernels;
- frontier Agent behavior.

OLMo 3 is the more relevant study for:

- data lineage;
- training-stage attribution;
- SFT/DPO/RLVR reproduction;
- checkpoint graph design;
- evaluation configuration;
- open model governance;
- low-risk experimentation on earlier stages.

## 4. Deployment comparison

### Kimi K3

Kimi K3 self-hosting is an infrastructure program. It requires extreme weight storage, distributed expert execution, specialized kernels, sophisticated cache management, and high-capacity accelerators.

### OLMo 3

OLMo 3 7B can support smaller research and restricted deployment environments. The 32B variants still require serious accelerator capacity but remain much more accessible than a 2.8T MoE.

This makes OLMo 3 a better initial target for validating AI Cloud's self-hosted Provider Adapter, lineage registry, evaluation runner, and staged approval workflow.

## 5. License and provenance comparison

Kimi K3's custom license requires conditional business-scale checks. OLMo 3 final checkpoints inspected in this study are marked Apache-2.0, simplifying model-level permission analysis.

However, AI Cloud must separately review:

- dataset and source licenses;
- code licenses;
- inference runtime licenses;
- derivative-model obligations;
- privacy and regional constraints.

A permissive model license is not a complete supply-chain decision.

## 6. Evaluation strategy

Use OLMo 3 to calibrate the evaluation system because its stage lineage is visible:

```text
prove that the harness detects known stage differences on OLMo
        ↓
apply the same pinned harness to Kimi K3
        ↓
interpret Kimi results without assuming hidden training details
```

This reduces the risk that the evaluation platform itself is insensitive or incorrectly configured.

## 7. Recommended AI Cloud roles

| Role | Preferred candidate |
|---|---|
| Open research baseline | OLMo 3 7B Base |
| Post-training research | OLMo 3 / Tülu 3 |
| Transparent reasoning route | OLMo 3.1 32B Think |
| Lower-cost internal route | OLMo 3 7B Instruct, subject to evaluation |
| Frontier multimodal/MoE study | Kimi K3 |
| Extreme long-context study | Kimi K3, with strict capacity controls |
| Governance reference | OLMo 3 Model Flow |

## 8. Strategic conclusion

The models are complementary:

```text
OLMo 3 teaches AI Cloud how to govern a visible model-manufacturing process.
Kimi K3 tests whether that governance remains useful when much of the process is invisible and the serving system is far more complex.
```

A mature AI Cloud needs both research tracks.