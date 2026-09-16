# R1: Task Execution Contract Design

**English** | [简体中文](task-execution-contract.zh-CN.md)

> Status: R1 design draft  
> Date: 2026-09-16  
> Upstream: [AI Execution Control Plane](ai-execution-control-plane.md)

## 1. R1 objective

R1 establishes a unified execution contract before building a complex Agent platform. Existing inference, agents, tools and future human tasks become Tasks with consistent planning, execution, observability and audit semantics.

```text
TaskDefinition → ExecutionPlan → ExecutionRun
```

`TaskDefinition` describes desired outcome; `ExecutionPlan` describes the intended method; `ExecutionRun` records what actually happened, consumed resources and outcome.

## 2. Design principles

Task is first-class while Models/Agents are execution capabilities. Definitions express intent without provider binding. Plans are revisable; Definitions remain stable. Runs are factual records. Policy owns final authorization. Every execution emits cost/latency/resource/outcome telemetry. Contracts support synchronous inference, long-running agents, DAGs and human approval. API comes first; CRDs are Kubernetes-native mappings rather than the only interface.

## 3. TaskDefinition

Suggested APIs: `POST /v1/tasks`, `GET /v1/tasks/{task_id}`, `POST /v1/tasks/{task_id}:plan`, `POST /v1/tasks/{task_id}:run`, `POST /v1/tasks/{task_id}:cancel`.

Example:

```yaml
apiVersion: aicloud.io/v1alpha1
kind: TaskDefinition
metadata:
  id: task-01J...
  tenant: tenant-a
  project: ops
spec:
  goal: "Analyze a StarRocks CN write stall and produce evidence-backed root-cause candidates"
  taskType: analysis
  inputs:
    - type: file
      ref: artifact://logs/fe-audit.log
  constraints:
    maxCostUsd: 5.0
    timeoutSeconds: 1800
    dataClass: internal
    regionPolicy: apac-only
    humanApproval: on-high-risk-action
  successCriteria:
    - type: output-schema-valid
    - type: evidence-cited
  preferences:
    quality: high
    latency: normal
    cost: balanced
```

Provider names should not appear unless explicitly supplied as hard constraints.

## 4. ExecutionPlan

Planner + Router create a plan; Policy authorizes it. Steps should prefer capabilities over hard-coded model names.

```yaml
kind: ExecutionPlan
metadata:
  id: plan-01J...
  taskId: task-01J...
spec:
  revision: 3
  steps:
    - id: inspect-logs
      type: tool
      tool: file.search
    - id: analyze
      type: model
      capability: deep-reasoning
      dependsOn: [inspect-logs]
    - id: validate
      type: evaluator
      profile: evidence-quality
      dependsOn: [analyze]
  estimated:
    costUsd: 1.40
    completionSeconds: 240
  policyDecision:
    status: allowed
    decisionId: pd-01J...
```

## 5. ExecutionRun

Run is an aggregate projection over an immutable event stream. It records resolved provider/model/runtime, usage, cost, latency, artifacts, outcome and human intervention. History is append-only so runs can be replayed, audited, debugged and later used for governed procedural memory.

## 6. State machine

```text
DRAFT → ACCEPTED → PLANNED → RUNNING → SUCCEEDED
                         │          ├→ FAILED
                         │          ├→ CANCELLED
                         │          └→ SUSPENDED → RUNNING
                         └→ REJECTED_BY_POLICY
```

Plan revision and run attempts remain separate: replanning changes `plan.revision`; infrastructure retries create a new attempt.

## 7. Event model

Minimum events: `task.created`, `plan.generated`, `policy.allowed`, `policy.denied`, `run.started`, `step.started`, `step.completed`, `step.failed`, `run.suspended`, `human.approval.requested`, `human.approval.received`, `run.completed`, `run.failed`, `run.cancelled`.

Every event includes task/plan/run/trace IDs, tenant, timestamp and actor.

## 8. Telemetry contract

Extend OpenTelemetry with `ai.task.id`, `ai.plan.id`, `ai.run.id`, `ai.step.id`, `ai.model.profile`, `ai.provider`, `ai.tool.id`, `ai.policy.decision_id`, `ai.resource.profile`, `ai.cost.usd`, token/cache usage, human interventions and outcome success.

This lets future routing learn from real task outcomes rather than static benchmarks alone.

## 9. Policy insertion points

Policy checks candidate plans for data/model/region/budget constraints and resolved plans for tool/credential/action permissions. High-risk actions can require human approval. R1 defines a Policy Decision Contract without requiring a complete policy language.

## 10. Runtime Adapter

Define a neutral SPI:

```text
ExecutionAdapter
├── ModelAPIAdapter
├── VLLMAdapter
├── SGLangAdapter
├── AgentAdapter
├── ToolAdapter
└── HumanAdapter
```

Common semantics: `prepare`, `execute`, `suspend`, `resume`, `cancel`, `collect_usage`. Agent frameworks and Codex-like harnesses remain adapters rather than core dependencies.

## 11. PoC: operations incident analysis

Use an objectively evaluable production-incident analysis task: logs/observations → TaskDefinition → Planner → Router → file/log tools → deep reasoning → evidence evaluator → ExecutionRun → structured RCA report.

Acceptance: switch the same Task across at least two commercial providers and one local model; trace model/tool/cost/latency; retry without losing history; enforce budget; pass evidence evaluation; replay by task ID.

## 12. R1 non-goals

No complex low-code workflow UI, autonomous long-term memory learning, complete ABAC/Rego policy system, full heterogeneous accelerator scheduling, automated training, or cross-organization agent marketplace.

## 13. Definition of Done

```text
Task API
+ versioned Plan
+ executable/cancellable/retryable Run
+ provider-neutral Adapter
+ Policy Decision insertion points
+ OTel + cost telemetry
+ append-only execution events
+ one real end-to-end PoC
```

Then proceed to R2: Model Registry 2.0 + Intelligent Router 2.0.