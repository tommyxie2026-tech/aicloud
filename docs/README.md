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

## Strategic Outlook

- [AI Evolution Outlook: From Agents to Governed Continual Learning](roadmap/2026-08-28-continuous-learning-and-organizational-memory.md)
- [AI 演进判断：从智能体走向受治理的持续学习](roadmap/2026-08-28-continuous-learning-and-organizational-memory.zh-CN.md)
- [aicloud 可扩展 AI 教育平台：产品架构与 P0–P4 实施路线](roadmap/2026-11-27-ai-education-platform-architecture-and-p0-p4.zh-CN.md) — confirmed direction, reusable subject-plugin contracts, stage gates and acceptance criteria.
- [钢琴智能陪练：开源视觉与声音识别训练平台技术选型及实验路线图](roadmap/2026-11-27-piano-ai-tutoring-platform.zh-CN.md) — piano-specific AMT/OMR models, data collection and experiments.

## Vertical AI Workloads

- [Piano AI research workspace](../research/piano-ai/README.md) — first vertical workload and reproducible AMT/OMR baseline; see the platform roadmap for P0–P4 acceptance.

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
