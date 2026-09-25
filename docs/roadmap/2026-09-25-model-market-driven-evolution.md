# 2026-09-25 Model Market Signals → aicloud Evolution Direction

## 1. Why this update exists

The model market is shifting from isolated benchmark competition toward end-to-end execution economics. Frontier models are increasingly differentiated by agent execution, multimodal interaction, tool use, latency, deployment flexibility, governance, and the cost of producing a successful result.

aicloud should therefore avoid being optimized around the question:

> Which model has the highest benchmark score?

The platform should instead optimize:

> Which combination of intelligence, context, tools, execution runtime and verification can produce a verified outcome with the required quality, policy compliance, latency and cost?

This does not change the current product boundary overnight. It gives the existing Registry / Evaluation / Routing / Execution / Governance work a clearer long-term direction.

## 2. North-star evolution

~~~text
Model Gateway
    ↓
Model / Provider Registry
    ↓
Independent Evaluation
    ↓
Intelligent Router
    ↓
Context + Tool + Agent Strategy
    ↓
Execution Provider
    ↓
Verifier
    ↓
Verified Outcome
    ↓
Trajectory / Evaluation
    ↓
Continuous Improvement
~~~

The economic metric should evolve in stages:

~~~text
$/token
   ↓
$/execution
   ↓
$/successful task
   ↓
$/verified outcome
~~~

Supporting metrics:
- Verified Outcome Rate
- Cost / Verified Outcome
- Time / Verified Outcome
- Retry / Recovery Rate
- Policy Compliance Rate
- Quality Regression Rate

## 3. Registry must evolve beyond endpoint metadata

Model Registry should gradually capture a capability and governance profile.

### Capability
- reasoning / coding / tool use / computer use
- multimodal input/output
- realtime / streaming
- context window
- structured output
- agent/runtime compatibility
- domain capability

### Economics
- input/output/cache pricing
- measured token consumption
- tool/runtime cost
- latency and throughput
- task success rate
- cost per successful task
- cost per verified outcome

### Deployment
- hosted API
- private endpoint
- open-weight self-hosted
- container / Kubernetes / bare metal / edge
- hardware requirements and precision
- memory footprint and throughput profile

### Openness and licensing

Do not collapse openness into one `license` field.

~~~text
OpennessProfile
├── weights
├── architecture
├── tokenizer
├── training code
├── training data
├── post-training data
├── RL environment
├── evaluation assets
├── commercial-use terms
└── redistribution / derivative restrictions
~~~

Taxonomy:

~~~text
Closed API
Open Weight
Open Model
Open Training
Open Science / Reproducible
~~~

### Evidence / provenance
- benchmark source
- independent evaluation
- evaluator/version/date
- safety evaluation
- known regressions
- pricing observation date
- license observation date

## 4. Evaluation becomes independent infrastructure

Evaluation must not be a one-time model leaderboard.

~~~text
Model / Agent / Tool / Provider
           ↓
       Evaluation
           ↓
Capability Evidence
Cost Evidence
Latency Evidence
Safety Evidence
Task Success Evidence
           ↓
        Registry
           ↓
         Router
~~~

Evaluation targets should expand from model responses to complete executions.

Key distinction:

~~~text
Benchmark Score != Task Success != Verified Outcome
~~~

## 5. Intelligent Router evolution

Static provider selection should evolve toward:

~~~text
Task
× Capability
× Quality Evidence
× Cost
× Latency
× Risk
× Context
× Data Boundary
× Deployment
× Runtime Availability
→ Intelligence Strategy
~~~

The router may select not only a model, but a strategy:

~~~text
Model + Reasoning Effort
Model + Tool Set
Coding Agent
Private Model + Domain Context
Realtime Multimodal Model
Execution Provider
Fallback / Verification Strategy
~~~

This is Intelligence Scheduling. Worker/Attempt scheduling remains an Execution Provider concern.

## 6. Execution and verification become first-class

The computecloud integration reinforces the separation:

~~~text
aicloud:
Goal / Plan / Intelligence / Context / Policy
                ↓
        Execution Contract
                ↓
Execution Provider:
reliable runtime execution
                ↓
        Execution Result
                ↓
aicloud Verifier
                ↓
        Verified Outcome
~~~

`ExecutionCompleted` must never imply `OutcomeVerified`.

Verifier types may include:
- deterministic tests
- schema/constraint validation
- policy checks
- environment observation
- independent judge model where appropriate
- human approval for high-risk operations

## 7. Trajectory becomes a strategic data asset

A useful execution trajectory includes:

~~~text
Goal
→ Context
→ Plan / Strategy
→ Model / Agent / Tool selection
→ Actions
→ Tool / Environment results
→ Recovery / Retry
→ Artifacts
→ Verification
→ Outcome
→ Cost / Latency / Resource usage
~~~

Execution Providers provide runtime facts. aicloud attaches semantic meaning: goal, strategy, evaluation and outcome.

Trajectory should feed:
- Router optimization
- Planner improvement
- Context selection
- policy tuning
- regression analysis
- evaluation datasets
- future post-training / synthetic-data pipelines

The initial implementation should use trajectories for evaluation and routing before attempting model training.

## 8. Training and learning boundary

aicloud is still not initially a foundation-model training platform.

Long-term learning can evolve as:

~~~text
Production Execution
→ Verified Trajectory
→ Evaluation Dataset
→ Synthetic / Curated Data
→ Optional Fine-tuning / Post-training / RL
→ New Model Version
→ Independent Evaluation
→ Registry
→ Controlled Rollout
~~~

Important emerging training assets:

~~~text
Data + Compute + Parameters
            +
Environment + Verifier + Trajectory
~~~

Training remains optional and downstream. The immediate moat is better execution/evaluation data, not owning a pretraining stack.

## 9. Market intelligence should become a platform input

Track commercial and open models continuously across:
- releases and deprecations
- capability evaluations
- pricing/cache/batch economics
- licenses and openness
- deployment methods
- ecosystem/cloud partnerships
- industry adoption
- safety/security/copyright/data controversies

Do not directly convert vendor claims into routing decisions. Market observations become candidates for independent evaluation.

~~~text
Market Signal
    ↓
Registry Candidate Update
    ↓
Independent Eval
    ↓
Evidence
    ↓
Routing Policy
~~~

## 10. Roadmap adjustment

### R1 — Evidence-rich Registry
Extend model/provider metadata with capability, economics, deployment, openness/license and provenance.

### R2 — Execution-aware Evaluation
Measure task success, tool/runtime behavior, latency, cost and verifier outcomes.

### R3 — Outcome-aware Router
Route using capability + evaluation evidence + cost + latency + risk + policy, not model name alone.

### R4 — Verified Execution
Make ExecutionProvider + Verifier + OutcomeStatus a standard execution path. Continue computecloud as one optional provider.

### R5 — Trajectory Platform
Persist normalized execution trajectories and correlate Goal → Execution → Verification → Outcome.

### R6 — Continuous Optimization
Use trajectories to improve routing, planning, context and policy. Add controlled experiments and regression gates.

### R7 — Optional Learning / Training Integration
Export verified datasets to external fine-tuning/post-training systems; evaluate resulting models before registry promotion.

## 11. Architectural invariants

1. Model is a replaceable intelligence resource, not the product center.
2. Vendor benchmark claims are evidence candidates, not truth.
3. Registry stores provenance and time-sensitive economics.
4. Router optimizes execution strategy, not a permanent model ranking.
5. Execution Result and Verified Outcome remain separate.
6. Execution Provider internals remain outside aicloud.
7. Evaluation is independent from Provider and Router.
8. Trajectory is governed data and must preserve provenance.
9. Training is optional; aicloud must deliver value without owning model training.
10. The long-term optimization target is reliable, policy-compliant Verified Outcome.

## 12. Product implication

The long-term product center evolves from:

~~~text
Governed hybrid model access + policy-aware agent workflows
~~~

toward:

~~~text
Governed Intelligence Orchestration
+ Verified Execution
+ Continuous Optimization
~~~

The migration is incremental: existing Gateway, Registry, Evaluation, Router, Agent Workflow, Policy and Execution Provider work become the foundation rather than being replaced.
