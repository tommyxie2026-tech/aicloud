# aicloud Product Architecture

## 1. Product Definition

`aicloud` is a hybrid private AI cloud platform.

It connects:

```text
- public general-purpose large models
- enterprise private large models
- self-hosted open-source models
- local small models
- future domain-specific custom models
```

and exposes them through:

```text
- governed model gateway
- model routing
- model evaluation
- structured output
- safety boundary
- audit
- policy-aware agent workflows
- Kubernetes-based infrastructure control scenarios
```

## 2. North-star Architecture

```text
Users / Apps / Agents
  ↓
AIGateway
  ↓
ModelRouter
  ├── Public LLM Provider
  ├── Private LLM Provider
  ├── Self-hosted Open Model Provider
  ├── Local Small Model Provider
  └── Domain Model Provider
  ↓
Model Governance
  ├── Schema Validator
  ├── Safety Guard
  ├── Evaluation Harness
  ├── Provider Registry
  ├── Audit Center
  └── Cost / Latency Tracker
  ↓
Agent Runtime
  ├── Planner
  ├── Context Manager
  ├── Tool Boundary
  └── Workflow State
  ↓
Policy / Approval / GitOps
  ↓
Infrastructure Control Plane / Enterprise APIs
```

## 3. Five-layer Architecture

```text
L1 Model Connectivity Layer
L2 Model Governance Layer
L3 Agent Runtime Layer
L4 Infrastructure Control Plane Layer
L5 Enterprise Integration Layer
```

## 4. L1: Model Connectivity Layer

Purpose:

```text
Connect different kinds of model providers behind a unified interface.
```

Main components:

```text
ProviderRegistry
ProviderAdapter
OpenAICompatibleProvider
PrivateProvider
LocalProvider
OpenModelProvider
CustomDomainProvider
MockProvider
```

Current implemented packages:

```text
model/provider
model/mock
model/openai
model/registry
```

## 5. L2: Model Governance Layer

Purpose:

```text
Ensure model usage is safe, evaluated, auditable, and policy-aware.
```

Main components:

```text
ModelRouter
SchemaValidator
SafetyGuard
ModelEvaluator
AuditRecord
ProviderScore
```

Current implemented packages:

```text
model/schema
model/safety
model/gateway
model/eval
model/routing
```

Core rule:

```text
Model output is untrusted until validated.
```

Every model output must pass:

```text
Schema validation
Safety validation
Policy check
Human review when required
```

## 6. L3: Agent Runtime Layer

Purpose:

```text
Turn user intent into controlled workflow proposals.
```

Future components:

```text
AgentPlanner
ContextManager
ToolBoundary
WorkflowEngine
AgentOperation
ChangeProposal
ValidationReport
RollbackPlan
```

Boundary:

```text
Agent runtime may propose.
It must not directly execute high-risk actions.
```

## 7. L4: Infrastructure Control Plane Layer

Purpose:

```text
Provide the first concrete enterprise scenario for aicloud.
```

Future components:

```text
ManagedCluster CRD
MachineClass CRD
AgentOperation CRD
RiskPolicy CRD
ApprovalPolicy CRD
InfraController
ClusterAPIAdapter
PolicyChecker
GitOps integration
```

First scenario:

```text
Scale dev-gpu-cluster gpu-workers from 3 to 6.
```

Control flow:

```text
User intent
  ↓
Model Gateway generates ChangePlan
  ↓
Policy Checker classifies risk
  ↓
GitHub PR is created
  ↓
GitOps syncs manifest
  ↓
Controller reconciles desired state
```

## 8. L5: Enterprise Integration Layer

Purpose:

```text
Integrate aicloud with enterprise systems.
```

Future components:

```text
GitHub / GitLab integration
SSO / RBAC
Audit Export
Knowledge Base Connector
Observability Connector
Ticket / ITSM Connector
Compliance Report Export
```

## 9. Current MVP Model Flow

The current MVP focuses on the model governance core:

```text
MockProvider
  ↓
Gateway.GeneratePlan
  ↓
SafetyGuard.ValidateRequest
  ↓
Provider.Generate
  ↓
SafetyGuard.ValidateResponse
  ↓
BasicValidator.ValidateChangePlan
  ↓
AuditRecord
  ↓
ChangePlan
```

Evaluation flow:

```text
DefaultDevScaleOutCase
  ↓
MockProvider.Generate
  ↓
BasicValidator
  ↓
ScoreBreakdown
  ↓
EvalReport
  ↓
EvalRecommendation
```

Routing flow:

```text
RouteRequest
  ↓
StaticRouter
  ↓
ProviderScore / Risk / Environment / DataSensitivity
  ↓
RouteDecision
```

## 10. Repository Structure

Current and planned structure:

```text
aicloud/
├── model/
│   ├── provider/
│   ├── schema/
│   ├── mock/
│   ├── safety/
│   ├── gateway/
│   ├── eval/
│   ├── routing/
│   ├── openai/
│   └── registry/
├── agent/
├── policy/
├── infra/
├── api/
├── integrations/
├── eval/
├── datasets/
├── docs/
└── hack/
```

## 11. Product Modules

```text
AIGateway        task-level model API
ModelRouter      task/risk/data-sensitive provider routing
ProviderRegistry provider registration and health
SchemaValidator  structured output validation
SafetyGuard      request/response safety boundary
ModelEvaluator   provider scoring and recommendation
AgentPlanner     future workflow proposal layer
PolicyChecker    deterministic risk and approval engine
InfraController  future infrastructure desired-state controller
AuditCenter      future audit storage and export
```

## 12. Current Conclusion

`aicloud` should be built as a hybrid private AI cloud platform first, with infrastructure control as the first high-value scenario.

The correct product center is:

```text
Governed hybrid model access + policy-aware agent workflows
```
