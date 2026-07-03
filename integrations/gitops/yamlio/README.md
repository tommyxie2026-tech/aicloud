# GitOps YAML IO Skeleton

## Goal

`integrations/gitops/yamlio` provides lightweight YAML-shaped parsing and serialization for GitOps dry-run workflows.

The current package focuses only on `ManagedCluster` object bytes.

It does not read files, write files, create branches, create pull requests, connect to Kubernetes, or apply manifests.

## Current Dependency Decision

The original preferred implementation was `gopkg.in/yaml.v3`.

During PR-037 stabilization, that dependency was deferred because `go.sum` could not be generated or verified in the current workflow.

Current implementation is dependency-free and intentionally narrow:

```text
no gopkg.in/yaml.v3
no Kubernetes runtime serializer
no client-go
no controller-runtime
```

This is a temporary skeleton parser/writer, not a general YAML implementation.

## Supported YAML Shape

The parser only supports the current `ManagedCluster` example shape:

```text
apiVersion
kind
metadata.name
metadata.namespace
metadata.labels
spec.environment
spec.workers[].name
spec.workers[].replicas
spec.workers[].machineClassRef.name
```

Unsupported YAML features include:

```text
anchors
aliases
multi-document YAML
inline maps
complex sequences outside workers
comment preservation
exact input ordering preservation
```

## Current Functions

```go
ReadManagedCluster(data []byte) (api.ManagedCluster, error)
WriteManagedCluster(cluster api.ManagedCluster) ([]byte, error)
```

## Current Flow

```text
YAML-shaped bytes
  ↓
narrow ManagedCluster parser
  ↓
ManagedClusterYAML DTO
  ↓
infra/api.ManagedCluster
  ↓
api.ValidateManagedCluster
```

Write flow:

```text
infra/api.ManagedCluster
  ↓
api.ValidateManagedCluster
  ↓
ManagedClusterYAML DTO
  ↓
deterministic formatter
  ↓
YAML-shaped bytes
```

## Current Safety Boundary

```text
- no filesystem reads
- no filesystem writes
- no GitHub API calls
- no Kubernetes API calls
- no kubectl apply
- no secret resolution
- no raw credential material
```

## Error Codes

```text
EmptyInput
InvalidYAML
InvalidManagedCluster
```

`MarshalFailed` is currently unused because the dependency-free formatter does not return an error.

## Current Tests

```text
valid ManagedCluster YAML parses
empty input fails closed
invalid YAML fails closed
invalid ManagedCluster fails closed
write/read round trip works
invalid ManagedCluster write fails closed
```

## Not Done Yet

```text
- real YAML library integration
- go.sum generation for YAML dependency
- multi-document YAML
- comment preservation
- exact key ordering preservation
- file-level patching
- GitHub PR creation
- Kubernetes runtime serializer
```

## Next Step

Run `go test ./...` in a local or CI environment.

Only reintroduce `gopkg.in/yaml.v3` after `go mod tidy` and `go.sum` can be generated and committed by Go tooling.
