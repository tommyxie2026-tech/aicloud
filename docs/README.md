# aicloud Docs

This directory contains product and technical design documents for `aicloud`.

## Product Positioning

`aicloud` is a hybrid private AI cloud platform.

It connects public general-purpose models, enterprise private models, self-hosted open-source models, local small models, and future domain-specific models through a governed model gateway and policy-aware agent workflow.

## Current Core Positioning

```text
Hybrid Private AI Cloud Platform + AI-native Infrastructure Control Plane
```

Product center:

```text
Governed hybrid model access + policy-aware agent workflows
```

## Documents

```text
aicloud-positioning.md          product positioning and boundaries
aicloud-product-architecture.md five-layer product architecture
aicloud-implementation-plan.md  executable milestone and backlog plan
```

## Current Engineering Status

The current MVP focuses on the model layer:

```text
model/provider
model/schema
model/mock
model/safety
model/gateway
model/eval
model/routing
model/openai
model/registry
```

The first working flow is:

```text
MockProvider
  ↓
Gateway.GeneratePlan
  ↓
SafetyGuard
  ↓
BasicValidator
  ↓
EvalRunner
  ↓
Router / Registry
```

## Design Principle

```text
Models propose.
Policy decides.
Humans approve when required.
Controllers execute.
```
