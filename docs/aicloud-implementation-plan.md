# aicloud Implementation Plan

## 1. Goal

This document converts the `aicloud` positioning and architecture into executable engineering milestones.

Current product positioning:

```text
Hybrid Private AI Cloud Platform + AI-native Infrastructure Control Plane
```

Current product center:

```text
Governed hybrid model access + policy-aware agent workflows
```

## 2. Roadmap Principles

```text
1. Build a model gateway before building many agents.
2. Build evaluation before trusting any model.
3. Build structured output before workflow execution.
4. Build policy boundaries before infrastructure control.
5. Integrate first, customize later.
6. Private/open-source model support must be first-class.
7. Infrastructure control is the first scenario, not the only product direction.
```

## 3. Milestones

```text
M0 Repository Foundation
M1 Model Core
M2 Mock Gateway MVP
M3 Provider Integration
M4 Agent Workflow MVP
M5 Infrastructure Scenario MVP
M6 Enterprise Governance
M7 Custom Model Experiments
```

## 4. M0 Repository Foundation

Status:

```text
Partially complete
```

Implemented:

```text
README.md
go.mod
.github/workflows/go-test.yml
docs/README.md
```

Acceptance criteria:

```text
- repository has clear product README
- go test ./... is wired into CI
- docs entrypoint exists
```

## 5. M1 Model Core

Status:

```text
Mostly complete
```

Implemented packages:

```text
model/provider
model/schema
model/mock
```

Implemented capabilities:

```text
- ModelProvider interface
- ProviderRequest / ProviderResponse
- structured output schemas
- BasicValidator
- deterministic MockProvider
- MockProvider unit tests
```

First working flow:

```text
MockProvider.GeneratePlan
  ↓
schema.ChangePlan
  ↓
schema.BasicValidator.ValidateChangePlan
```

## 6. M2 Mock Gateway MVP

Status:

```text
Mostly complete
```

Implemented packages:

```text
model/safety
model/gateway
model/eval
model/routing
model/registry
```

Implemented capabilities:

```text
- SafetyGuard request validation
- SafetyGuard response validation
- Gateway.GeneratePlan
- AuditRecord generation
- EvalRunner
- DefaultDevScaleOutCase
- StaticRouter
- MemoryRegistry
```

Current full flow:

```text
MockProvider
  ↓
Gateway.GeneratePlan
  ↓
SafetyGuard
  ↓
BasicValidator
  ↓
AuditRecord
  ↓
EvalRunner
  ↓
Router / Registry
```

Acceptance criteria:

```text
- Gateway.GeneratePlan returns ChangePlan
- unsafe request is blocked
- EvalRunner scores MockProvider
- Router uses ProviderScore
- Registry lists provider capabilities and health
```

## 7. M3 Provider Integration

Status:

```text
Skeleton started
```

Implemented package:

```text
model/openai
```

Current capability:

```text
- OpenAI-compatible Provider skeleton
- public/private provider type support
- Config with Endpoint / EndpointRef / SecretRef
- fake client tests
- fail-closed behavior while structured parser is not implemented
```

Next tasks:

```text
- implement strict JSON/YAML structured parser
- add OpenAI-compatible request body implementation
- add provider config loading
- add retry and timeout policy
- add optional integration tests guarded by env vars
```

## 8. M4 Agent Workflow MVP

Status:

```text
Not started
```

Goal:

```text
Turn validated model output into controlled workflow objects.
```

Planned packages:

```text
agent/proposal
agent/planner
agent/prdraft
```

Planned objects:

```text
ChangeProposal
PRDraft
WorkflowState
RollbackPlanRef
ValidationPlan
```

First flow:

```text
ChangePlan
  ↓
PolicyChecker
  ↓
ChangeProposal
  ↓
PRDraft
```

Acceptance criteria:

```text
- ChangePlan can convert to ChangeProposal
- risk is determined by deterministic policy, not model alone
- PR draft contains intent, risk, rollback, validation checklist
- no direct execution instructions are generated
```

## 9. M5 Infrastructure Scenario MVP

Status:

```text
Not started
```

Goal:

```text
Implement the first high-value scenario: AI-assisted infrastructure change planning.
```

First scenario:

```text
ManagedCluster workers replicas 3 -> 6
```

Planned packages:

```text
infra/api
infra/controller
infra/adapter
policy/risk
policy/approval
```

Planned objects:

```text
ManagedCluster
MachineClass
AgentOperation
RiskPolicy
ApprovalPolicy
```

Acceptance criteria:

```text
- CRD design exists
- fake controller can update status
- model-generated ChangePlan can produce PR-ready manifest change
- PolicyChecker classifies dev 3 -> 6 as Medium
- real execution remains through GitOps / Controller, not model
```

## 10. M6 Enterprise Governance

Status:

```text
Not started
```

Planned capabilities:

```text
ProviderRegistry persistence
ModelRoutingPolicy
ModelEvaluationReport storage
AuditCenter
RBAC
SSO integration
Cost and latency tracking
Data sensitivity policy
Audit export
```

Acceptance criteria:

```text
- admin can register providers
- admin can set routing policy
- each model call is audited
- provider evaluation reports are stored
- sensitive data rules affect routing
```

## 11. M7 Custom Model Experiments

Status:

```text
Not started
```

Candidate custom models:

```text
InfraChangeRiskClassifier
KubernetesYamlRepairModel
PolicyExplanationModel
ValidationReportSummarizer
RunbookGenerationModel
```

Prerequisites:

```text
- synthetic golden dataset exists
- sanitized dataset format exists
- evaluation harness exists
- baseline providers are measured
- safety boundary is stable
```

Acceptance criteria:

```text
- custom model beats baseline on one narrow task
- no safety regression occurs
- model is routed only to its approved task class
```

## 12. Recommended PR Order

Completed or mostly completed:

```text
PR-001 repo skeleton and docs
PR-002 model/provider
PR-003 model/schema + validator
PR-004 model/mock
PR-005 model/safety
PR-006 model/gateway with MockProvider
PR-007 model/eval first case
PR-008 model/routing
PR-009 model/openai skeleton
PR-010 model/registry
```

Next PRs:

```text
PR-011 model structured parser
PR-012 provider config loading
PR-013 agent ChangeProposal
PR-014 policy checker MVP
PR-015 PR draft generator
PR-016 infra scenario design
PR-017 ManagedCluster API skeleton
```

## 13. Immediate Next Steps

Recommended next implementation sequence:

```text
1. Fix any CI failures from current model packages.
2. Update model/README.md to include all packages.
3. Implement structured parser for OpenAI-compatible provider.
4. Add agent ChangeProposal model.
5. Add deterministic PolicyChecker MVP.
6. Add PR draft generator.
```

## 14. Current Done Definition

The current model-layer MVP is done when:

```text
1. go test ./... passes.
2. MockProvider flow passes through Gateway.
3. SafetyGuard blocks unsafe requests and outputs.
4. EvalRunner produces EvalReport.
5. Router can route based on ProviderScore.
6. Registry can register, list, and health-check providers.
7. OpenAI-compatible provider skeleton fails closed until parser is implemented.
```
