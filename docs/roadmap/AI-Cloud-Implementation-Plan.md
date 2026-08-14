# AI Cloud Implementation Plan

## 1. Strategic Direction

AI Cloud should not compete with foundation model providers or remain a simple Token Gateway.

The product differentiation is:

```text
Safe AI execution
+
Enterprise governance
+
Agent orchestration
+
Model independence
+
Hybrid deployment
+
Continuous evaluation
```

The model market is evolving from isolated model competition toward system competition across models, agents, tools, governance, infrastructure, and industry workflows. The roadmap therefore treats models as replaceable intelligent resources and treats task execution, reliability, governance, and business outcomes as the durable platform layer.

## 2. Market-Driven Technical Priorities

The following priorities apply across all implementation phases.

### P0-1 Unified Model Protocol and Model Registry

AI Cloud must avoid binding its API, data model, task runtime, or evaluation system to one provider.

Required capabilities:

- provider-neutral request and response protocol;
- provider adapter interface;
- capability metadata for reasoning, coding, multimodal, tool use, context length, and structured output;
- versioned model registration;
- commercial API, enterprise private endpoint, self-hosted open-source model, and local model support;
- model lifecycle states: draft, active, degraded, deprecated, and retired.

The Model Registry is the source of truth for immutable model identity, capability, evaluation, license, provenance and approval evidence. Deployment-specific health, capacity, region, endpoint and runtime pricing belong to the Deployment Registry introduced in R5.

### P0-2 Task-Level Cost Accounting

Token usage is an input metric, not the final cost unit.

AI Cloud must calculate:

```text
Task Cost
=
Model Input and Output Cost
+ Tool Cost
+ Workflow Runtime Cost
+ Sandbox Compute Cost
+ Storage and Network Cost
+ Retry and Failure Cost
+ Human Review Cost
```

Required dimensions:

- tenant;
- project;
- user or service account;
- agent;
- task;
- workflow run;
- model and deployment;
- tool;
- sandbox execution.

Primary outcome metric:

```text
Cost per Successful Task
```

### P0-3 Agent Observability and Continuous Evaluation

Every task must produce a complete execution trace covering planning, model calls, deployment selection, tool calls, policy decisions, sandbox execution, retries, approvals, validation, cost, route outcomes, and final result.

Evaluation must include:

- offline benchmark datasets;
- production trace sampling;
- regression tests before model, prompt, harness, workflow or route-policy upgrades;
- quality, cost, latency, safety, stability, fallback and human-intervention metrics;
- model routing decisions based on enterprise task evidence rather than public benchmarks alone.

Evaluation is progressively extended from model-level scoring to execution-configuration evaluation under R7.

### P0-4 Hybrid Model Deployment

AI Cloud must support a mixed model estate:

- public commercial APIs;
- dedicated enterprise APIs;
- private cloud endpoints;
- self-hosted open-source models;
- local small models;
- edge deployments where policy allows;
- future domain-specific models.

Control-plane APIs, policy, task execution, observability, and evaluation must work consistently across all deployment modes. A model version may have multiple deployments and the router selects a deployment independently from immutable model identity.

### P0-5 Secure Tool and Enterprise-System Access

Models and agents must never directly access enterprise systems.

The mandatory execution path is:

```text
Agent
  |
Tool Gateway
  |
Policy Engine
  |
Approval when required
  |
Credential Broker
  |
Sandbox or Approved Enterprise Resource
```

Required controls:

- short-lived credentials;
- tool-level permission and risk metadata;
- deterministic policy checks before invocation;
- input and output filtering;
- human approval for high-risk actions;
- complete audit records.

### P1-1 Open-Model License and Supply-Chain Governance

The platform must not treat a model-card license label as sufficient evidence for commercial use.

The Model Registry must track:

- weight license;
- upstream base-model license;
- dataset and fine-tuning provenance where available;
- commercial-use restrictions;
- redistribution and hosted-service restrictions;
- attribution and notice requirements;
- model artifact digest and signature;
- security scan and approval status.

A model may enter production routing only after the required license and supply-chain checks pass.

### P1-2 Capacity-Aware Routing and Graceful Degradation

The router must account for more than quality and unit price.

Routing inputs must include:

- deployment health;
- quota and rate-limit availability;
- regional availability;
- estimated queue time;
- context and capability fit;
- cost budget;
- tenant policy;
- execution-evaluation evidence;
- data-residency restrictions;
- inference effort and service tier when supported.

Required reliability controls:

- health checking;
- timeout and bounded retry;
- circuit breaking;
- deployment and model fallback chains;
- request admission control;
- budget and capacity reservation;
- degraded-mode behavior;
- no cross-tenant cache leakage.

## 3. Phase 0: Foundation Validation

Goal: validate model abstraction, structured output, safety workflow, and the minimum model-governance metadata.

Components:

- ModelProvider interface;
- provider-neutral model schema;
- deterministic Mock Provider;
- structured output;
- basic validation;
- evaluation runner;
- initial Model Registry fields for provider, capability, deployment mode, price, license, and risk;
- task and trace identifiers propagated through the complete path.

Flow:

```text
MockProvider
    |
GeneratePlan
    |
ChangePlan
    |
BasicValidator
    |
Evaluation Result
```

Exit criteria:

- one provider-neutral request can execute through at least two adapters;
- each execution records model version, deployment identity where applicable, trace ID, token usage, estimated cost, and validation result;
- unsupported provider-specific fields do not leak into the platform domain model.

## 4. Phase 1: AI Cloud MVP

Goal: build the first usable AI Control Plane and a reliable hybrid model execution path.

Deliver:

- unified model protocol;
- Model Gateway and provider adapters;
- Model Registry;
- Deployment Registry foundation;
- commercial API and self-hosted model integration paths;
- capability-aware model and deployment routing;
- health checks, circuit breaker, timeout, retry, and fallback;
- task-level usage and cost ledger;
- Policy Engine;
- Agent workflow engine;
- basic sandbox;
- Tool Gateway foundation;
- audit logging and end-to-end trace;
- basic evaluation regression suite.

Recommended technology:

- LiteLLM as an initial Gateway data-plane component, behind an AI Cloud-owned protocol boundary;
- PostgreSQL metadata and task ledger;
- Redis cache and rate-limit coordination;
- Temporal workflow;
- OPA policy engine;
- Kubernetes execution runtime;
- OpenTelemetry tracing and metrics.

Exit criteria:

- at least one commercial provider and one self-hosted or private provider are available behind the same API;
- one model version can be represented independently from one or more deployment endpoints;
- a provider outage or quota exhaustion triggers a policy-compliant fallback;
- every task exposes total cost, selected model and deployment, selection reason, retry history, and final status;
- task traces connect API request, workflow, route decision, model call, tool call, and sandbox execution.

## 5. Phase 2: Enterprise AI Platform

Goal: turn the MVP into an enterprise-governed AI execution platform.

Add:

- multi-tenant architecture;
- RBAC and workload identity;
- Credential Vault and short-lived credentials;
- production MCP and enterprise Tool Gateway;
- execution-evaluation platform;
- AI FinOps budgets, showback, and chargeback;
- model license and supply-chain governance;
- model artifact signature and approval workflow;
- production capacity management and admission control;
- regional routing and data-residency policy;
- Agent Harness Registry;
- route-outcome feedback and controlled route-policy improvement;
- agent package and marketplace foundation.

Exit criteria:

- tenant, project, agent, task, model, deployment, tool, and sandbox costs are independently reportable;
- unapproved or license-incompatible models cannot enter production routing;
- high-risk tool actions require policy approval and optional human approval;
- model, prompt, harness or routing-policy upgrades must pass regression gates before production rollout;
- the platform can operate in degraded mode when one provider, deployment or region is unavailable.

## 6. Phase 3: AI Operating System

Long-term goal:

```text
AI Cloud OS

= Model Platform
+ Agent Platform
+ Tool Platform
+ Security Platform
+ Governance Platform
+ Intelligent Infrastructure
+ Industry Workflow Platform
```

Capabilities:

- autonomous agent lifecycle;
- enterprise tool ecosystem;
- model supply-chain management;
- continuous online and offline execution evaluation;
- intelligent workload scheduling;
- multi-cluster and multi-region execution;
- capacity-aware and value-aware model/deployment routing;
- evidence-driven route-policy improvement;
- domain-specific agent and model packages;
- industry workflow templates;
- policy-controlled human and agent collaboration.

At this stage, competitive advantage is measured less by access to one model and more by the quality, reliability, safety, cost, and repeatability of complete business workflows.

## 7. Cross-Phase Engineering Workstreams

### 7.1 Model Independence

No business workflow may depend directly on a provider SDK. Provider SDKs remain inside adapters.

### 7.2 Model and Deployment Independence

Immutable model identity, provider integration and runtime deployment are separate concerns. A task route may select model, deployment, inference effort and service tier independently under policy constraints.

### 7.3 Task Economics

All platform components must emit cost events linked to tenant, project, task, workflow, route decision and trace IDs.

### 7.4 Observability and Execution Evaluation

New model, deployment, prompt, harness, workflow, tool and route-policy versions require traceability and regression comparison.

### 7.5 Security and Trust Boundaries

Models propose. Policies decide. Humans approve when required. Controllers and sandboxed workers execute.

### 7.6 Reliability and Capacity

Each external or private model deployment must publish health, quota, capacity, latency, queue and error signals to the router with explicit signal freshness.

### 7.7 License and Supply Chain

Every model artifact or external endpoint must have an owner, source, version, license status, risk status, and approval record.

### 7.8 Route Outcome Feedback

Every completed route attempt should produce evidence that can be joined to task result, cost, latency, retries, fallback, validation, safety and human-intervention outcomes. Adaptive routing may consume this evidence only through versioned, reversible and policy-governed mechanisms.

## 8. Priority Order

### Immediate P0

1. Unified model protocol and adapter contract.
2. Model Registry operational metadata.
3. Deployment Registry foundation and model/deployment separation.
4. Hybrid commercial and private model access.
5. Task-level cost and trace ledger.
6. Health-aware routing, circuit breaking, and fallback.
7. Tool Gateway and Policy Engine boundary.
8. Agent trace and minimum regression evaluation.

### Next P1

1. Joint model/deployment/effort/service-tier routing.
2. Execution Evaluation and RouteOutcome evidence.
3. Agent Harness Registry.
4. Open-model license and supply-chain governance.
5. Capacity admission and regional routing.
6. Enterprise evaluation datasets and release gates.
7. Multi-tenant budgets and cost governance.
8. Workload identity and short-lived credentials.

### Later P2

1. Agent and tool marketplace.
2. Controlled evidence-driven routing adaptation.
3. Intelligent value-aware scheduling.
4. Multi-cluster optimization.
5. Domain-specific model and workflow packages.
6. Automated business-value and ROI optimization.

## 9. Architecture Decisions

The following ADRs guide implementation:

- ADR-001 Architecture Pattern;
- ADR-002 Workflow Engine;
- ADR-003 Sandbox Isolation;
- ADR-004 Model Protocol;
- ADR-005 Policy Engine;
- ADR-006 Agent Marketplace;
- ADR-007 FinOps;
- ADR-008 Observability;
- ADR-009 MCP Tool Gateway;
- ADR-010 Multi Tenant;
- ADR-011 Identity and Credential Vault;
- ADR-012 Model Marketplace.

Future ADR work should focus on implementation decisions that directly support the roadmap, especially model/deployment separation, joint routing and fallback, execution-evaluation identity, route-outcome evidence, harness versioning, model supply-chain governance, task cost accounting, and hybrid deployment boundaries.

## 10. R5-R10 Next-Stage Contract References

R5-R10 extend the current roadmap without replacing or renumbering v0.1 Engineering Milestones 1-9. They define the next architectural contracts to freeze before implementation expands.

### R5 Deployment Registry

Separate immutable model identity from runtime deployment identity. A Deployment references one ModelVersion and owns endpoint, deployment mode, region, runtime, quantization, cost model, health, capacity, quota, queue, lifecycle, ownership and routing-eligibility metadata.

### R6 Capability / Economics / Runtime-Aware Router

Route selection becomes joint selection across model, deployment, inference effort and service tier. Hard policy constraints are applied before soft optimization over expected task success, total task cost, latency, capacity and reliability.

### R7 Execution Evaluation

Evaluation identity expands from model-only scoring to the exact execution configuration: model, deployment, prompt, harness, workflow, tools, permissions, policy, sandbox/runtime, budgets, retry configuration, dataset and validators.

### R8 Route Outcome Feedback Loop

Persist a RouteOutcome linked to Task, RouteDecision, Trace and Evaluation Evidence. It records success, fallback, retries, final cost, latency, quality, human intervention and safety/policy outcomes. Automatic production-policy mutation remains deferred until replay, rollback and governance controls exist.

### R9 Agent Harness Registry

Version prompt/context strategy, memory/state strategy, tool set, control flow, retry/recovery, verification, permission profile and tracing configuration as a first-class Harness definition. Model-harness pairings become reproducible evaluation targets.

### R10 Tool Execution Governance

Keep Tool Gateway as the mandatory enterprise execution choke point regardless of MCP or other tool protocols. Enforce deterministic policy, task-scoped credentials, approval, sandbox/network boundaries, audit evidence, timeout/idempotency and rollback/compensation metadata where supported.

Detailed rationale, architecture shape, acceptance direction and external evidence are maintained in:

- `docs/roadmap/2026-08-11-industry-signals-and-architecture-implications.md`
- `docs/roadmap/2026-08-11-industry-signals-and-architecture-implications.zh-CN.md`

The canonical next-stage execution chain is:

```text
Task
  -> Capability Requirements
  -> Model
  -> Deployment
  -> Inference Effort / Service Tier
  -> Harness / Agent Runtime
  -> Governed Tool Execution
  -> Trace + Evaluation + CostEvent + RouteOutcome
  -> Policy-bounded Routing Feedback
```
