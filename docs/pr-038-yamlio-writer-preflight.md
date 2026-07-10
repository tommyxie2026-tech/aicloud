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

Current interface:

```go
type ManifestWriter interface {
    WriteManagedCluster(plan ManifestPatchPlan, cluster infraapi.ManagedCluster) (*WriteResult, error)
}
```

Current `WriteResult`:

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

Keep current `DryRunManifestWriter` object-level only, and add a wrapper:

```go
type ManagedClusterManifestBytesWriter struct {
    ObjectWriter ManifestWriter
}
```

Preferred method:

```go
WriteManagedClusterBytes(plan ManifestPatchPlan, input []byte) (*WriteResult, []byte, error)
```

Expected internal flow:

```text
input bytes
  -> yamlio.ReadManagedCluster
  -> ObjectWriter.WriteManagedCluster
  -> yamlio.WriteManagedCluster(result.Updated)
  -> return original WriteResult plus rendered bytes
```

Pros:

```text
preserves existing object-level writer behavior
keeps serialization as an explicit layer
lets branch/commit integration consume bytes later without mutating WriteResult now
less risky while PR-037 test result is still pending
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
```

The wrapper should accept bytes, not an already parsed object.

Reason:

```text
The purpose of PR-038 is to exercise yamlio in the writer path.
If the wrapper accepts an object, yamlio.ReadManagedCluster is not covered by the main PR-038 flow.
```

## Proposed PR-038 Scope

After PR-037 test confirmation, PR-038 should:

```text
1. Add a yamlio-backed wrapper around DryRunManifestWriter.
2. Keep DryRunManifestWriter object-level and in-memory.
3. Parse input bytes with yamlio.ReadManagedCluster.
4. Patch through the existing object-level ManifestWriter.
5. Serialize only the Updated ManagedCluster returned by the object writer.
6. Return manifest bytes separately from WriteResult.
7. Add tests proving bytes can be read back through yamlio.ReadManagedCluster.
```

## Proposed Tests

```text
- valid ManagedCluster bytes are patched and rendered
- rendered bytes can be parsed back by yamlio.ReadManagedCluster
- invalid YAML-shaped bytes fail closed
- current value mismatch still fails closed
- unsupported patch field still fails closed
- existing DryRunManifestWriter tests remain unchanged
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

## Next Step

Do not implement PR-038 until PR-037 has a real test result.

The immediate next command remains:

```bash
go test ./...
```
