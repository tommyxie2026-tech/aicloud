# Model Layer

## 1. Goal

The `model/` directory is the hybrid model gateway and governance layer for `aicloud`.

It provides the foundation for connecting:

```text
- public general-purpose large models
- enterprise private large models
- self-hosted open-source models
- local small models
- future domain-specific custom models
```

through a unified, safe, auditable interface.

## 2. Core Principle

```text
Models propose.
Policy decides.
Humans approve when required.
Controllers execute.
```

The model layer may generate structured proposals and explanations.

It must not directly perform infrastructure execution, credential access, approval, or merge operations.

## 3. Current Packages

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

## 4. Package Responsibilities

### provider

Path:

```text
model/provider/provider.go
```

Purpose:

```text
Define the common provider abstraction for all model backends.
```

Important types:

```text
ModelProvider
ProviderType
TaskType
ProviderCapabilities
ProviderRequest
ModelContext
ProviderResponse
ProviderHealth
ProviderError
```

Supported provider types:

```text
Hosted
Private
Local
Mock
CustomDomain
```

### schema

Path:

```text
model/schema/schema.go
model/schema/validator.go
```

Purpose:

```text
Define and validate structured model output.
```

Current structured output types:

```text
ChangePlan
YamlPatchProposal
RiskExplanation
RollbackPlan
ValidationReport
StateSummary
PolicyFailureExplanation
```

Validator:

```text
BasicValidator
```

### mock

Path:

```text
model/mock/provider.go
model/mock/provider_test.go
```

Purpose:

```text
Provide a deterministic provider for tests and offline development.
```

Current supported fixture:

```text
scale dev-gpu-cluster gpu-workers from 3 to 6
```

Expected output:

```text
ChangePlan
- target: ManagedCluster/dev-gpu-cluster
- field: spec.workers[name=gpu-workers].replicas
- from: 3
- to: 6
- riskHint: Medium
- rollback: set replicas back to 3
```

### safety

Path:

```text
model/safety/safety.go
model/safety/safety_test.go
```

Purpose:

```text
Validate model requests and responses before they enter workflow or policy layers.
```

Current checks:

```text
- restricted instruction detection
- sensitive context detection
- forbidden patch field detection
- editable field allowlist
- evidence requirement for validation reports
```

### gateway

Path:

```text
model/gateway/gateway.go
model/gateway/gateway_test.go
```

Purpose:

```text
Expose task-level model APIs to agent workflows.
```

Current API:

```text
GeneratePlan
```

Current flow:

```text
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

### eval

Path:

```text
model/eval/eval.go
model/eval/eval_test.go
```

Purpose:

```text
Evaluate provider quality before routing real tasks to a provider.
```

Current evaluation case:

```text
DefaultDevScaleOutCase
```

Evaluation dimensions:

```text
SchemaCompliance
TaskCorrectness
SafetyCompliance
PolicyAlignment
RollbackQuality
EvidenceGrounding
LanguageHandling
```

### routing

Path:

```text
model/routing/router.go
model/routing/router_test.go
```

Purpose:

```text
Route tasks to providers based on task type, risk, environment, data sensitivity and provider evaluation score.
```

Current routing outputs:

```text
RouteDecision
```

Current behavior:

```text
- restricted data fails closed
- risk classification routes to deterministic-policy
- provider must support task type
- provider must meet evaluation threshold
```

### openai

Path:

```text
model/openai/provider.go
model/openai/provider_test.go
```

Purpose:

```text
Provide an OpenAI-compatible adapter for public, private, and self-hosted model endpoints.
```

Current status:

```text
Skeleton implemented.
Structured parser intentionally fails closed until implemented.
No raw API key is stored in code.
Config uses Endpoint / EndpointRef / SecretRef.
```

### registry

Path:

```text
model/registry/registry.go
model/registry/registry_test.go
```

Purpose:

```text
Register, list, query and health-check model providers.
```

Current implementation:

```text
MemoryRegistry
```

## 5. Current Working Flows

### GeneratePlan flow

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
ChangePlan
```

### Evaluation flow

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

### Routing flow

```text
RouteRequest
  ↓
StaticRouter
  ↓
ProviderScore / Risk / Environment / DataSensitivity
  ↓
RouteDecision
```

### Registry flow

```text
Provider
  ↓
MemoryRegistry.Register
  ↓
List / Get / Health
```

## 6. Tests

Current package tests:

```text
model/mock/provider_test.go
model/safety/safety_test.go
model/gateway/gateway_test.go
model/eval/eval_test.go
model/routing/router_test.go
model/openai/provider_test.go
model/registry/registry_test.go
```

Run locally:

```bash
go test ./...
```

CI:

```text
.github/workflows/go-test.yml
```

## 7. Current Model-layer MVP Status

Current status:

```text
Model-layer MVP skeleton is implemented.
```

Done:

```text
- Provider abstraction
- Structured output schemas
- Basic schema validator
- Deterministic MockProvider
- SafetyGuard
- Gateway.GeneratePlan
- EvalRunner
- StaticRouter
- OpenAI-compatible provider skeleton
- MemoryRegistry
- Unit tests
- GitHub Actions go test workflow
```

Not done yet:

```text
- strict structured parser for OpenAI-compatible provider
- provider config loading
- persistent registry
- persistent audit store
- cost / latency metrics
- agent ChangeProposal
- deterministic PolicyChecker
- PR draft generator
- infrastructure scenario implementation
```

## 8. Next Engineering Steps

Recommended next implementation sequence:

```text
1. Fix any CI failures from current packages.
2. Implement strict JSON/YAML structured parser for OpenAI-compatible provider.
3. Add provider config loading.
4. Add agent ChangeProposal model.
5. Add deterministic PolicyChecker MVP.
6. Add PR draft generator.
7. Start infra scenario design for ManagedCluster workers 3 -> 6.
```

## 9. Related Docs

```text
docs/aicloud-positioning.md
docs/aicloud-product-architecture.md
docs/aicloud-implementation-plan.md
```
