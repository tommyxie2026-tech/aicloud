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
```

## 9. v0.1 Skeleton Implementation

The repository now includes a runnable modular-monolith skeleton for the
Developer AI Cloud path:

```text
cmd/api-server       HTTP API entrypoint
cmd/worker           workflow worker placeholder
internal/            control-plane and execution seams
model/               existing model/provider packages
agent/               existing planning/workflow packages
policy/              existing deterministic policy packages
db/migrations/       PostgreSQL persistence contract
deploy/helm/aicloud  minimal Kubernetes deployment
```

Run locally without external services:

```bash
go run ./cmd/api-server
curl http://localhost:8080/healthz
curl http://localhost:8080/api/v1/models
```

The v0.1 API exposes `/healthz`, `/readyz`, `GET/POST /api/v1/models`, and
`GET/POST /api/v1/tasks` plus task lookup by ID. Runtime persistence is
in-memory for fast startup; PostgreSQL and Redis are available through
`docker compose up -d`, and the migration is the contract for the next
persistence adapter.

The implementation intentionally keeps one deployable application and
separates internal packages by domain. Temporal, LiteLLM, OPA, OpenTelemetry,
and stronger sandbox runtimes remain integration seams for the next sprint.
