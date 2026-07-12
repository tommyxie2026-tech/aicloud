# PR-038 Implementation Checklist

## Scope

PR-038 is intended to wire `integrations/gitops/yamlio` into a dry-run manifest bytes path without changing the existing object-level writer behavior.

This checklist depends on:

```text
PR-037 static stabilization complete
PR-037 real go test ./... result still pending
```

## Hard Gate

Do not implement PR-038 code until the following has been observed in a real environment:

```bash
go mod tidy
go test ./...
git status --short
```

A missing visible CI status is not enough.

A connector response with empty status arrays is not enough.

## Architecture Rule

Keep these layers separate:

```text
Patch planning
Object patching
YAML-shaped byte rendering
Branch/commit/PR planning
Live GitHub/Kubernetes execution
```

PR-038 may touch only:

```text
YAML-shaped byte rendering
Object patching delegation
Unit tests for the dry-run wrapper
Documentation
```

## Recommended Code Additions

Add a new wrapper type rather than modifying `DryRunManifestWriter`.

```go
type ManagedClusterManifestBytesWriter struct {
    ObjectWriter ManifestWriter
}
```

Add a new result type rather than changing `WriteResult`.

```go
type ManagedClusterManifestBytesResult struct {
    WriteResult *WriteResult
    Manifest    []byte
}
```

Add a byte-oriented method:

```go
func (w *ManagedClusterManifestBytesWriter) WriteManagedClusterBytes(plan ManifestPatchPlan, input []byte) (*ManagedClusterManifestBytesResult, error)
```

## Expected Flow

```text
input []byte
  -> yamlio.ReadManagedCluster(input)
  -> ObjectWriter.WriteManagedCluster(plan, parsedCluster)
  -> yamlio.WriteManagedCluster(writeResult.Updated)
  -> ManagedClusterManifestBytesResult
```

## Do Not Change

Do not change these existing shapes in PR-038:

```text
ManifestWriter
DryRunManifestWriter
WriteResult
BranchPlan
CommitPlan
FileChangePlan
PullRequestPlan
```

Do not make `DryRunManifestWriter` import `yamlio`.

Do not make `BranchPlan` carry rendered bytes in PR-038.

## Error Boundaries

Keep these failure boundaries distinguishable:

```text
Read boundary: yamlio.ReadManagedCluster failed
Patch boundary: ObjectWriter.WriteManagedCluster failed
Render boundary: yamlio.WriteManagedCluster failed
```

A caller should be able to tell which stage failed.

## Tests Required

Add tests for:

```text
valid input bytes are parsed, patched, rendered, and returned
rendered bytes can be parsed by yamlio.ReadManagedCluster
patched replica count is visible in parsed rendered output
invalid input bytes fail before patching
current value mismatch fails through the existing object writer
unsupported field fails through the existing object writer
existing DryRunManifestWriter tests remain unchanged
existing BranchPlan tests remain unchanged
```

## Explicit Non-Goals

PR-038 must not introduce:

```text
filesystem reads
filesystem writes
real GitHub branches
real GitHub commits
real GitHub pull requests
kubectl apply
Kubernetes API calls
controller-runtime
provider credential resolution
raw secret handling
```

## yamlio Limitation

Current `yamlio` is a dependency-free narrow skeleton.

Do not expand it into a general YAML parser in PR-038.

If real YAML support is needed later, reintroduce `gopkg.in/yaml.v3` only after Go tooling can generate and verify `go.sum`.

## Done Criteria

PR-038 is done only when:

```text
existing object-level writer behavior is unchanged
new bytes wrapper has unit tests
rendered bytes round-trip through yamlio.ReadManagedCluster
no live side effects are introduced
go test ./... has passed in a real run
```
