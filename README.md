# aicloud

`aicloud` is a hybrid private AI cloud platform.

It connects public general-purpose large models, enterprise private large models, self-hosted open-source models, local small models, and future domain-specific custom models through a governed model gateway and policy-aware agent workflow.

## 1. Product Positioning

```text
Hybrid Private AI Cloud Platform + AI-native Infrastructure Control Plane
```

Core product center:

```text
Governed hybrid model access + policy-aware agent workflows
```

`aicloud` is designed to support:

```text
- external public LLM providers
- internal private LLM providers
- self-hosted open-source model providers
- local small model providers
- domain-specific custom model providers
- model routing
- model evaluation
- structured model output
- safety validation
- audit logging
- agent workflow
- infrastructure control scenarios
```

## 2. What aicloud Is

`aicloud` is:

```text
- a hybrid model gateway
- a private AI platform layer
- a provider abstraction layer
- a model routing and evaluation platform
- a safety and audit boundary for agents
- an AI-native infrastructure control plane foundation
```

## 3. What aicloud Is Not Initially

`aicloud` is not initially:

```text
- a foundation model training platform
- a simple API proxy only
- a chatbot only
- an uncontrolled autonomous agent
- a direct infrastructure executor
```

Training and fine-tuning may come later, but the first priority is safe hybrid model access and governed agent workflows.

## 4. Architecture Layers

```text
L1 Model Connectivity Layer
L2 Model Governance Layer
L3 Agent Runtime Layer
L4 Infrastructure Control Plane Layer
L5 Enterprise Integration Layer
```

## 5. Initial Repository Structure

```text
model/        hybrid model gateway and governance
agent/        planner and workflow runtime
policy/       deterministic policy and approval checks
infra/        Kubernetes CRDs, controllers, adapters
api/          shared API types
integrations/ GitHub, GitLab, SSO, observability, knowledge connectors
eval/         model evaluation cases and reports
datasets/     synthetic and sanitized model datasets
docs/         product and technical documents
hack/         development scripts
```

## 6. First Engineering Milestone

The first milestone is to make the model core compile:

```text
model/provider
model/schema
model/mock
```

First executable path:

```text
MockProvider
  ↓
GeneratePlan
  ↓
ChangePlan
  ↓
BasicValidator
```

First scenario:

```text
scale dev-gpu-cluster gpu-workers from 3 to 6
```

Expected structured output:

```text
ChangePlan
- target: ManagedCluster/dev-gpu-cluster
- field: spec.workers[name=gpu-workers].replicas
- from: 3
- to: 6
- riskHint: Medium
- rollback: set replicas back to 3
```

## 7. Safety Principle

```text
Models propose.
Policy decides.
Humans approve when required.
Controllers execute.
```

Model output is untrusted until it passes:

```text
Schema validation
Safety validation
Policy check
Human review when required
```

## 8. Immediate Backlog

```text
AICLOUD-001 Initialize repository structure
AICLOUD-003 Add ModelProvider interface
AICLOUD-004 Add structured output schemas
AICLOUD-005 Add BasicValidator
AICLOUD-006 Add deterministic MockProvider
AICLOUD-008 Add ModelGateway with MockProvider path
AICLOUD-009 Add EvalRunner and first golden case
AICLOUD-010 Add operational Model Registry fields
AICLOUD-011 Add model, inference-effort, and service-tier routing schemas
AICLOUD-012 Add immutable Task CostEvent ledger
AICLOUD-013 Add commercial and private/self-hosted provider routes
AICLOUD-014 Add Tool Gateway, OPA, credential, and Sandbox foundations
AICLOUD-015 Add Agent attack and privilege-boundary evaluation suite
AICLOUD-016 Add health, capacity, quota, and fallback routing
AICLOUD-017 Add specialist code, security, and document routing
AICLOUD-018 Add reproducible evaluation evidence records
AICLOUD-019 Add model license and provenance admission
```

## 9. Current P0 Roadmap

The current implementation priority is not to connect the largest possible number of models. It is to execute complete tasks through an explainable, secure, cost-effective, recoverable, and provider-independent path.

```text
Operational Model Registry
-> deterministic / efficient / specialist / flagship routing
-> model + inference effort + service tier selection
-> hybrid commercial and internal-model deployment
-> task-level cost ledger
-> Tool Gateway + Policy + short-lived credentials + Sandbox
-> Agent attack and privilege-boundary tests
-> health, quota, capacity, latency, circuit breaker, and fallback
-> full Trace and reproducible evaluation evidence
-> license and provenance production admission
```

The ten current P0 capabilities are:

1. operational Model Registry;
2. joint routing of model, inference effort, and service tier;
3. commercial API and internal-model hybrid deployment;
4. total cost per successful task;
5. Tool Gateway, Policy Engine, and Sandbox before broad Agent autonomy;
6. model-license and provenance evidence admission;
7. Agent-level attack and privilege-escalation tests;
8. health-, quota-, capacity-, latency-, and failure-aware routing;
9. specialist routing for security, code, document, and future domain tasks;
10. end-to-end Trace and reproducible evaluation configuration.

Detailed implementation requirements, current progress, module mapping, test gates, and staged delivery are documented in:

- [`docs/roadmap/2026-07-27-model-routing-security-and-evaluation-priorities.md`](docs/roadmap/2026-07-27-model-routing-security-and-evaluation-priorities.md)
- [`docs/roadmap/AI-Cloud-Implementation-Plan.md`](docs/roadmap/AI-Cloud-Implementation-Plan.md)
- [`docs/roadmap/v0.1-engineering-milestones.md`](docs/roadmap/v0.1-engineering-milestones.md)
- [`docs/roadmap/AI-Cloud-Module-Implementation-Plan-and-Status.md`](docs/roadmap/AI-Cloud-Module-Implementation-Plan-and-Status.md)

The runnable v0.1 skeleton remains in Draft PR #1 until reviewed and merged. Code is counted as completed only after its acceptance criteria pass on `main`.
