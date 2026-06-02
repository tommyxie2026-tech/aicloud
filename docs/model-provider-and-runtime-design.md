# Model Provider and Runtime Design

## 1. Purpose

This document defines how `aicloud` connects public, private, self-hosted open-source, local, and future custom domain models.

The goal is to keep `aicloud` centered on:

```text
Governed hybrid model access + policy-aware agent workflows
```

This document intentionally avoids turning `aicloud` into only:

```text
- a simple model API proxy
- a foundation model training platform
- a GPU scheduler
- a chatbot
```

The first product problem is not model training.

The first product problem is:

```text
How can an enterprise safely use multiple kinds of models through one governed platform?
```

## 2. Design Principles

```text
1. Model access must be unified.
2. Private/open-source model support must be first-class.
3. Public model access must respect data boundaries.
4. Every model output is untrusted until validated.
5. Evaluation must exist before production routing.
6. Routing must consider task, risk, environment, data sensitivity, provider health, and evaluation score.
7. Models may propose; policy decides.
8. No model provider may directly execute infrastructure actions.
```

## 3. Provider Types

`aicloud` should support these provider categories:

```text
PublicGeneralLLM
PrivateEnterpriseLLM
SelfHostedOpenModel
LocalSmallModel
CustomDomainModel
MockProvider
```

## 3.1 PublicGeneralLLM

Purpose:

```text
Use strong public general-purpose models for high-quality reasoning and planning when data boundary permits.
```

Typical tasks:

```text
- architecture planning
- general reasoning
- code explanation
- non-sensitive summarization
- complex plan generation for sanitized context
```

Boundary:

```text
- must not receive restricted or confidential raw enterprise data
- must pass the same schema/safety/policy validators
- must be optional and disabled unless configured
```

Implementation adapter:

```text
model/openai
```

## 3.2 PrivateEnterpriseLLM

Purpose:

```text
Use enterprise-private model endpoints for confidential or regulated internal context.
```

Typical tasks:

```text
- private document reasoning
- internal runbook reasoning
- confidential infrastructure planning
- enterprise codebase summarization
```

Boundary:

```text
- endpoint is controlled by enterprise
- credentials are referenced by SecretRef, not stored in code
- provider must still pass evaluation and safety rules
```

Implementation adapter:

```text
OpenAI-compatible private endpoint first
Dedicated private provider adapters later
```

## 3.3 SelfHostedOpenModel

Purpose:

```text
Connect self-hosted open-source model runtimes.
```

Candidate runtimes:

```text
- vLLM-compatible endpoint
- TGI-compatible endpoint
- Ollama-compatible endpoint
- KServe-hosted endpoint
- Ray Serve-hosted endpoint
- custom internal model service
```

Target tasks:

```text
- validation report summarization
- policy explanation
- PR description generation
- runbook summarization
- simple YAML repair after evaluation
```

Boundary:

```text
- low-risk tasks first
- blocked from high-risk planning until evaluation threshold is met
- must implement ModelProvider through adapter
```

## 3.4 LocalSmallModel

Purpose:

```text
Use lightweight local models for low-cost, low-risk, local or offline tasks.
```

Typical tasks:

```text
- short summary
- simple classification
- runbook snippet summarization
- validation report wording
```

Boundary:

```text
- not used for high-risk infrastructure planning by default
- not used for production destructive workflows
- routed only by explicit policy
```

## 3.5 CustomDomainModel

Purpose:

```text
Support narrow domain-specific model experiments after data and evaluation are ready.
```

Candidate models:

```text
InfraChangeRiskClassifier
KubernetesYamlRepairModel
PolicyExplanationModel
ValidationReportSummarizer
RunbookGenerationModel
```

Prerequisites:

```text
- synthetic golden dataset
- sanitized dataset format
- baseline provider evaluation
- stable safety boundary
- no safety regression
```

## 3.6 MockProvider

Purpose:

```text
Provide deterministic local behavior for tests, CI, demos, and offline development.
```

Current first fixture:

```text
scale dev-gpu-cluster gpu-workers from 3 to 6
```

## 4. ProviderConfig

A provider must be configured through metadata, not hardcoded credentials.

Suggested config shape:

```yaml
apiVersion: ai.aicloud.dev/v1alpha1
kind: ModelProviderConfig
metadata:
  name: private-strong
spec:
  type: PrivateEnterpriseLLM
  endpointRef:
    name: private-openai-compatible-endpoint
  secretRef:
    name: private-model-secret
  defaultModel: enterprise-strong-model
  dataBoundary: ConfidentialAllowed
  supportedTasks:
    - GeneratePlan
    - GenerateRollback
    - GenerateValidationReport
    - ExplainRisk
  maxInputTokens: 32000
  maxOutputTokens: 4096
  enabled: true
```

## 5. EndpointRef and SecretRef

### EndpointRef

Endpoint configuration should be separated from provider identity:

```yaml
apiVersion: ai.aicloud.dev/v1alpha1
kind: ModelEndpoint
metadata:
  name: private-openai-compatible-endpoint
spec:
  protocol: OpenAICompatible
  baseURL: https://model-gateway.internal/v1
  healthPath: /models
  timeoutSeconds: 30
```

### SecretRef

Secret reference should never expose raw credentials in code or docs:

```yaml
secretRef:
  name: private-model-secret
  key: apiKey
```

Rules:

```text
- no raw API key in repository
- no raw API key in logs
- no raw API key in audit record
- secret material resolved only at runtime
```

## 6. DataBoundary

Provider routing must respect data sensitivity.

Suggested data boundary values:

```text
PublicOnly
InternalAllowed
ConfidentialAllowed
RestrictedDenied
LocalOnly
```

Suggested routing behavior:

| Data Sensitivity | Public Provider | Private Provider | Self-hosted Provider | Local Provider |
|---|---:|---:|---:|---:|
| Public | allowed | allowed | allowed | allowed |
| Internal | allowed with policy | allowed | allowed | allowed |
| Confidential | blocked by default | allowed | allowed | allowed |
| Restricted | blocked | blocked unless explicit | blocked unless explicit | local-only with explicit policy |

## 7. Supported Task Classes

Provider tasks should remain high-level and structured:

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

Restricted operations are not provider tasks:

```text
DirectExecution
ManifestApply
CredentialRead
MachineControl
ProductionDelete
AutoApprove
AutoMerge
```

## 8. Evaluation Before Routing

A provider should not receive production workflow tasks until it has evaluation results.

Evaluation dimensions:

```text
schema compliance
task correctness
safety compliance
policy alignment
rollback quality
evidence grounding
latency
cost
language handling
```

Evaluation result should produce:

```text
ProviderScore
AllowedTasks
BlockedTasks
SafetyFailures
SchemaFailures
RecommendedRoutingTier
```

## 9. Routing Rules

Routing inputs:

```text
task type
risk level
environment
data sensitivity
latency budget
cost budget
evaluation score
provider health
private provider requirement
```

Routing examples:

```text
Low-risk summary -> LocalSmallModel or SelfHostedOpenModel
Medium-risk planning -> PrivateEnterpriseLLM or strong hosted model with sanitized context
Confidential context -> PrivateEnterpriseLLM / SelfHostedOpenModel / LocalSmallModel
Risk classification -> deterministic PolicyChecker
Restricted operation -> block
Unevaluated provider -> block for production planning
```

## 10. Runtime Adapter Strategy

Recommended adapter order:

```text
1. MockProvider
2. OpenAI-compatible hosted/private endpoint
3. OpenAI-compatible self-hosted endpoint
4. vLLM-style runtime adapter
5. Ollama-style runtime adapter
6. TGI-style runtime adapter
7. CustomDomainProvider adapter
```

## 11. Current Implementation Mapping

Current packages:

```text
model/provider   common ModelProvider interface
model/schema     structured output schemas
model/mock       deterministic provider
model/openai     OpenAI-compatible provider and JSON parser
model/safety     request/response safety boundary
model/eval       provider evaluation runner
model/routing    static routing policy
model/registry   in-memory provider registry
model/gateway    task-level gateway
```

## 12. Non-goals for Current Stage

Do not build these too early:

```text
- foundation model training
- broad fine-tuning platform
- full model serving platform
- GPU scheduler
- multi-tenant billing
- production destructive automation
```

## 13. Next Design Tasks

```text
1. Add ModelProviderConfig schema.
2. Add ModelEndpoint schema.
3. Add provider config loader.
4. Add self-hosted runtime adapter design.
5. Add provider score persistence design.
6. Add data-boundary-aware routing policy.
7. Add custom domain model experiment protocol.
```
