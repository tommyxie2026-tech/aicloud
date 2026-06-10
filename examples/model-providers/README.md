# Model Provider Examples

This directory contains example OpenAI-compatible provider configurations for public, private, and self-hosted model endpoints.

These examples are intentionally credential-free.

## Safety Boundary

Provider examples must use:

```text
SecretRef
```

They must not include:

```text
raw API keys
bearer tokens
passwords
private keys
inline secrets
```

## Current Examples

```text
public-openai-compatible.yaml
private-enterprise-gateway.yaml
self-hosted-vllm.yaml
```

## Mapping to model/openai.ConfigSource

Example fields map to:

```text
name            -> ConfigSource.Name
endpoint        -> ConfigSource.Endpoint
endpointRef     -> ConfigSource.EndpointRef
secretRef       -> ConfigSource.SecretRef
defaultModel    -> ConfigSource.DefaultModel
timeoutSeconds  -> ConfigSource.TimeoutSeconds
maxRetries      -> ConfigSource.MaxRetries
maxInputTokens  -> ConfigSource.MaxInputTokens
maxOutputTokens -> ConfigSource.MaxOutputTokens
private         -> ConfigSource.Private
```

Exactly one of the following should be configured:

```text
endpoint
endpointRef
```

## Intended Runtime Flow

```text
Example provider config
  ↓
ConfigSource
  ↓
LoadConfig
  ↓
ValidateConfig
  ↓
HTTPClient with SecretResolver
  ↓
OpenAI-compatible endpoint
```

## Not Included

```text
- real API keys
- Kubernetes Secrets
- production endpoint credentials
- live integration tests
```
