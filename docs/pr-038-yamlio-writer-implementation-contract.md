# PR-038 yamlio Writer Implementation Contract

## Contract

PR-038 must implement the byte-oriented GitOps writer path as a thin wrapper around the existing object-oriented writer.

Canonical flow:

```text
input bytes
  -> yamlio.ReadManagedCluster
  -> existing ManifestWriter.WriteManagedCluster
  -> yamlio.WriteManagedCluster(result.Updated)
  -> return WriteResult + Manifest bytes
```

## Required Dependency Direction

```text
ManagedClusterManifestBytesWriter
  -> yamlio
  -> ManifestWriter interface

DryRunManifestWriter
  -> no yamlio dependency
```

`DryRunManifestWriter` must remain object-level and in-memory.

## Recommended Types

```go
type ManagedClusterManifestBytesResult struct {
    WriteResult *WriteResult
    Manifest    []byte
}

type ManagedClusterManifestBytesWriter struct {
    ObjectWriter ManifestWriter
}
```

Recommended constructor:

```go
func NewManagedClusterManifestBytesWriter(objectWriter ManifestWriter) *ManagedClusterManifestBytesWriter
```

Recommended method:

```go
func (w *ManagedClusterManifestBytesWriter) WriteManagedClusterBytes(plan ManifestPatchPlan, input []byte) (*ManagedClusterManifestBytesResult, error)
```

## Required Behavior

The wrapper must:

```text
1. reject a nil ObjectWriter
2. parse input bytes using yamlio.ReadManagedCluster
3. delegate patching to ObjectWriter.WriteManagedCluster
4. render only result.Updated using yamlio.WriteManagedCluster
5. return both the object-level WriteResult and rendered Manifest bytes
```

## Explicit Non-Goals

PR-038 must not:

```text
modify DryRunManifestWriter
modify WriteResult
modify BranchPlan
modify CommitPlan
modify PullRequestPlan
read files from disk
write files to disk
create commits
create pull requests
call Kubernetes
call provider APIs
handle raw secret values
```

## Error Boundaries

Failures must remain distinguishable by stage:

```text
yamlio.ReadManagedCluster failure -> input/read boundary
ObjectWriter.WriteManagedCluster failure -> patch boundary
yamlio.WriteManagedCluster failure -> render boundary
nil ObjectWriter -> configuration boundary
```

## Required Tests

PR-038 should add tests for:

```text
valid input bytes are parsed, patched, and rendered
rendered manifest bytes can be read back by yamlio.ReadManagedCluster
patched output has the expected worker replica count
invalid input bytes fail before patching
current value mismatch fails through the object writer
nil object writer fails closed
existing DryRunManifestWriter tests remain unchanged
existing BranchPlan tests remain unchanged
```

## Gate

Do not implement this contract until PR-037 has a real test result:

```bash
go mod tidy
go test ./...
git status --short
```

Do not claim PR-038 is complete unless `go test ./...` has been observed to pass after implementation.
