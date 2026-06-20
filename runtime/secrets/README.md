# Runtime Secrets

## Goal

`runtime/secrets` provides a lightweight secret resolver boundary for runtime integrations.

It exists so provider configs can keep credentials behind references instead of embedding raw API keys.

## Current Boundary

```text
Provider Config
  ↓
SecretRef
  ↓
runtime/secrets.Resolver
  ↓
resolved secret value
```

## Supported Reference Format

```text
secret/<namespace>/<name>:<key>
```

Example:

```text
secret/aicloud-system/openai-public:api-key
```

## Current Implementation

```text
Resolver interface
SecretRef parser
MemoryResolver
SecretError
```

`MemoryResolver` is intended for unit tests and local wiring tests only.

## Safety Rules

```text
- Do not store raw API keys in provider config.
- Do not log resolved secret values.
- Do not read Kubernetes Secrets in this package yet.
- Do not introduce controller-runtime or client-go here yet.
```

## OpenAI-Compatible Provider Integration

`runtime/secrets.MemoryResolver` already satisfies `model/openai.SecretResolver` because it exposes:

```text
ResolveSecret(ctx context.Context, ref string) (string, error)
```

This is verified by:

```text
model/openai/http_client_runtime_secret_test.go
```

## Not Done Yet

```text
- Kubernetes-backed Secret resolver
- namespace scoping policy
- RBAC-aware secret access
- audit event emission for secret resolution
```
