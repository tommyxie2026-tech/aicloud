# YAML Parser / Writer Dependency Decision

## 1. Goal

This document records the dependency decision for future YAML parsing and writing in the GitOps dry-run path.

The goal is to choose a safe, testable, dependency-light path before adding real manifest read/write behavior.

This PR is design-only. It does not add a YAML dependency, does not parse files, and does not write manifests.

## 2. Current State

Current GitOps implementation can produce dry-run planning objects:

```text
ManifestPatchPlan
WriteResult
BranchPlan
CommitPlan
PullRequestPlan
```

Current implementation does not yet perform:

```text
real YAML parsing
real YAML serialization
preserving comments
multi-document YAML editing
file-system writes
GitHub PR creation
kubectl apply
```

## 3. Requirements

The future YAML layer should support:

```text
parse ManagedCluster YAML
apply allowlisted spec changes
serialize updated object
produce deterministic output for tests
avoid executing changes
avoid live Kubernetes access
avoid raw credential material in generated manifests
```

Nice-to-have but not required for the first implementation:

```text
preserve comments
preserve key ordering exactly
support multi-document files
schema-aware Kubernetes decoding
```

## 4. Options Considered

### Option A: Standard Library Only

Go standard library does not include YAML support.

Pros:

```text
no dependency
minimal supply-chain surface
```

Cons:

```text
cannot parse YAML directly
would require custom parser
high bug risk
not suitable for Kubernetes manifests
```

Decision:

```text
Reject
```

## 5. Option B: gopkg.in/yaml.v3

Use `gopkg.in/yaml.v3` for basic parse and serialization.

Pros:

```text
small dependency surface
widely used
supports YAML node model
can support deterministic unit tests
works without Kubernetes dependencies
```

Cons:

```text
not Kubernetes schema-aware by itself
comment preservation is limited and must be tested
manual conversion to internal structs required
```

Decision:

```text
Preferred first implementation
```

Reason:

```text
The current project needs object-level dry-run manifest mutation first, not Kubernetes runtime decoding.
```

## 6. Option C: Kubernetes apimachinery YAML / Runtime Serializer

Use Kubernetes machinery for decoding and encoding Kubernetes-style objects.

Pros:

```text
schema-aware direction
closer to real Kubernetes resources
better long-term fit for CRD-style objects
```

Cons:

```text
larger dependency surface
introduces Kubernetes dependency earlier
requires scheme registration decisions
not needed for current provider-neutral objects
```

Decision:

```text
Defer
```

This can be revisited when real CRDs and controller-runtime integration are introduced.

## 7. Decision

Use a two-step strategy:

```text
Step 1: Add lightweight YAML reader/writer using gopkg.in/yaml.v3.
Step 2: Keep Kubernetes runtime serializer deferred until real CRDs/controllers exist.
```

Initial package target:

```text
integrations/gitops/yamlio
```

The package should remain independent of:

```text
controller-runtime
client-go
Kubernetes apiserver
GitHub API
```

## 8. Proposed Package Layout

```text
integrations/gitops/yamlio/
  README.md
  managedcluster.go
  managedcluster_test.go
```

Initial public functions:

```go
ReadManagedCluster(data []byte) (api.ManagedCluster, error)
WriteManagedCluster(cluster api.ManagedCluster) ([]byte, error)
```

Future optional functions:

```go
PatchManagedCluster(data []byte, plan ManifestPatchPlan) ([]byte, error)
```

## 9. Safety Boundary

The YAML layer must not:

```text
execute kubectl
connect to Kubernetes
create GitHub branches
open pull requests
read provider credentials
write raw secret values
mutate fields outside allowlist
```

The YAML layer only transforms bytes into internal objects and internal objects into bytes.

## 10. Test Strategy

Initial tests should verify:

```text
valid ManagedCluster YAML parses
invalid YAML fails closed
missing required fields fail validation
serialized YAML can be parsed back
replicas change is deterministic
unknown high-risk fields are not mutated by writer helpers
```

Golden tests may be added later after output format stabilizes.

## 11. Dependency Policy

Before adding `gopkg.in/yaml.v3`, update:

```text
go.mod
go.sum
```

Do not add Kubernetes dependencies for YAML I/O in the first implementation.

Do not add transitive controller-runtime dependencies for YAML I/O.

## 12. Not Done Yet

```text
- add gopkg.in/yaml.v3
- implement yamlio package
- parse real files
- write real files
- preserve comments
- support multi-document YAML
- integrate with GitHub PR creation
```

## 13. Next Step

The next engineering step is to add `integrations/gitops/yamlio` with `gopkg.in/yaml.v3`, focused only on `ManagedCluster` parse/write round-trip tests.
