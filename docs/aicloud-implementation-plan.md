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

Current dependency note:

```text
gopkg.in/yaml.v3 is declared in go.mod.
go.sum has not been confirmed by go mod tidy in this workflow.
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

Implemented packages:

```text
model/openai
runtime/secrets
runtime/secrets/kubernetes
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
- HTTPClient accepts runtime/secrets.MemoryResolver without an adapter
- HTTPClient rejects missing resolver, empty secret, non-retryable non-2xx response and missing choices
- runtime/secrets Resolver interface
- runtime/secrets SecretRef parser
- runtime/secrets MemoryResolver for tests and local wiring
- runtime/secrets/kubernetes design document
- runtime/secrets/kubernetes fakeable SecretGetter boundary
- runtime/secrets/kubernetes namespace allowlist
- runtime/secrets/kubernetes fail-closed resolver skeleton
- runtime/secrets/kubernetes tests for allowed namespace, missing secret, missing key, empty value and canceled context
```

Env-guarded integration test:

```text
AICLOUD_OPENAI_INTEGRATION_TEST=1
AICLOUD_OPENAI_ENDPOINT
AICLOUD_OPENAI_MODEL
AICLOUD_OPENAI_API_KEY
```

The integration test is skipped by default and must not run in normal CI unless the environment is explicitly configured.

Runtime secret reference format:

```text
secret/<namespace>/<name>:<key>
```

Current provider flow:

```text
ConfigSource
  ↓
LoadConfig
  ↓
ValidateConfig
  ↓
runtime/secrets.Resolver
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

Future Kubernetes resolver flow:

```text
SecretRef
  ↓
runtime/secrets.ParseSecretRef
  ↓
namespace allowlist
  ↓
SecretGetter.GetSecret
  ↓
key lookup
  ↓
non-empty value
```

Remaining tasks:

```text
- add client-go-backed SecretGetter after dependency and RBAC boundaries are finalized
- add external config file loader if needed
- keep streaming and tool use out of the MVP unless needed
```

Important boundary:

```text
runtime/secrets/kubernetes still does not read real Kubernetes Secrets.
It does not import client-go or controller-runtime yet.
It only proves the Kubernetes resolver boundary with a fakeable SecretGetter.
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
infra/mapping/clusterapi
infra/mapping/kubevirt
infra/mapping/metal3
integrations/gitops
integrations/gitops/yamlio
```

Implemented docs and examples:

```text
docs/infra-control-plane-scenario.md
docs/managedcluster-api-design.md
docs/cluster-api-mapping-design.md
docs/kubevirt-mapping-design.md
docs/metal3-mapping-design.md
docs/yaml-parser-writer-dependency-decision.md
infra/README.md
infra/controller/README.md
infra/adapter/README.md
infra/mapping/clusterapi/README.md
infra/mapping/kubevirt/README.md
infra/mapping/metal3/README.md
integrations/gitops/README.md
integrations/gitops/yamlio/README.md
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
DesiredCluster
DesiredMachineDeployment
Cluster API MappingResult
Cluster API MappingError
ClusterAPI Mapper
DesiredVirtualMachine
KubeVirt MappingResult
KubeVirt MappingError
KubeVirt Mapper
DesiredBareMetalHostClaim
Metal3 MappingResult
Metal3 MappingError
Metal3 Mapper
ManagedClusterYAML
ManagedClusterSpecYAML
WorkerGroupYAML
YAMLIOError
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
- Cluster API mapping design
- provider-neutral DesiredCluster shape
- provider-neutral DesiredMachineDeployment shape
- ManagedCluster -> DesiredMachineDeployment[] mapper
- MachineDeployment patch path helper
- KubeVirt mapping design
- provider-neutral DesiredVirtualMachine shape
- ManagedCluster + MachineClass[] -> DesiredVirtualMachine[] mapper
- KubeVirt replica expansion into stable VM identities
- KubeVirt missing MachineClass fail-closed behavior
- Metal3 mapping design
- provider-neutral DesiredBareMetalHostClaim shape
- ManagedCluster + MachineClass[] -> DesiredBareMetalHostClaim[] mapper
- Metal3 replica expansion into stable host claim identities
- Metal3 missing MachineClass fail-closed behavior
- YAML parser/writer dependency decision
- gopkg.in/yaml.v3 declared in go.mod
- ManagedCluster YAML read/write skeleton
- ManagedCluster YAML round-trip tests
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

Current yamlio flow:

```text
YAML bytes
  ↓
yaml.Unmarshal
  ↓
ManagedClusterYAML DTO
  ↓
infra/api.ManagedCluster
  ↓
api.ValidateManagedCluster
```

Write flow:

```text
infra/api.ManagedCluster
  ↓
api.ValidateManagedCluster
  ↓
ManagedClusterYAML DTO
  ↓
yaml.Marshal
  ↓
YAML bytes
```

Current Cluster API mapping flow:

```text
ManagedCluster
  ↓
clusterapi.Mapper.MapManagedCluster
  ↓
MappingResult
  ↓
DesiredCluster
  ↓
DesiredMachineDeployment[]
```

Current KubeVirt mapping flow:

```text
ManagedCluster + MachineClass[]
  ↓
kubevirt.Mapper.MapManagedCluster
  ↓
MappingResult
  ↓
DesiredVirtualMachine[]
```

Current Metal3 mapping flow:

```text
ManagedCluster + MachineClass[]
  ↓
metal3.Mapper.MapManagedCluster
  ↓
MappingResult
  ↓
DesiredBareMetalHostClaim[]
```

Remaining tasks:

```text
- run go mod tidy to confirm go.sum
- run or verify go test ./... status
- wire yamlio into DryRunManifestWriter after tests stabilize
- postpone real controller-runtime implementation
- postpone real GitHub PR creation
```

Important yamlio boundary:

```text
yamlio does not read files.
yamlio does not write files.
yamlio does not call GitHub APIs.
yamlio does not call Kubernetes APIs.
yamlio only transforms bytes into internal objects and internal objects into bytes.
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
PR-031 runtime secret resolver integration
PR-032 Cluster API mapping design
PR-033 KubeVirt mapping design
PR-034 Metal3 mapping design
PR-035 Kubernetes-backed Secret resolver design
PR-036 YAML parser/writer dependency decision and yamlio skeleton
```

Next PRs:

```text
PR-037 go test ./... stabilization
PR-038 wire yamlio into DryRunManifestWriter
```

## 13. Immediate Next Steps

Recommended next implementation sequence:

```text
1. Run go mod tidy to generate or verify go.sum.
2. Run or verify go test ./... status.
3. Fix compile issues from new yamlio dependency and existing tests.
4. Wire yamlio into DryRunManifestWriter only after tests stabilize.
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
13. Runtime secret resolver boundary exists and can inject MemoryResolver into HTTPClient.
14. Kubernetes-backed Secret resolver design and fakeable skeleton exist without client-go.
15. Agent workflow can convert ChangePlan to evaluated ChangeProposal.
16. PolicyChecker decides risk and approval deterministically.
17. PR draft generation requires policy evaluation.
18. ManagedCluster / MachineClass API types and validation exist.
19. FakeController updates status through ClusterAdapter boundary.
20. GitOps integration can produce ManifestPatchPlan and dry-run updated ManagedCluster object.
21. GitOps integration can produce dry-run branch/commit/PR metadata.
22. Cluster API mapping design and provider-neutral mapper skeleton exist.
23. KubeVirt mapping design and provider-neutral mapper skeleton exist.
24. Metal3 mapping design and provider-neutral mapper skeleton exist.
25. yamlio can read/write ManagedCluster YAML bytes in skeleton form.
26. Real execution remains outside model and agent layers.
```
