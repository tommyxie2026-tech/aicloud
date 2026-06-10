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
model/openai/http_request.go
model/openai/http_client.go
model/openai/*_test.go
examples/model-providers/README.md
examples/model-providers/public-openai-compatible.yaml
examples/model-providers/private-enterprise-gateway.yaml
examples/model-providers/self-hosted-vllm.yaml
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

The runtime implementation should resolve `SecretRef` outside config loading.

## 7. Provider Config Examples

Current examples:

```text
examples/model-providers/public-openai-compatible.yaml
examples/model-providers/private-enterprise-gateway.yaml
examples/model-providers/self-hosted-vllm.yaml
```

Example categories:

```text
public hosted API
private enterprise model gateway
self-hosted OpenAI-compatible vLLM endpoint
```

All examples intentionally use `SecretRef` only. They do not include real API keys.

## 8. Public Hosted Provider Example

```text
Name:         openai-public
Endpoint:     https://api.openai-compatible.example/v1
SecretRef:    secret/openai-public
DefaultModel: gpt-test
Private:      false
```

## 9. Private Enterprise Gateway Example

```text
Name:         private-enterprise-gateway
EndpointRef:  endpoint/private-model-gateway
SecretRef:    secret/private-model-gateway
DefaultModel: qwen-enterprise
Private:      true
```

## 10. Self-Hosted Open Model Example

```text
Name:         self-hosted-vllm
EndpointRef:  endpoint/vllm-internal
SecretRef:    secret/vllm-internal
DefaultModel: qwen2.5-coder
Private:      true
```

## 11. HTTP Request Builder

`BuildChatCompletionRequest` converts:

```text
CompatibleRequest
```

into:

```text
ChatCompletionRequest
```

It builds an OpenAI-compatible `/chat/completions` request body.

Current request fields:

```text
model
messages
temperature
max_tokens
stream=false
response_format=json_object when OutputSchema is set
```

It does not:

```text
- send HTTP requests
- attach credentials
- resolve secrets
```

## 12. Narrow HTTP Client

`HTTPClient` uses injected interfaces:

```text
HTTPDoer
SecretResolver
```

Current flow:

```text
CompatibleRequest
  ↓
BuildChatCompletionRequest
  ↓
BuildChatCompletionsURL
  ↓
SecretResolver.ResolveSecret
  ↓
HTTPDoer.Do
  ↓
CompatibleResponse
```

The default implementation does not read environment variables, files, or Kubernetes Secrets directly.

## 13. Safety Properties

Current config and HTTP layers fail closed when:

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
- model is missing from request
- instruction is missing from request
- endpoint URL is missing
- SecretResolver is missing
- resolved secret is empty
- HTTP response is non-2xx
- HTTP response has no choices
```

## 14. Provider Capabilities

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

## 15. Not Done Yet

```text
- retry policy implementation
- timeout propagation refinements
- streaming
- tool use
- env-guarded integration tests
- Kubernetes Secret resolver
- external config file loader
```

## 16. Recommended Next Steps

```text
1. Add retry policy implementation.
2. Keep credential resolution outside config loading.
3. Add integration tests only behind explicit environment variables.
4. Add a Kubernetes Secret resolver in a separate runtime integration package.
5. Add external config file loader only if needed.
```
