# PR-038 yamlio Writer Preflight

## Purpose

This note records the preflight design check for wiring `integrations/gitops/yamlio` into the GitOps dry-run writer path.

This is intentionally a preflight note only.

It does not wire yamlio into `DryRunManifestWriter` yet.

## Current Gate

PR-037 static stabilization is complete, but a real test run is still pending.

Required before implementation:

```bash
go mod tidy
go test ./...
git status --short
```

Do not claim the repository is green until that has been observed.

## Current DryRunManifestWriter Shape

Current file:

```text
integrations/gitops/manifest_writer.go
```

Verified current interface:

```go
type ManifestWriter interface {
    WriteManagedCluster(plan ManifestPatchPlan, cluster infraapi.ManagedCluster) (*WriteResult, error)
}
```

Verified current `WriteResult`:

```go
type WriteResult struct {
    SourcePath string
    OutputPath string
    Summary    string
    Updated    infraapi.ManagedCluster
    Changes    []ManifestFieldChange
    Rollback   []ManifestFieldChange
}
```

Current writer behavior:

```text
ManifestPatchPlan + infra/api.ManagedCluster
  -> ApplyManagedClusterPatch
  -> WriteResult with Updated object
```

It does not currently produce manifest bytes.

It does not read files, write files, create commits, create PRs, or call Kubernetes.

## Current BranchPlan Shape

Current file:

```text
integrations/gitops/branch_plan.go
```

Verified current `FileChangePlan`:

```go
type FileChangePlan struct {
    Path    string
    Summary string
}
```

`BuildBranchPlan` currently uses:

```text
result.OutputPath
result.Summary
plan.PR.BranchName
plan.PR.CommitMessage
plan.PR.Title
plan.PR.Draft
```

It does not carry rendered file bytes.

This means PR-038 can introduce rendered manifest bytes without changing `BranchPlan` immediately.

Future real commit creation may need bytes, but that belongs to a later GitHub integration step, not this preflight.

## yamlio Current Shape

Current package:

```text
integrations/gitops/yamlio
```

Current public functions:

```go
ReadManagedCluster(data []byte) (api.ManagedCluster, error)
WriteManagedCluster(cluster api.ManagedCluster) ([]byte, error)
```

Current implementation is dependency-free and intentionally narrow.

It supports only the current ManagedCluster YAML-shaped example.

## Main Design Question

To wire yamlio into GitOps writer behavior, there are two possible designs.

### Option A: Extend WriteResult

Add manifest bytes directly to `WriteResult`:

```go
type WriteResult struct {
    SourcePath string
    OutputPath string
    Summary    string
    Updated    infraapi.ManagedCluster
    Manifest   []byte
    Changes    []ManifestFieldChange
    Rollback   []ManifestFieldChange
}
```

Pros:

```text
simple
keeps one writer path
branch/commit planning can use result.Manifest
```

Cons:

```text
changes existing result shape
may require test updates in branch planning
couples object patching and serialization earlier
```

### Option B: Add a Separate ManifestBytesWriter

Keep current `DryRunManifestWriter` object-level only, and add a wrapper.

Recommended result type:

```go
type ManagedClusterManifestBytesResult struct {
    WriteResult *WriteResult
    Manifest    []byte
}
```

Recommended wrapper:

```go
type ManagedClusterManifestBytesWriter struct {
    ObjectWriter ManifestWriter
}
```

Recommended method:

```go
func (w *ManagedClusterManifestBytesWriter) WriteManagedClusterBytes(plan ManifestPatchPlan, input []byte) (*ManagedClusterManifestBytesResult, error)
```

Expected flow:

```text
input YAML-shaped bytes
  -> yamlio.ReadManagedCluster(input)
  -> ObjectWriter.WriteManagedCluster(plan, parsedCluster)
  -> yamlio.WriteManagedCluster(result.Updated)
  -> ManagedClusterManifestBytesResult{WriteResult: result, Manifest: renderedBytes}
```

Pros:

```text
preserves existing object-level writer behavior
keeps serialization as an explicit layer
input boundary is bytes, not an already parsed object
less risky while PR-037 test result is still pending
does not require BranchPlan changes because BranchPlan currently tracks path/summary only
```

Cons:

```text
one more type
slightly more plumbing before branch/commit integration
```

## Recommended Direction

Use Option B first.

Reason:

```text
The current architecture deliberately separates patch planning, object patching, branch planning, and live execution.
A wrapper preserves that separation and avoids changing DryRunManifestWriter behavior before go test ./... is confirmed.
The byte-oriented method should accept input bytes so yamlio is genuinely part of the read/patch/write path.
```

Additional verified reason:

```text
BuildBranchPlan currently does not need rendered file bytes.
Therefore byte rendering can remain one layer above object writing and one layer before future real commit creation.
```

## Proposed PR-038 Scope

After PR-037 test confirmation, PR-038 should:

```text
1. Add ManagedClusterManifestBytesResult.
2. Add ManagedClusterManifestBytesWriter around the existing ManifestWriter interface.
3. Keep DryRunManifestWriter object-level and in-memory.
4. Read input bytes with yamlio.ReadManagedCluster.
5. Patch by delegating to ObjectWriter.WriteManagedCluster.
6. Serialize only the Updated ManagedCluster returned by the object writer.
7. Return manifest bytes separately from the existing WriteResult.
8. Add tests proving bytes can be read back through yamlio.ReadManagedCluster.
9. Keep BranchPlan unchanged unless a future real commit-file content layer requires bytes.
```

## Error Boundaries

PR-038 should keep read, patch, and write failures distinguishable.

Recommended error handling:

```text
yamlio.ReadManagedCluster failure -> read/input error boundary
ObjectWriter.WriteManagedCluster failure -> patch/write-result error boundary
yamlio.WriteManagedCluster failure -> render/output error boundary
```

Do not collapse all failures into a generic writer error.

## Proposed Tests

```text
- valid ManagedCluster bytes are patched and rendered
- rendered bytes can be parsed back by yamlio.ReadManagedCluster
- invalid YAML-shaped bytes fail closed at read boundary
- current value mismatch still fails closed at patch boundary
- unsupported patch field still fails closed at patch boundary
- existing DryRunManifestWriter tests remain unchanged
- BranchPlan tests remain unchanged unless future commit-file content needs bytes
```

## Explicit Non-Goals

```text
- no filesystem reads
- no filesystem writes
- no real GitHub branch creation
- no real commits
- no real pull requests
- no kubectl apply
- no Kubernetes API access
- no controller-runtime integration
- no raw secret material
```

## Safety Boundary

The model continues to produce plans only.

Policy and approval remain outside yamlio.

Controller/live execution remains deferred.

GitOps writer behavior remains dry-run only.

## Temporary yamlio Limitation

The current `yamlio` implementation is a dependency-free narrow skeleton.

It only supports the current ManagedCluster example shape.

Do not treat it as a general YAML parser.

Do not expand it into a full parser inside PR-038.

If real YAML support is needed, reintroduce `gopkg.in/yaml.v3` only after Go tooling can generate and verify `go.sum`.

## Next Step

Do not implement PR-038 until PR-037 has a real test result.

The immediate next command remains:

```bash
go test ./...
```
