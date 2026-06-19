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
11. Provider credentials must stay behind references and resolvers, not raw config fields.
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
Mostly complete
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
- ConfigSource -> LoadConfig -> ValidateConfig
- config defaults for timeout, retry and token limits
- raw credential rejection in provider config
- public hosted provider config example
- private enterprise gateway config example
- self-hosted vLLM config example
- fake client tests
- env-guarded OpenAI-compatible provider integration test
- strict JSON structured parser
- parser rejects markdown fenced output
- parser rejects unknown fields
- parser rejects empty output
- parser rejects unsupported schemas
- parser rejects trailing JSON content
- parser can parse ChangePlan / RollbackPlan / ValidationReport / RiskExplanation and other schema kinds
- OpenAI-compatible chat completions request body builder
- /chat/completions URL builder
- RetryPolicy with deterministic retry decisions
- TimeoutPolicy with request-scoped deadline propagation
- narrow HTTPClient with injectable HTTPDoer and SecretResolver
- HTTPClient parses choices, finish reason and token usage
- HTTPClient retries transport error and retryable status codes
- HTTPClient propagates timeout context into SecretResolver and HTTP request
- HTTPClient rejects missing resolver, empty secret, non-retryable non-2xx response and missing choices
```

Env-guarded integration test:

```text
AICLOUD_OPENAI_INTEGRATION_TEST=1
AICLOUD_OPENAI_ENDPOINT
AICLOUD_OPENAI_MODEL
AICLOUD_OPENAI_API_KEY
```

The integration test is skipped by default and must not run in normal CI unless the environment is explicitly configured.

Current provider flow:

```text
ConfigSource
  ↓
LoadConfig
  ↓
ValidateConfig
  ↓
HTTPClient
  ↓
TimeoutPolicy.WithTimeout
  ↓
BuildChatCompletionRequest
  ↓
RetryPolicy.ShouldRetry
  ↓
HTTPDoer
  ↓
CompatibleResponse
```

Remaining tasks:

```text
- add Kubernetes Secret resolver in a runtime integration package
- add external config file loader if needed
- keep streaming and tool use out of the MVP unless needed
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

First scenario:

```text
ManagedCluster workers replicas 3 -> 6
```

Implemented packages:

```text
infra/api
infra/controller
infra/adapter
integrations/gitops
```

Implemented docs and examples:

```text
docs/infra-control-plane-scenario.md
docs/managedcluster-api-design.md
infra/README.md
infra/controller/README.md
infra/adapter/README.md
integrations/gitops/README.md
examples/infra/managedcluster-dev-gpu.yaml
examples/infra/machineclass-gpu-large.yaml
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
PR-026 provider config loading
PR-027 OpenAI-compatible HTTP client
PR-028 private/self-hosted provider config examples
PR-029 retry and timeout policy refinements
PR-030 env-guarded provider integration tests
```

Next PRs:

```text
PR-031 Kubernetes Secret resolver runtime integration
PR-032 Cluster API mapping design
PR-033 KubeVirt mapping design
PR-034 Metal3 mapping design
```

## 13. Immediate Next Steps

Recommended next implementation sequence:

```text
1. Run or verify go test ./... status.
2. Add Kubernetes Secret resolver in a separate runtime integration package.
3. Add YAML parser/writer only after dependency choice is clear.
4. Add Cluster API / KubeVirt / Metal3 mapping design details.
5. Keep real controller-runtime and real GitHub PR creation postponed.
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
8. OpenAI-compatible provider has config loading and validation.
9. OpenAI-compatible provider has request body builder and narrow HTTP client.
10. OpenAI-compatible provider has public/private/self-hosted config examples without raw credentials.
11. OpenAI-compatible provider has retry and timeout policy refinements.
12. OpenAI-compatible provider has env-guarded integration test.
13. Agent workflow can convert ChangePlan to evaluated ChangeProposal.
14. PolicyChecker decides risk and approval deterministically.
15. PR draft generation requires policy evaluation.
16. ManagedCluster / MachineClass API types and validation exist.
17. FakeController updates status through ClusterAdapter boundary.
18. GitOps integration can produce ManifestPatchPlan and dry-run updated ManagedCluster object.
19. GitOps integration can produce dry-run branch/commit/PR metadata.
20. Real execution remains outside model and agent layers.
```
