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
8. Kubernetes-style infrastructure APIs should be designed before real controllers.
9. Fake controllers and adapters should prove semantics before real backend integration.
10. GitOps patch planning should be separated from live execution.
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
Mostly complete
```

Implemented:

```text
README.md
go.mod
.github/workflows/go-test.yml
docs/README.md
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

Current flow:

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

## 7. M3 Provider Integration

Status:

```text
In progress
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
- strict JSON structured parser
- parser rejects markdown fenced output
- parser rejects unknown fields
- parser rejects empty output
- parser rejects unsupported schemas
- parser rejects trailing JSON content
- parser can parse ChangePlan / RollbackPlan / ValidationReport / RiskExplanation and other schema kinds
```

Remaining tasks:

```text
- add provider config loading
- add OpenAI-compatible HTTP request body implementation
- add retry and timeout policy
- add optional integration tests guarded by env vars
- add private/self-hosted runtime adapters or configs
```

## 8. M4 Agent Workflow MVP

Status:

```text
MVP skeleton complete
```

Implemented packages:

```text
agent/proposal
agent/workflow
agent/prdraft
agent/pipeline
policy/checker
```

Implemented objects:

```text
ChangeProposal
ProposalChange
PolicyResult
RollbackProposal
ValidationPlan
Draft
DraftPipeline
```

Implemented flow:

```text
Gateway.GeneratePlan
  ↓
ChangePlan
  ↓
proposal.FromChangePlan
  ↓
PolicyChecker.Evaluate
  ↓
ChangeProposal.ApplyPolicyResult
  ↓
PRDraftGenerator.Generate
  ↓
PR Draft
```

Implemented capabilities:

```text
- ChangePlan can convert to ChangeProposal
- model riskHint is preserved only as a hint
- deterministic PolicyChecker decides risk and approval
- PR draft requires policy evaluation
- PR draft contains intent, target, changes, risk, approval, rollback and validation checklist
- pipeline composes gateway + workflow planner + PR draft generator
```

## 9. M5 Infrastructure Scenario MVP

Status:

```text
Design and fake implementation in progress
```

Goal:

```text
Implement the first high-value scenario: AI-assisted infrastructure change planning using Kubernetes-style APIs.
```

First scenario:

```text
ManagedCluster workers replicas 3 -> 6
```

Implemented docs:

```text
docs/infra-control-plane-scenario.md
docs/managedcluster-api-design.md
infra/README.md
infra/controller/README.md
infra/adapter/README.md
integrations/gitops/README.md
```

Implemented packages:

```text
infra/api
infra/controller
infra/adapter
integrations/gitops
```

Implemented objects:

```text
ManagedCluster
ManagedClusterSpec
ManagedClusterStatus
WorkerGroupSpec
MachineClass
MachineClassSpec
Condition
ClusterAdapter
ObservedClusterState
FakeClusterAdapter
FakeController
FakeStateStore
ManifestPatchPlan
ManifestFieldChange
PRMetadata
PatchPlanner
DryRunManifestWriter
WriteResult
BranchPlan
CommitPlan
FileChangePlan
PullRequestPlan
```

Implemented capabilities:

```text
- Kubernetes-style TypeMeta / ObjectMeta
- Spec / Status separation
- ManagedCluster static validation
- MachineClass static validation
- worker group uniqueness validation
- replicas >= 0 validation
- MachineClass GPU validation
- deterministic fake controller
- adapter boundary with ClusterAdapter
- FakeClusterAdapter with in-memory observed state
- normalized AdapterError codes
- Ready / Reconciling / Degraded conditions
- observedGeneration updates when ready
- ChangeProposal -> ManifestPatchPlan
- GitOps field allowlist and blocked field checks
- rollback inverse patch generation
- PR-ready metadata generation
- ManagedCluster object-level patch 3 -> 6
- DryRunManifestWriter returns updated object and summary
- dry-run BranchPlan / CommitPlan / PullRequestPlan generation
```

Current infrastructure flow:

```text
ManagedCluster.spec
  ↓
FakeController.Reconcile
  ↓
ClusterAdapter.ApplyDesiredState
  ↓
ClusterAdapter.Observe
  ↓
ManagedCluster.status
```

Current GitOps planning flow:

```text
Evaluated ChangeProposal
  ↓
PatchPlanner.BuildPatchPlan
  ↓
ManifestPatchPlan + PRMetadata
  ↓
DryRunManifestWriter.WriteManagedCluster
  ↓
Updated ManagedCluster object + WriteResult
  ↓
BuildBranchPlan
  ↓
BranchPlan / CommitPlan / PullRequestPlan
```

Acceptance criteria:

```text
- CRD-style design exists
- fake controller can update status
- adapter boundary exists
- evaluated ChangeProposal can produce ManifestPatchPlan
- ManifestPatchPlan can produce updated ManagedCluster object in dry-run mode
- dry-run branch/commit/PR metadata can be generated
- PolicyChecker classifies dev 3 -> 6 as Medium
- real execution remains through GitOps / Controller, not model
```

Remaining tasks:

```text
- add YAML parser/writer after dependency choice is clear
- add Cluster API mapping design details
- add KubeVirt mapping design details
- add Metal3 mapping design details
- postpone real controller-runtime implementation
- postpone real GitHub PR creation
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
Policy versioning
Approval workflow state
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
PR-011 model structured parser
PR-012 agent ChangeProposal
PR-013 deterministic policy checker
PR-014 PR draft generator
PR-015 agent pipeline
PR-016 infrastructure scenario docs
PR-017 ManagedCluster API skeleton
PR-018 infra API validation
PR-019 fake controller
PR-020 infra adapter boundary
PR-021 GitOps ManifestPatchPlan
PR-022 ManagedCluster object patcher
PR-023 DryRunManifestWriter
PR-024 PR-ready metadata fields for ManifestPatchPlan
PR-025 dry-run branch/commit abstraction
```

Next PRs:

```text
PR-026 provider config loading
PR-027 OpenAI-compatible HTTP client
PR-028 private/self-hosted provider config examples
PR-029 Cluster API mapping design
PR-030 KubeVirt mapping design
PR-031 Metal3 mapping design
```

## 13. Immediate Next Steps

Recommended next implementation sequence:

```text
1. Run or verify go test ./... status.
2. Add provider config loading.
3. Add OpenAI-compatible HTTP client implementation.
4. Add private/self-hosted model provider config examples.
5. Add YAML parser/writer only after dependency choice is clear.
6. Keep real controller-runtime and real GitHub PR creation postponed.
```

## 14. Current Done Definition

Current done definition for this phase:

```text
1. go test ./... passes.
2. MockProvider flow passes through Gateway.
3. SafetyGuard blocks unsafe requests and outputs.
4. EvalRunner produces EvalReport.
5. Router can route based on ProviderScore.
6. Registry can register, list, and health-check providers.
7. OpenAI-compatible provider can parse strict structured JSON.
8. Agent workflow can convert ChangePlan to evaluated ChangeProposal.
9. PolicyChecker decides risk and approval deterministically.
10. PR draft generation requires policy evaluation.
11. ManagedCluster / MachineClass API types and validation exist.
12. FakeController updates status through ClusterAdapter boundary.
13. GitOps integration can produce ManifestPatchPlan and dry-run updated ManagedCluster object.
14. GitOps integration can produce dry-run branch/commit/PR metadata.
15. Real execution remains outside model and agent layers.
```
