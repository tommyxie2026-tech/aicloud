# 2026-09-25 Model Market Signals → aicloud Evolution Hypotheses

## 1. Purpose

This document records a set of strategic hypotheses for aicloud based on product boundaries, market signals, and architecture economics. It is not a claim that the future is predetermined.

The central question is:

> Which capabilities are sufficiently stable and product-relevant to deserve long-term architectural investment, and which should remain reversible bets?

aicloud currently targets the problem of turning a Goal into a governed, observable and verifiable Outcome by coordinating models, agents, tools, context and execution providers.

That framing is the starting point for the hypotheses below.

## 2. Facts vs hypotheses

### Observed facts / current product constraints

These are treated as current facts about the product or market environment:
- models, prices, APIs, licenses and benchmark positions change quickly;
- different tasks can favor different models, runtimes and deployment modes;
- enterprises may need public, private and self-hosted models at the same time;
- agentic tasks introduce runtime, tool, workspace, retry and policy concerns that are outside plain model inference;
- execution completion does not automatically prove task correctness;
- aicloud already contains Registry, Evaluation, Routing, Policy and ExecutionProvider concepts;
- computecloud is an optional execution provider and remains an independent product.

### Strategic hypotheses

These are not facts and must be continuously tested:
- H1: model/provider abstraction remains valuable because model choice will stay heterogeneous;
- H2: independent evaluation will remain necessary because vendor claims alone are insufficient for routing;
- H3: verified outcome is a better optimization target than benchmark score or token price alone;
- H4: outcome-aware routing can materially improve cost, quality or reliability;
- H5: normalized execution trajectories will become useful for evaluation and optimization;
- H6: continuous learning from trajectories can create durable product advantage;
- H7: external fine-tuning/post-training integration may become useful, but aicloud does not need to own model training.

## 3. Confidence levels

### High-confidence core

Invest now because these capabilities have value even if individual models or vendors change.

#### Registry
Why:
- model/provider/deployment metadata changes frequently;
- heterogeneous deployment is already a product requirement;
- licensing, economics and provenance affect enterprise use.

Investment gate:
- capability must support at least two materially different providers or deployment types.

#### Independent Evaluation
Why:
- benchmark claims, price and production behavior are not identical;
- routing and governance need evidence independent of vendor marketing.

Investment gate:
- evaluation result must influence a real routing, rollout or governance decision.

#### Policy / Governance
Why:
- data boundary, tool permissions and approval requirements exist independently of model quality.

Investment gate:
- policy must be enforceable, auditable and versioned.

#### Execution Contract
Why:
- agent execution has different lifecycle semantics from intelligence selection;
- execution backend should remain replaceable.

Investment gate:
- aicloud can run with more than one provider implementation and without computecloud-specific domain objects.

#### Verification
Why:
- ExecutionCompleted and task correctness are different states.

Investment gate:
- verifier must provide a deterministic or explicitly bounded judgment for at least one production-relevant task class.

## 4. Medium-confidence strategic bets

These deserve controlled investment but require measurable evidence before becoming foundational.

### H4 — Outcome-aware Router

Hypothesis:
Routing based on capability, evaluation evidence, cost, latency, risk and policy can outperform static provider selection.

Expected signal:
- higher Verified Outcome Rate;
- lower Cost / Verified Outcome;
- lower Time / Verified Outcome;
- lower retry or human intervention rate.

Falsification conditions:
- a single provider dominates nearly all relevant tasks for a sustained period;
- routing complexity adds more latency/cost than it saves;
- evaluation evidence is too noisy to improve routing decisions.

Investment gate:
Do not build advanced adaptive routing until offline or shadow evaluation shows a measurable advantage over static routing.

### H5 — Trajectory Platform

Hypothesis:
Normalized execution trajectories become valuable for regression analysis, routing, context selection and policy improvement.

Expected signal:
- repeated failure patterns can be identified;
- trajectory-derived features improve routing or recovery;
- replay/regression testing catches production-quality regressions.

Falsification conditions:
- trajectory data is too expensive or sensitive to retain;
- trajectory-derived features do not improve any measurable outcome;
- provider/runtime heterogeneity makes normalization impractical.

Investment gate:
Persist only the minimum normalized trajectory needed for evaluation first; expand only after a downstream consumer proves value.

## 5. Low-confidence options

These should remain optional and downstream.

### H6 — Continuous Optimization / Learning Loop

Hypothesis:
Verified trajectories can improve planner, router, context and policy behavior over time.

Expected signal:
- measurable improvement in Verified Outcome Rate or cost after controlled updates.

Falsification conditions:
- improvements are not statistically or operationally distinguishable from noise;
- feedback introduces instability or regressions.

Investment gate:
No self-modifying production behavior. All learned changes must pass offline evaluation and regression gates.

### H7 — Training / Post-training Integration

Hypothesis:
Verified production data may become useful for external fine-tuning, post-training or RL systems.

Falsification conditions:
- frontier/API models remain cheaper and better than custom adaptation;
- domain data volume/quality is insufficient;
- compliance or data provenance constraints outweigh benefits.

Investment gate:
aicloud should first export governed datasets and evaluate externally trained models. Owning a training stack is not required.

## 6. Stable variables vs volatile variables

### Volatile variables

~~~text
Model name
Model version
Benchmark position
Token price
Context window
API shape
Best provider
Hardware target
License terms
Deployment economics
~~~

### Relatively stable product questions

~~~text
What is the task?
What capability is required?
What data may leave the boundary?
What cost/latency budget applies?
What tools are permitted?
Where should execution happen?
Did execution finish?
Is the result correct?
Should failure retry or replan?
What evidence should influence the next decision?
~~~

Architecture principle:

> Turn volatile market choices into replaceable resources; invest in the stable decision and verification problems.

## 7. North-star architecture hypothesis

~~~text
Model / Provider Resources
          ↓
Evidence-rich Registry
          ↓
Independent Evaluation
          ↓
Intelligent Router
          ↓
Context / Tool / Agent Strategy
          ↓
Execution Provider
          ↓
Execution Result
          ↓
Verifier
          ↓
Verified Outcome
          ↓
Trajectory / Evaluation
          ↓
Optional Continuous Optimization
~~~

This is a hypothesis architecture, not a mandated end state.

## 8. Economics hypothesis

The platform should test whether optimization value moves through these levels:

~~~text
$/token
   ↓
$/execution
   ↓
$/successful task
   ↓
$/verified outcome
~~~

Core metrics:
- Verified Outcome Rate
- Cost / Verified Outcome
- Time / Verified Outcome
- Retry / Recovery Rate
- Policy Compliance Rate
- Quality Regression Rate

If Verified Outcome cannot be measured cheaply and reliably for a task class, outcome-aware optimization for that class should not be treated as mature.

## 9. Registry evolution

Model Registry should gradually capture:

### Capability
- reasoning / coding / tool use / computer use
- multimodal input/output
- realtime / streaming
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

### Openness / licensing

~~~text
Closed API
Open Weight
Open Model
Open Training
Open Science / Reproducible
~~~

Do not collapse openness into a single license field.

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

### Evidence / provenance
- benchmark source
- independent evaluation
- evaluator/version/date
- safety evaluation
- known regressions
- pricing observation date
- license observation date

## 10. Market intelligence as evidence input

Commercial/open model tracking should cover:
- releases and deprecations;
- capability evaluations;
- pricing/cache/batch economics;
- licenses and openness;
- deployment methods;
- ecosystem/cloud partnerships;
- industry adoption;
- safety/security/copyright/data controversies.

Market data must not directly change production routing.

~~~text
Market Signal
    ↓
Registry Candidate Update
    ↓
Independent Evaluation
    ↓
Evidence
    ↓
Routing / Rollout Decision
~~~

Vendor claims are evidence candidates, not authoritative truth.

## 11. Execution and verification boundary

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

Rules:
- ExecutionCompleted != OutcomeVerified.
- Worker/Attempt scheduling remains outside aicloud.
- computecloud is one optional provider, not aicloud's kernel.
- verifier may be deterministic tests, schema checks, policy checks, environment observation, bounded judge models or human approval depending on risk.

## 12. Trajectory hypothesis

A normalized trajectory may contain:

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

Initial allowed uses:
- regression analysis;
- evaluation datasets;
- router comparison;
- failure taxonomy;
- context and policy analysis.

Future uses require evidence:
- adaptive routing;
- planner optimization;
- synthetic data;
- post-training / RL.

## 13. Roadmap mapped by confidence

### Core / high confidence
- R1 Evidence-rich Registry
- R2 Independent / execution-aware Evaluation
- R4 Verified Execution
- Policy / Governance foundations

### Strategic bets / medium confidence
- R3 Outcome-aware Router
- R5 Trajectory Platform

### Options / low confidence
- R6 Continuous Optimization
- R7 Learning / Training Integration

This means R1-R7 are not equal commitments.

## 14. Decision review framework

Every major roadmap review should answer:

1. What changed in the external market?
2. Which observation is fact versus vendor claim?
3. Which aicloud hypothesis does it support or weaken?
4. Has confidence changed?
5. Is there production evidence?
6. Has an investment gate been met?
7. Should a capability move between Core, Strategic Bet and Option?
8. What should be stopped if evidence is negative?

Recommended artifact:

~~~text
Signal
→ Hypothesis
→ Evidence
→ Confidence
→ Falsification Test
→ Investment Gate
→ Decision
~~~

## 15. Architectural invariants

1. Model is a replaceable intelligence resource, not the product center.
2. Vendor benchmark claims are evidence candidates, not truth.
3. Registry stores provenance and time-sensitive economics.
4. Router optimizes execution strategy only when evidence proves value.
5. Execution Result and Verified Outcome remain separate.
6. Execution Provider internals remain outside aicloud.
7. Evaluation remains independent from Provider and Router.
8. Trajectory is governed data and must preserve provenance.
9. Training is optional; aicloud must deliver value without owning model training.
10. Long-term assumptions must remain falsifiable.

## 16. Product implication

Current product center:

~~~text
Governed hybrid model access
+ policy-aware agent workflows
~~~

High-confidence evolution:

~~~text
Governed Intelligence Orchestration
+ Independent Evaluation
+ Verified Execution
~~~

Potential long-term extension, subject to evidence:

~~~text
+ Outcome-aware Routing
+ Trajectory-driven Optimization
+ Optional Learning / Training Integration
~~~

The goal is not to predict the final AI platform architecture correctly in advance. The goal is to build aicloud so that market changes can be absorbed through evidence, reversible interfaces and measurable decisions rather than architecture rewrites.
