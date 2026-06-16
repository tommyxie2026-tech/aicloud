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
model/openai/retry_policy.go
model/openai/timeout_policy.go
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

## 12. Retry Policy

`RetryPolicy` provides deterministic retry decisions.

Retryable cases:

```text
transport error
408 Request Timeout
429 Too Many Requests
502 Bad Gateway
503 Service Unavailable
504 Gateway Timeout
```

Non-retryable examples:

```text
400 Bad Request
401 Unauthorized
403 Forbidden
404 Not Found
500 Internal Server Error
max retries reached
```

## 13. Timeout Policy

`TimeoutPolicy` normalizes timeout seconds and derives request-scoped context deadlines.

Current behavior:

```text
timeoutSeconds < 1 -> DefaultTimeoutSeconds
nil parent context -> context.Background()
Generate -> context with deadline
SecretResolver receives deadline context
HTTP request receives deadline context
```

## 14. Narrow HTTP Client

`HTTPClient` uses injected interfaces:

```text
HTTPDoer
SecretResolver
```

Current flow:

```text
CompatibleRequest
  ↓
TimeoutPolicy.WithTimeout
  ↓
BuildChatCompletionRequest
  ↓
BuildChatCompletionsURL
  ↓
SecretResolver.ResolveSecret
  ↓
RetryPolicy.ShouldRetry
  ↓
HTTPDoer.Do
  ↓
CompatibleResponse
```

The default implementation does not read environment variables, files, or Kubernetes Secrets directly.

## 15. Safety Properties

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
- HTTP response is non-2xx and non-retryable
- HTTP response has no choices
- max retries reached
```

## 16. Provider Capabilities

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

## 17. Not Done Yet

```text
- streaming
- tool use
- env-guarded integration tests
- Kubernetes Secret resolver
- external config file loader
```

## 18. Recommended Next Steps

```text
1. Add integration tests only behind explicit environment variables.
2. Add a Kubernetes Secret resolver in a separate runtime integration package.
3. Add external config file loader only if needed.
4. Keep streaming and tool use out of the MVP unless needed.
```
