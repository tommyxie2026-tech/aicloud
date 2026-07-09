# PR-038 yamlio Wiring Plan

## Status

```text
Planned only
Do not implement until PR-037 has a real successful go test ./... run
```

## Context

PR-037 statically stabilized the repository after adding infra mapping packages and the initial yamlio skeleton.

Current PR-037 state:

```text
static stabilization complete
real test run pending
```

Because a real `go test ./...` result has not been observed yet, PR-038 must remain a design/preparation step only.

## Goal

Wire `integrations/gitops/yamlio` into the GitOps dry-run path so the system can transform ManagedCluster YAML-shaped bytes into an updated ManagedCluster manifest without touching live infrastructure.

## Non-Goals

PR-038 must not add:

```text
real filesystem writes
real GitHub branch creation
real GitHub commit creation
real pull request creation
kubectl apply
Kubernetes API access
controller-runtime integration
client-go integration
raw secret values
provider credential resolution
```

## Required Precondition

Before implementation:

```bash
go mod tidy
go test ./...
git status --short
```

Required result:

```text
go test ./... must pass in a real local or CI run
```

If `git status --short` shows changed module files, commit only Go-tool-generated metadata.

Do not hand-write `go.sum`.

## Current yamlio Boundary

Current yamlio is a dependency-free narrow skeleton:

```text
ReadManagedCluster(data []byte) (api.ManagedCluster, error)
WriteManagedCluster(cluster api.ManagedCluster) ([]byte, error)
```

It supports only the current ManagedCluster example shape.

It is not a general YAML parser.

## Intended Integration Shape

Add a byte-oriented dry-run helper rather than changing live execution behavior first.

Candidate function:

```go
func (w *DryRunManifestWriter) WriteManagedClusterBytes(plan ManifestPatchPlan, data []byte) (*ManifestWriteBytesResult, error)
```

Candidate result:

```go
type ManifestWriteBytesResult struct {
    SourcePath string
    OutputPath string
    Original   []byte
    Updated    []byte
    Changes    []ManifestFieldChange
    Rollback   []ManifestFieldChange
    Summary    string
}
```

Flow:

```text
input YAML-shaped bytes
  ↓
yamlio.ReadManagedCluster
  ↓
ApplyManagedClusterPatch
  ↓
yamlio.WriteManagedCluster
  ↓
return updated bytes in dry-run result
```

## Error Handling

Errors should preserve existing fail-closed behavior.

Suggested error code mapping:

```text
yamlio.EmptyInput              -> InvalidManifestInput
yamlio.InvalidYAML             -> InvalidManifestInput
yamlio.InvalidManagedCluster   -> InvalidManifestObject
ApplyManagedClusterPatch error -> existing GitOpsError passthrough
```

Do not swallow yamlio errors.

Do not create partial output on parse or validation failure.

## Tests To Add

```text
valid ManagedCluster YAML bytes are patched from replicas 3 to 6
invalid YAML bytes fail closed
valid YAML with target mismatch fails closed
negative replicas fail closed
updated bytes can be read back by yamlio
source/output path metadata is preserved
no filesystem writes occur
```

## Reintroducing gopkg.in/yaml.v3

Do not reintroduce `gopkg.in/yaml.v3` in the same step as wiring unless PR-037 is already green.

Preferred order:

```text
1. Confirm PR-037 go test ./... green with dependency-free yamlio.
2. Wire dependency-free yamlio into dry-run byte helper.
3. Confirm go test ./... again.
4. In a later PR, reintroduce gopkg.in/yaml.v3 with Go-generated go.sum.
```

## Done Definition

PR-038 is done only when:

```text
- byte-oriented yamlio wiring exists
- no live side effects are introduced
- tests cover success and fail-closed paths
- go test ./... is observed passing
- documentation says this is still dry-run only
```
