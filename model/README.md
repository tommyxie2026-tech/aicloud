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
```

## 4. provider

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

Supported safe task types:

```text
GeneratePlan
GeneratePatch
ExplainRisk
GenerateRollback
GenerateValidationReport
SummarizeState
RepairYAML
ExplainPolicyFailure
```

## 5. schema

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

The validator checks required schema fields such as:

```text
schemaVersion
kind
requestId
taskType
createdBy
target.kind
target.name
changes
rollback.summary
evidence
```

## 6. mock

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

## 7. First Working Flow

The first model-layer flow is:

```text
MockProvider.GeneratePlan
  ↓
schema.ChangePlan
  ↓
schema.BasicValidator.ValidateChangePlan
  ↓
go test ./...
```

This flow proves that `aicloud` can produce and validate a structured model output without relying on any external model provider.

## 8. Tests

Current tests:

```text
model/mock/provider_test.go
```

Test coverage:

```text
- MockProvider.GeneratePlan returns ChangePlan
- ChangePlan passes BasicValidator
- MockProvider blocks restricted instruction
- MockProvider.Health is available
```

Run locally:

```bash
go test ./...
```

CI:

```text
.github/workflows/go-test.yml
```

## 9. Next Packages

Recommended next packages:

```text
model/safety
model/gateway
model/eval
model/routing
model/openai
```

Recommended implementation order:

```text
1. model/safety   - request/response safety validation
2. model/gateway  - task-level model API
3. model/eval     - provider evaluation harness
4. model/routing  - provider selection policy
5. model/openai   - OpenAI-compatible public/private provider adapter
```

## 10. Next Milestone

Next milestone:

```text
MockProvider
  ↓
SafetyGuard
  ↓
ModelGateway.GeneratePlan
  ↓
BasicValidator
  ↓
EvalRunner
```

The model layer should stay dependency-light until this path is stable.
