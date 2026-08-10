# AI Cloud Provider / Model / Deployment 契约

> 状态：S0 Contract Freeze

## 1. 目标

通过拆分 Provider Connectivity、Logical Model Identity、Immutable ModelVersion 与 Callable Deployment，确保 AI Cloud 不绑定任何第三方 Model Catalog。

## 2. 核心分离

```text
Provider
  = 谁提供连接、如何连接

Model
  = AI Cloud 中的逻辑模型资产

ModelVersion
  = 不可变版本与 Evidence Identity

Deployment
  = 真实可调用 Endpoint / Capacity / Region
```

四者是独立 Resource，拥有不同 Lifecycle 与 Policy。

## 3. Provider

Provider 表示 Integration Adapter 以及商业/运行时边界。

示例：

```text
OpenAI API
Anthropic API
Gemini API
Azure/OpenAI-compatible endpoint
Internal vLLM cluster
Internal SGLang cluster
```

Provider 保存 Connection Protocol、Credential Reference、Supported API Surface 与 Health Collection Capability，但不拥有 AI Cloud 的 Canonical Model Registry。

## 4. Model

Model 是 Policy、Evaluation、Application Alias 使用的逻辑 AI Asset。

```yaml
model:
  model_id: string
  scope: global | tenant
  display_name: string
  modality: []string
  capability_tags: []string
  lifecycle: discovered | admitted | active | deprecated | retired
```

Application 正常情况下引用 Logical Model/Alias 或 Capability Requirement，而不是 Provider-native Model Name。

## 5. ModelVersion

ModelVersion 一旦 Admission 后不可变：

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

License、Provenance、Security Evidence 绑定 ModelVersion，而不是绑定会变化的 Display Name。

## 6. Deployment

Deployment 是 Router 真正可以调用的 Execution Target：

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

同一个 ModelVersion 可以有多个 Deployment；一个 Provider 也可以服务多个 Model 和 Deployment。

## 7. Router Target

Router 最终选择：

```text
ModelVersion + Deployment
```

而不是简单选择：

```text
provider_model_name
```

Routing Flow：

```text
Task Requirements
  -> Model Registry Candidates
  -> Admission/Policy Hard Filters
  -> Eligible ModelVersions
  -> Eligible Deployments
  -> Router Optimization
  -> Selected Deployment
```

## 8. Provider Adapter Contract

Vendor SDK Type 不能泄漏进 Domain Contract。Provider Adapter 负责把 AI Cloud Neutral Request 转换为厂商/Runtime Specific API。

最小 Port：

```text
Generate
Stream
Health
Capabilities
```

Embedding、Batch、Rerank、Image 等能力通过独立 Capability Interface 提供，而不是假设所有 Provider 都必须实现。

## 9. Logical Alias

Application 可以使用：

```text
coding-efficient
reasoning-premium
private-confidential
```

Alias Resolution 返回符合 Policy 的 ModelVersion/Deployment，并允许未来调整 Mapping 而不修改 Application Code。

Alias 不能绕过 Policy；它只是 Candidate Selector，不是 Authorization Grant。

## 10. Lifecycle

Provider、Model、ModelVersion、Deployment Lifecycle 相互独立。

例如：

- Provider Degraded 不代表 Logical Model Asset 失效；
- Deployment Draining 不代表 ModelVersion Retired；
- 某个 ModelVersion Deprecated 时，其他 Version 可以继续 Active；
- Provider 可以被移除，而 Logical Model History/Evidence 仍然保留。

## 11. Admission 与 Evaluation

Admission 判断 ModelVersion 是否有资格进入/使用于 AI Cloud，依据 Provenance、License、Artifact Integrity、Security Evidence。

Evaluation 判断该版本是否适合某个 Task/Release。

```text
ModelVersion
  -> Admission Gate
  -> Registry Eligible
  -> Evaluation Gate
  -> Production/Task Eligible
```

## 12. Pricing

Pricing 必须版本化并绑定 Deployment/Service Tier，而不是硬编码在 Model Name 上。CostEvent 保存执行时实际使用的 Pricing Version。

## 13. 验收条件

- Domain API 不依赖 Provider-native SDK Type；
- RouteDecision 同时标识 ModelVersion 与 Deployment；
- 替换 Deployment 不改变 Task/Agent Contract；
- 删除 Provider 不删除 Logical Model History/Evidence；
- Admission Evidence 对 ModelVersion 不可变；
- Pricing Change 不改写历史 CostEvent；
- Logical Alias 可以重新 Mapping 而 Application Code 不变。

## 14. 对实现的影响

S2 冻结 API/Schema Resource Identifier；S6 在本分离模型上实现 Deployment-aware Routing、Live Health/Capacity 与 Provider Failover。