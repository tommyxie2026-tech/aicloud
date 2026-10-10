# Model Market Intelligence, Pricing and Value Evaluation

## 1. Purpose

This document defines how aicloud should observe, normalize and evaluate the commercial and open-weight model market without coupling product behavior to vendor marketing or transient benchmark rankings.

The goal is to turn fast-changing external signals into governed evidence for Registry, Evaluation and Router decisions.

The monitoring loop is:

~~~text
Market Signal
   ↓
Normalized Model Identity
   ↓
Pricing / License / Deployment Observation
   ↓
Independent Evaluation
   ↓
Value Evidence
   ↓
Registry Candidate Update
   ↓
Routing / Rollout Decision
~~~

Market monitoring is an evidence source. It must never directly mutate production routing.

## 2. Model identity must be concrete

A marketing model name is insufficient for production decisions.

Every tracked commercial/API model should record:

~~~text
ModelIdentity
├── vendor
├── product_name
├── official_model_id
├── fixed_version_id
├── stable_alias
├── release_date
├── status                 # GA / preview / beta / deprecated
├── endpoint_type
├── context_window
├── max_output_tokens
├── modalities
├── supports_tools
├── supports_computer_use
├── supports_structured_output
├── supports_reasoning
├── supports_streaming
├── supports_batch
└── supports_cache
~~~

Rules:
- prefer the official API model ID over a marketing name;
- when both a floating alias and immutable/date-pinned ID exist, store both;
- production evaluation should preferably bind to an immutable or otherwise reproducible version;
- if a model ID is not public, store it as unavailable rather than guessing.

For open-weight models, identity should additionally capture:

~~~text
OpenWeightIdentity
├── official_repository
├── repository_provider    # Hugging Face / ModelScope / GitHub / etc.
├── revision_or_tag
├── weight_format
├── runtime_model_name
├── quantization_variants
└── artifact_availability
~~~

Announced, API Available, Weights Available, License Verified and Self-host Validated are separate states.

## 3. Pricing is a versioned profile, not two scalar fields

A model no longer has one stable input price and one stable output price.

Actual execution cost may depend on:
- model ID;
- service tier;
- context length;
- cache state;
- batch vs realtime;
- reasoning effort;
- region/data residency;
- tool usage;
- date/effective pricing period.

Therefore pricing should be modeled separately from Model.

~~~text
Model
  ↓
DeploymentProfile
  ↓
PricingProfile
  ↓
ObservedEconomics
~~~

Suggested schema:

~~~text
PricingProfile
├── pricing_profile_id
├── model_id
├── deployment_profile_id
├── currency
├── unit                    # normally per 1M tokens
├── input_price
├── cached_input_price
├── cache_read_price
├── cache_write_price
├── output_price
├── batch_discount
├── async_discount
├── context_thresholds[]
├── context_tier_prices[]
├── reasoning_billing
├── tool_prices[]
├── search_price
├── computer_use_price
├── service_tier
├── service_tier_multiplier
├── region
├── region_surcharge
├── data_residency_surcharge
├── effective_from
├── effective_to
├── source_type
├── source_uri
└── observed_at
~~~

Consumer subscription pricing must never be mixed with API token pricing.

## 4. Commercial/API token price monitoring

The daily market feed should normalize major commercial and hosted models to USD per 1M tokens where possible.

Minimum tracked fields:
- Vendor
- Product Name
- Official Model ID
- Version / Release Date
- Status
- Input $/1M
- Cached Input / Cache Read $/1M
- Cache Write $/1M
- Output $/1M
- Batch / Async discount
- Long-context threshold and surcharge
- Reasoning/Thinking billing semantics
- Tool/Search/Computer Use extra fees
- Context Window
- Max Output
- Region/Data Residency surcharge
- Priority/Fast/Ultrafast service tier
- Effective Date
- Source Type
- Change vs previous observation

Priority source order:
1. vendor pricing/API documentation;
2. first-party cloud/platform pricing;
3. trusted hosting provider pricing;
4. third-party reporting only as a temporary signal.

News articles and community posts should not overwrite an official price observation.

## 5. Price-change events

Pricing changes should be modeled as events so Registry and evaluation history remain reproducible.

~~~text
PricingChangeEvent
├── model_id
├── pricing_profile_id
├── dimension
├── old_value
├── new_value
├── percentage_change
├── effective_date
├── source
└── observed_at
~~~

Tracked dimensions should include:
- input/output price;
- cache read/write;
- batch discount;
- long-context premium;
- reasoning/thinking billing;
- fast/priority/ultrafast tier;
- region/data residency surcharge;
- free quota;
- tool/search/computer-use pricing;
- model deprecation;
- alias remapping.

A model alias change is economically relevant because the same alias may silently point to a different implementation or version.

## 6. Open-weight deployment cost is a separate economic model

Open-weight models must not be assigned a fabricated universal token price.

For each important open model, track:

~~~text
OpenDeploymentProfile
├── repository_id
├── license_profile
├── weights_available
├── total_parameters
├── active_parameters
├── context_window
├── recommended_runtime
├── precision
├── quantization
├── accelerator_requirement
├── hosted_api_prices[]
└── self_host_cost_observations[]
~~~

Self-host cost estimation is valid only when the assumptions are explicit.

Minimum assumptions:
- accelerator/GPU type and count;
- rental price or depreciation model;
- input and output throughput;
- batch/concurrency;
- utilization;
- precision/quantization;
- context length;
- region;
- runtime/inference engine.

A simplified estimate is:

~~~text
Compute cost per 1M tokens
≈
(accelerator_hour_cost × accelerator_count)
/
(tokens_per_hour / 1,000,000)
~~~

Production TCO must also consider:

~~~text
TCO =
Compute
+ Memory / Storage
+ Network
+ Runtime Operations
+ Availability Margin
+ Engineering
+ Observability
+ Security / Governance
~~~

If reliable throughput or utilization evidence is unavailable, report a qualitative TCO and do not generate a pseudo-precise $/1M number.

Hosted API pricing and self-hosted TCO must remain separate.

## 7. Economics should evolve beyond $/token

The platform should test optimization at progressively more meaningful levels:

~~~text
$/token
   ↓
$/execution
   ↓
Cost / Useful Task
   ↓
Cost / Successful Task
   ↓
Cost / Verified Outcome
~~~

Recommended observed metrics:

~~~text
ObservedEconomics
├── observed_input_tokens
├── observed_output_tokens
├── cache_tokens
├── reasoning_tokens
├── tool_cost
├── runtime_cost
├── retry_cost
├── verifier_cost
├── total_execution_cost
├── task_success
├── verified_outcome
├── latency
├── time_to_verified_outcome
└── cost_per_verified_outcome
~~~

A cheap token price does not imply a cheap task if the model:
- generates much longer outputs;
- retries more often;
- requires stronger verification;
- fails tool calls;
- causes higher runtime cost.

## 8. Comprehensive value Top 10

Aicloud may publish a daily Top 10 Value Models view, but it must be treated as an analytical projection over evidence, not a universal ranking.

The default scoring model is 100 points:

| Dimension | Weight | Meaning |
|---|---:|---|
| Task capability / quality | 30 | General, reasoning, coding, agent and domain quality |
| Cost efficiency | 25 | Typical task cost, not raw token price alone |
| Agent / Tool capability | 15 | Tool use, structured output, computer use, long-horizon execution |
| Latency / throughput | 10 | Verified latency and throughput evidence |
| Context / cache economics | 5 | Context capacity and effective reuse economics |
| Deployment / sovereignty | 5 | Hosted, private, self-hosted and local options |
| License / commercial usability | 5 | Commercial, redistribution and hosted-service rights |
| Evidence quality | 5 | Independent benchmark and production evidence quality |

Suggested record:

~~~text
ValueEvaluation
├── model_id
├── deployment_profile_id
├── evaluation_date
├── capability_score
├── cost_efficiency_score
├── agent_tool_score
├── latency_throughput_score
├── context_cache_score
├── deployment_score
├── license_score
├── evidence_score
├── total_score
├── confidence
├── best_fit_workloads[]
├── caveats[]
└── evidence_refs[]
~~~

Missing data must reduce confidence rather than be filled with assumptions.

## 9. Evidence hierarchy

Every capability claim in the ranking should be tagged as:

~~~text
Vendor Benchmark
Independent Benchmark
Internal / Production Evidence
Community Signal
~~~

Preferred order for decision weight:

~~~text
Production Evidence
    >
Independent Benchmark
    >
Vendor Benchmark
    >
Community Signal
~~~

This ordering is contextual, not absolute. A reproducible vendor benchmark can be stronger than an opaque third-party test.

For benchmark evidence, record:
- benchmark name/version;
- evaluator/harness;
- model version;
- date;
- config/reasoning effort;
- tool availability;
- result;
- provenance.

## 10. Scenario-specific winners

A single global Top 10 should not replace workload-aware evaluation.

The daily view should also identify best-value candidates for at least:
1. General purpose
2. Coding / Software Engineering
3. Agent / Tool Use
4. Long Context
5. High-throughput / Low-cost
6. Private / Local deployment
7. Multimodal
8. Specialist model such as Decision / Guard / Verifier

These are candidates for evaluation, not automatic Router defaults.

## 11. Router integration

Pricing and value data should feed Router only through evaluation evidence.

~~~text
Task
 ↓
Capability Requirement
 ↓
Policy / Data Boundary
 ↓
Candidate Models
 ↓
Deployment Profiles
 ↓
Evaluation Evidence
 ↓
Pricing Profiles
 ↓
Strategy Comparison
 ↓
Execution Strategy
 ↓
Verifier
 ↓
Observed Economics
~~~

The optimization unit becomes:

~~~text
Model
× DeploymentProfile
× ServiceTier
× ReasoningEffort
× CacheStrategy
× ToolSet
~~~

rather than just model_id.

Potential strategy contract:

~~~text
ExecutionStrategy
├── model_id
├── deployment_profile_id
├── reasoning_effort
├── service_tier
├── cache_strategy
├── tool_set
├── data_boundary
├── runtime
└── verifier
~~~

Advanced routing remains a medium-confidence strategic bet. It becomes justified only when it beats a static baseline on metrics such as:
- Verified Outcome Rate;
- Cost / Verified Outcome;
- Time / Verified Outcome;
- Retry Rate;
- Policy Compliance;
- Human Intervention Rate.

## 12. Registry implications

Registry should distinguish stable identity from volatile observations.

~~~text
Model
├── ModelIdentity
├── CapabilityProfile
├── OpennessProfile
├── LicenseProfile
├── ProvenanceProfile
│
├── DeploymentProfiles[]
├── PricingProfiles[]
├── EvaluationEvidence[]
└── SafetyEvidence[]
~~~

This prevents transient prices, benchmark scores or provider availability from becoming immutable model properties.

Recommended invariants:
1. product_name is not model_id.
2. A model may have multiple deployment profiles.
3. A deployment profile may have multiple pricing profiles over time.
4. Price observations are timestamped and source-backed.
5. Open-weight does not imply unrestricted commercial use.
6. Self-hosted TCO is not interchangeable with hosted token price.
7. Value scores are derived evidence, not source-of-truth metadata.
8. Production Router decisions are never based on market news alone.

## 13. Daily market output contract

A production-quality market report should contain:

### A. Market developments
- releases and deprecations;
- independent capability evidence;
- license changes;
- ecosystem/cloud/runtime partnerships;
- industry adoption;
- safety/security/copyright/regulatory disputes.

### B. Model identity table
- marketing name;
- official model ID;
- fixed version/alias;
- status;
- API/runtime capabilities.

### C. Commercial/API pricing table
- normalized token/cache/tool/tier economics.

### D. Open-weight deployment table
- repository ID;
- license;
- artifact availability;
- runtime;
- hosted API price;
- self-host TCO assumptions where evidence permits.

### E. Price changes
- old → new;
- percentage;
- effective date;
- affected model ID.

### F. Top 10 value models
- score;
- confidence;
- best-fit workload;
- main caveat.

### G. Scenario winners
- general;
- coding;
- agent;
- long-context;
- low-cost/high-throughput;
- private deployment;
- multimodal;
- specialist.

### H. aicloud hypothesis impact
Only update confidence when the new signal changes evidence for:
- Registry;
- Independent Evaluation;
- Policy / Authorization;
- Deployment Profile;
- Execution Contract;
- Guard;
- Verifier;
- Task-aware / Outcome-aware Router;
- Trajectory;
- Continuous Learning / Training.

## 14. Relationship to computecloud

This market intelligence design does not change the product boundary.

aicloud owns:
- intelligence resource identity;
- evaluation evidence;
- pricing/economics evidence;
- policy;
- strategy selection;
- verification.

Execution providers own:
- runtime execution;
- scheduling;
- worker lifecycle;
- provider-specific resource management.

computecloud remains one optional ExecutionProvider. No pricing, Registry or Router domain object should require computecloud internals.

## 15. Decision principle

The durable design principle is:

> Do not optimize for the cheapest token or the highest benchmark. Optimize for the least expensive, policy-compliant strategy that reliably produces a verified outcome.

This keeps volatile model economics replaceable while preserving a stable decision, execution and verification architecture.
