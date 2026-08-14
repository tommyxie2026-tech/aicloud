# 2026-08-13 Model Market Signals and Development Directives

## Purpose

This document converts the 2026-08-13 commercial and open-weight model-market signals into implementation constraints for AI Cloud. It extends the R5-R10 architecture references without adding vendor-specific dependencies.

## Executive Conclusion

Three market signals now deserve direct engineering treatment:

1. open-weight availability and commercial-use rights are becoming separate economic concepts;
2. model endpoints and prices are increasingly dynamic across deployment, region, cache, batch, service tier, context and inference effort;
3. hosted models have explicit lifecycle events such as preview, deprecation, replacement, quota reduction and retirement.

The durable execution chain remains:

```text
Task
  -> Policy / Requirements
  -> Capability
  -> ModelVersion
  -> Deployment
  -> Pricing / Capacity / Region / Service Tier
  -> Execution
  -> Evaluation
  -> CostEvent
  -> RouteOutcome
  -> Policy-bounded Feedback
```

## 1. License Economics Is a Routing Constraint

Open weights do not imply unrestricted or zero-cost commercial use. Commercial terms can differ by revenue threshold, hosted-service use, redistribution, derivatives or attribution requirements.

### Development directive

Extend license evidence so production admission can be machine-evaluated.

Minimum fields should support:

- weight availability and weight license;
- commercial-use allowed/conditional/forbidden;
- hosted-service allowed/conditional/forbidden;
- redistribution allowed/conditional/forbidden;
- derivative-work restrictions;
- attribution and notice requirements;
- revenue or usage threshold when present;
- revenue-sharing or additional-fee obligation reference;
- geographic or customer-class restrictions when present;
- effective date, expiry/review date and authoritative evidence reference;
- reviewer, approval state and evidence digest.

Do not encode a provider-specific license schema in routing. Normalize commercial constraints into policy-evaluable fields plus immutable evidence references.

### Acceptance direction

A route candidate is rejected before execution when the tenant/task usage would violate commercial rights, even if the model is technically available and passes capability checks.

## 2. Pricing Must Be Versioned and Deployment-Scoped

A single `price_per_token` field cannot represent modern inference economics.

Pricing may vary by:

- deployment/provider;
- region;
- input versus output;
- cache hit/miss;
- context band;
- batch/asynchronous mode;
- service tier;
- inference/reasoning effort;
- time window or promotional schedule;
- reserved/dedicated capacity;
- self-hosted GPU allocation and utilization.

### Development directive

Introduce a versioned `PricingPolicy` or equivalent deployment-scoped cost model referenced by Deployment.

A pricing snapshot used by a RouteDecision must be immutable/replayable for that decision.

The router should predict total task cost from the active pricing policy, but final CostEvents must record actual measured usage and the exact pricing-policy version used for reconciliation.

### Acceptance direction

Given two deployments of the same ModelVersion with different region/cache/service-tier economics, the router can explain why one was selected and the decision can be replayed from stored pricing evidence.

## 3. Model and Deployment Lifecycle Must Be First-Class

Hosted model identifiers are not permanent infrastructure. Providers can move models through preview, stable, deprecated and retired states, reduce quota during deprecation, or nominate replacements.

### Development directive

Keep ModelVersion lifecycle and Deployment lifecycle separate.

Suggested ModelVersion states:

```text
DRAFT -> ACTIVE -> DEPRECATED -> RETIRED
                  \-> REVOKED
```

Suggested Deployment states:

```text
DISCOVERED -> READY -> DEGRADED -> DRAINING -> RETIRED
                     \-> BLOCKED
```

Record lifecycle events with:

- announced-at;
- effective-at;
- replacement ModelVersion/Deployment references when known;
- provider notice/evidence reference;
- quota/rate-limit change when relevant;
- routing eligibility;
- migration state.

### Acceptance direction

Deprecated deployments may continue existing traffic under policy but cannot silently remain the preferred route. Retired, revoked or blocked objects are excluded from new routing. Historical Tasks remain linked to the exact objects used.

## 4. Add a Controlled Migration Workflow

Model lifecycle events imply a control-plane workflow, not only a status flag.

Required migration sequence:

```text
Provider notice / internal decision
  -> lifecycle event
  -> replacement candidate discovery
  -> shadow / regression evaluation
  -> policy and license check
  -> canary traffic
  -> traffic shift
  -> observation window
  -> rollback or completion
  -> old deployment drain / retire
```

### Development directive

Do not implement automatic replacement by model name alias. Migration must be evidence-driven and reversible.

Minimum migration record:

- source and target ModelVersion/Deployment;
- reason;
- evaluation evidence;
- policy/license evidence;
- canary percentage or cohort;
- start/end timestamps;
- success gates;
- rollback target;
- final decision and actor.

## 5. Public Benchmark Is Reference Data, Not Production Truth

Enterprise model selection also depends on trust, supply chain, residency, commercial rights, runtime reliability and deployment freedom.

### Development directive

Keep public benchmark data as one evidence source. Production routing should prefer task-class-specific Execution Evaluation and RouteOutcome evidence.

The router must never use a public aggregate score as a bypass around hard policy, license, residency, lifecycle or capacity constraints.

## 6. Open Weight + Managed Inference Must Be Representable Without Special Cases

The same open-weight ModelVersion may be served by a vendor API, managed inference provider, enterprise private endpoint or self-hosted runtime.

### Development directive

Provider and Deployment remain replaceable operational resources. Open-weight status belongs to ModelVersion evidence; managed-service pricing/health/capacity belongs to Deployment.

No business workflow should distinguish `open` versus `commercial` by hard-coded provider names.

## 7. Router Hard Constraints and Soft Objectives

### Hard constraints

Apply before optimization:

- capability/context fit;
- tenant model/provider allow/deny policy;
- lifecycle/routing eligibility;
- license and commercial-use eligibility;
- residency and region requirements;
- security/provenance approval;
- available safe capacity;
- tool/credential policy where execution requires tools.

### Soft objectives

Optimize only among eligible candidates:

- predicted task success;
- predicted cost per successful task;
- expected latency/queue time;
- historical reliability;
- retry/fallback probability;
- human-intervention probability;
- deployment preference and operational cost.

## 8. Required Schema Direction

The next implementation slices should converge on these relations:

```text
Model
  1 -> N ModelVersion

ModelVersion
  1 -> N Deployment
  1 -> N LicenseEvidence
  1 -> N EvaluationEvidence

Deployment
  1 -> N PricingPolicyVersion
  1 -> N RuntimeSignalWindow
  1 -> N LifecycleEvent

Task
  1 -> N RouteDecision
  1 -> N CostEvent
  1 -> N RouteOutcome

ModelMigration
  source ModelVersion/Deployment
  target ModelVersion/Deployment
  evaluation + policy + rollout evidence
```

## 9. Development Priority Impact

These signals do not justify creating a new architecture layer. They refine R5, R6, R7 and supply-chain governance.

### Immediate

1. finish ModelVersion versus Deployment separation;
2. add deployment lifecycle and routing eligibility;
3. define versioned deployment-scoped PricingPolicy;
4. extend license evidence with machine-evaluable commercial-use constraints;
5. ensure RouteDecision stores pricing, license, lifecycle and evidence versions.

### Next

1. add model/deployment migration workflow and canary/rollback evidence;
2. incorporate dynamic pricing dimensions into predicted task cost;
3. add lifecycle-triggered evaluation and replacement recommendations;
4. use RouteOutcome for production-evidence comparison of replacement candidates.

### Deferred

Do not implement autonomous pricing arbitrage or automatic model migration until replay, rollback, policy governance and evidence quality are proven.

## 10. Engineering Gates

A routing implementation is not complete if:

- ModelVersion contains mutable endpoint health/capacity state;
- one static token-price field is treated as authoritative cost;
- license is represented only as a free-text label;
- deprecated/retired deployments can remain eligible without explicit policy;
- model aliases can silently change the underlying ModelVersion;
- route decisions cannot be replayed against the exact pricing/license/lifecycle evidence used.

A production model upgrade is not complete until it has:

- replacement evaluation evidence;
- license and policy admission;
- controlled traffic shift;
- rollback capability;
- post-shift RouteOutcome observation.

## Relationship to Existing Contracts

This document refines, rather than replaces:

- R5 Deployment Registry;
- R6 Capability / Economics / Runtime-Aware Router;
- R7 Execution Evaluation;
- R8 Route Outcome Feedback Loop;
- evidence-based model supply-chain governance.

The strategic principle remains:

> AI Cloud should optimize governed execution paths, not chase individual model releases.
