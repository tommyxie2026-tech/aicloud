# GitOps YAML IO Skeleton

## Goal

`integrations/gitops/yamlio` provides lightweight YAML parsing and serialization for GitOps dry-run workflows.

The current package focuses only on `ManagedCluster` object bytes.

It does not read files, write files, create branches, create pull requests, connect to Kubernetes, or apply manifests.

## Dependency Decision

The first implementation uses:

```text
gopkg.in/yaml.v3
```

Reason:

```text
small dependency surface
works without Kubernetes runtime dependencies
sufficient for provider-neutral internal objects
```

Kubernetes runtime serializers remain deferred until real CRDs and controllers exist.

## Current Functions

```go
ReadManagedCluster(data []byte) (api.ManagedCluster, error)
WriteManagedCluster(cluster api.ManagedCluster) ([]byte, error)
```

## Current Flow

```text
YAML bytes
  ↓
yaml.Unmarshal
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
yaml.Marshal
  ↓
YAML bytes
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
MarshalFailed
```

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
- go.sum has not been confirmed by go mod tidy in this workflow
- multi-document YAML
- comment preservation
- exact key ordering preservation
- file-level patching
- GitHub PR creation
- Kubernetes runtime serializer
```

## Next Step

Run `go mod tidy` and `go test ./...` in a local or CI environment, then stabilize any compile or dependency issues before wiring yamlio into DryRunManifestWriter.
