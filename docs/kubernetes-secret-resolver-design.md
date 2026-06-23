# Kubernetes-backed Secret Resolver Design

## 1. Goal

This document defines the design boundary for a future Kubernetes-backed Secret resolver in `aicloud`.

The current `runtime/secrets` package only contains:

```text
Resolver interface
SecretRef parser
MemoryResolver
```

This document defines how a future implementation may read Kubernetes Secret values safely.

This PR does not implement real Kubernetes Secret reads.

## 2. Current Secret Reference Format

Current supported reference format:

```text
secret/<namespace>/<name>:<key>
```

Example:

```text
secret/aicloud-system/openai-public:api-key
```

The parser already validates:

```text
namespace is present
name is present
key is present
format is exact
```

## 3. Future Package Layout

Recommended future layout:

```text
runtime/secrets/kubernetes/
  README.md
  resolver.go
  resolver_test.go
```

This should remain separate from the lightweight `runtime/secrets` core package.

Reason:

```text
runtime/secrets should stay dependency-light
Kubernetes-backed resolver can import client-go later
provider packages should depend only on the Resolver interface
```

## 4. Dependency Boundary

Allowed future dependency:

```text
k8s.io/client-go/kubernetes
```

Optional future dependency:

```text
k8s.io/apimachinery
```

Not required for this design step:

```text
controller-runtime
custom informer cache
dynamic client
```

The first implementation should use a narrow Kubernetes Secret client abstraction so it can be tested with fakes.

## 5. Resolver Contract

Future Kubernetes resolver should implement:

```go
ResolveSecret(ctx context.Context, ref string) (string, error)
```

Behavior:

```text
1. Parse SecretRef.
2. Check namespace policy.
3. Read Kubernetes Secret by namespace and name.
4. Read requested key.
5. Return secret value only to caller.
6. Never log the resolved value.
```

## 6. Namespace Policy

The resolver should support an explicit namespace allowlist.

Example configuration shape:

```text
AllowedNamespaces:
  - aicloud-system
  - model-gateways
```

Default policy should be restrictive.

Recommended default:

```text
only allow the controller runtime namespace
```

A request for a namespace outside the allowlist should fail closed.

## 7. RBAC Boundary

The service account should only have permission to read explicitly allowed Secret objects or namespaces.

Recommended RBAC policy:

```text
verbs:
  - get
resources:
  - secrets
resourceNames:
  - explicitly-approved-secret-name
```

If resourceNames are not practical, namespace scope should still be narrow.

Avoid cluster-wide Secret read permissions for the first implementation.

## 8. Audit Boundary

The resolver should emit audit metadata without exposing secret values.

Allowed audit fields:

```text
resolver kind
secret namespace
secret name
secret key name
request id
caller component
success or failure
error code
latency
```

Forbidden audit fields:

```text
resolved value
encoded value
raw Secret object payload
```

## 9. Error Codes

Recommended normalized error codes:

```text
InvalidSecretRef
NamespaceNotAllowed
SecretNotFound
SecretKeyNotFound
EmptySecretValue
AccessDenied
ContextCanceled
BackendUnavailable
```

These should map into existing `SecretError` style errors.

## 10. Caching Boundary

Initial implementation should not cache secret values.

Reason:

```text
simpler correctness
no stale credential risk
no cache invalidation policy yet
```

Future caching may be added only with:

```text
short TTL
explicit disable switch
audit events
metrics
rotation tests
```

## 11. Provider Integration

Provider packages should not import Kubernetes packages.

Correct direction:

```text
model/openai.HTTPClient
  depends on runtime/secrets.Resolver interface

runtime/secrets/kubernetes.Resolver
  implements runtime/secrets.Resolver
```

Incorrect direction:

```text
model/openai imports client-go
model/openai reads Kubernetes Secret directly
provider config stores raw credential values
```

## 12. Test Strategy

Unit tests should cover:

```text
valid SecretRef lookup
invalid SecretRef rejection
namespace allowlist rejection
missing Secret rejection
missing key rejection
empty value rejection
context cancellation
no raw value in error message
```

Integration tests should be separate and disabled by default.

Possible integration guard:

```text
AICLOUD_K8S_SECRET_RESOLVER_INTEGRATION_TEST=1
```

## 13. Not Done Yet

```text
- client-go dependency
- Kubernetes-backed resolver implementation
- RBAC manifests
- integration test against real API server
- caching
- metrics
- audit sink integration
```

## 14. Next Step

The next engineering step is a lightweight package skeleton with an interface-compatible resolver and fake Kubernetes Secret getter.

No real cluster access should be added until the fake implementation and tests are stable.
