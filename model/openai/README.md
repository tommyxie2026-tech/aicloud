# OpenAI-Compatible Provider

## 1. Goal

The `model/openai` package provides an OpenAI-compatible provider adapter for:

```text
- public hosted model APIs
- private enterprise model gateways
- self-hosted open model servers
- local OpenAI-compatible endpoints
```

The provider exposes the shared `model/provider.ModelProvider` interface while keeping endpoint configuration and credential handling separated from model execution.

## 2. Current Files

```text
model/openai/provider.go
model/openai/parser.go
model/openai/config_loader.go
model/openai/*_test.go
```

## 3. Configuration Model

The main runtime config is:

```text
Config
```

The loader input is:

```text
ConfigSource
```

`LoadConfig` converts `ConfigSource` into validated `Config`.

## 4. Required Fields

A valid config requires:

```text
Name
DefaultModel
SecretRef
Endpoint or EndpointRef
```

Exactly one of the following must be set:

```text
Endpoint
EndpointRef
```

## 5. Defaults

`LoadConfig` applies these defaults:

```text
TimeoutSeconds  = 30
MaxRetries      = 2
MaxInputTokens  = 32768
MaxOutputTokens = 2048
```

## 6. Credential Boundary

Raw credentials must not be stored in provider config.

Allowed:

```text
SecretRef: secret/openai-public
SecretRef: secret/private-model-gateway
SecretRef: secret/local-vllm
```

Rejected examples:

```text
sk-...
bearer ...
api_key=...
apikey=...
token=...
```

The runtime implementation should resolve `SecretRef` outside this package.

## 7. Public Hosted Provider Example

```text
Name:         openai-public
Endpoint:     https://api.openai-compatible.example/v1
SecretRef:    secret/openai-public
DefaultModel: gpt-test
Private:      false
```

## 8. Private Enterprise Gateway Example

```text
Name:         private-enterprise-gateway
EndpointRef:  endpoint/private-model-gateway
SecretRef:    secret/private-model-gateway
DefaultModel: qwen-enterprise
Private:      true
```

## 9. Self-Hosted Open Model Example

```text
Name:         self-hosted-vllm
EndpointRef:  endpoint/vllm-internal
SecretRef:    secret/vllm-internal
DefaultModel: qwen2.5-coder
Private:      true
```

## 10. Safety Properties

Current config loading fails closed when:

```text
- name is missing
- defaultModel is missing
- endpoint and endpointRef are both missing
- endpoint and endpointRef are both set
- secretRef is missing
- secretRef looks like a raw credential
- timeoutSeconds is invalid
- maxRetries is invalid
- maxInputTokens is invalid
- maxOutputTokens is invalid
```

## 11. Provider Capabilities

The provider advertises:

```text
- structured output
- JSON schema
- long context
- Chinese support
- code generation
- local deployment when Private=true
```

Restricted capabilities include:

```text
- direct execution
- manifest apply
- credential read
- machine control
- production delete
- auto approve
- auto merge
```

## 12. Not Done Yet

```text
- real HTTP client
- retry policy implementation
- timeout propagation to HTTP request
- streaming
- tool use
- env-guarded integration tests
- Kubernetes Secret resolver
- external config file loader
```

## 13. Recommended Next Steps

```text
1. Add OpenAI-compatible HTTP request body builder.
2. Add a narrow HTTP client implementation.
3. Keep credential resolution outside this package.
4. Add provider config examples for public/private/self-hosted endpoints.
5. Add integration tests only behind explicit environment variables.
```
