# Kubernetes-backed Secret Resolver Skeleton

## Goal

`runtime/secrets/kubernetes` defines a future Kubernetes-backed implementation boundary for `runtime/secrets.Resolver`.

This package currently does not import `client-go`, does not connect to an API server, and does not read real Kubernetes Secrets.

## Current Input

```text
secret/<namespace>/<name>:<key>
```

The parser is reused from:

```text
runtime/secrets.ParseSecretRef
```

## Current Components

```text
SecretGetter
Resolver
ResolverConfig
ResolverError
SecretData
```

## Current Flow

```text
ResolveSecret(ctx, ref)
  ↓
ParseSecretRef
  ↓
namespace allowlist check
  ↓
SecretGetter.GetSecret
  ↓
key lookup
  ↓
non-empty value return
```

## Fakeable Backend

The resolver depends on:

```go
SecretGetter interface {
    GetSecret(ctx context.Context, namespace string, name string) (SecretData, error)
}
```

This keeps the package testable without a real Kubernetes API server.

A future client-go implementation can satisfy this interface.

## Namespace Policy

The resolver requires explicit allowed namespaces.

If no namespace allowlist is configured, `NewResolver` fails closed.

If a SecretRef points to a namespace outside the allowlist, `ResolveSecret` fails closed.

## Error Codes

Current normalized error codes:

```text
MissingSecretGetter
MissingAllowedNamespace
InvalidSecretRef
NamespaceNotAllowed
SecretNotFound
SecretKeyNotFound
EmptySecretValue
ContextCanceled
```

## Security Boundary

The resolver must not log or expose resolved values in errors.

The provider layer must continue to depend only on:

```text
runtime/secrets.Resolver
```

Provider packages must not import Kubernetes packages directly.

## Not Done Yet

```text
- client-go SecretGetter
- RBAC manifests
- real API server integration test
- metrics
- audit sink integration
- caching
```

## Next Step

Add a client-go-backed `SecretGetter` only after dependency, RBAC and deployment boundaries are finalized.
