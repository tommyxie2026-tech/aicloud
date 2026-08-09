# AI Cloud Provider / Model / Deployment Contract

> Status: S0 Contract Freeze

## 1. Purpose

Prevent AI Cloud from becoming coupled to any third-party model catalog by separating provider connectivity, logical model identity, immutable model versions and callable deployments.

## 2. Core Separation

```text
Provider
  = who/how we connect

Model
  = logical model asset in AI Cloud

ModelVersion
  = immutable version/evidence identity

Deployment
  = concrete callable endpoint/capacity/location
```

These are distinct resources with separate lifecycle and policy.

## 3. Provider

Provider represents an integration adapter and commercial/runtime boundary.

Examples:

```text
OpenAI API
Anthropic API
Gemini API
Azure/OpenAI-compatible endpoint
Internal vLLM cluster
Internal SGLang cluster
```

Provider contains connection protocol, credential reference, supported API surface and health collection capabilities. It does not own the canonical Model Registry.

## 4. Model

Model is the logical AI asset used by policy, evaluation and application aliases.

```yaml
model:
  model_id: string
  scope: global | tenant
  display_name: string
  modality: []string
  capability_tags: []string
  lifecycle: discovered | admitted | active | deprecated | retired
```

Applications should normally reference a logical model/alias or required capability rather than a provider-native model name.

## 5. ModelVersion

ModelVersion is immutable once admitted.

```yaml
model_version:
  model_version_id: string
  model_id: string
  version: string
  artifact_or_vendor_ref: string
  capability_profile: object
  context_limits: object
  admission_status: string
  admission_evidence_ref: string
  evaluation_baseline_ref: string?
  created_at: timestamp
```

License, provenance and security evidence bind to ModelVersion, not a mutable display name.

## 6. Deployment

Deployment is the actual routable execution target.

```yaml
deployment:
  deployment_id: string
  model_version_id: string
  provider_id: string
  endpoint_ref: string
  region: string
  residency: string
  service_tier: string
  pricing_ref: string
  health: string
  quota: object
  capacity: object
  tenant_scope: string?
  status: active | draining | unavailable | retired
```

Multiple deployments may serve the same ModelVersion. One provider may expose many deployments and many models.

## 7. Routing Target

Router selects:

```text
ModelVersion + Deployment
```

not merely:

```text
provider_model_name
```

Routing flow:

```text
Task requirements
  -> Model Registry candidates
  -> Admission/Policy hard filters
  -> Eligible ModelVersions
  -> Eligible Deployments
  -> Router optimization
  -> selected deployment
```

## 8. Provider Adapter Contract

Provider SDK types must not leak into Domain contracts. Provider adapters translate between AI Cloud neutral requests and vendor/runtime-specific APIs.

Minimum port:

```text
Generate
Stream
Health
Capabilities
```

Optional capabilities such as embeddings, batch, rerank or image generation are separate capability interfaces, not assumptions built into every provider.

## 9. Logical Aliases

Applications may use logical aliases such as:

```text
coding-efficient
reasoning-premium
private-confidential
```

Alias resolution returns policy-eligible ModelVersions/Deployments and can change without application code changes.

Aliases never bypass policy; they are candidate selectors, not authorization grants.

## 10. Lifecycle

Provider, Model, ModelVersion and Deployment have independent lifecycle.

Examples:

- a Provider may be degraded while its Models remain valid assets;
- a Deployment may be drained without retiring the ModelVersion;
- a ModelVersion may be deprecated while another version remains active;
- a Provider may be removed and the same logical Model served by another deployment.

## 11. Admission vs Evaluation

Admission decides whether a ModelVersion is eligible to exist/use in AI Cloud based on provenance, license, artifact integrity and security evidence.

Evaluation determines suitability/performance for a task/release.

```text
ModelVersion
  -> Admission Gate
  -> Registry Eligible
  -> Evaluation Gate
  -> Production/Task Eligible
```

## 12. Pricing

Pricing is versioned and bound to Deployment/service tier, not hard-coded on Model names. CostEvent records the pricing version used at execution time.

## 13. Acceptance Criteria

- Domain APIs never require provider-native SDK types.
- Router decision identifies both model version and deployment.
- A deployment can be replaced without changing Task/Agent contracts.
- A provider can be removed without deleting logical model history/evidence.
- Admission evidence is immutable per ModelVersion.
- Pricing changes do not rewrite historical CostEvents.
- Logical aliases can be remapped without application code changes.

## 14. Implementation Impact

S2 freezes API/schema resource identifiers. S6 implements deployment-aware routing, live health/capacity and provider failover behind this separation.