# PR-037 GitOps Test Helper Audit

## Goal

This note records the static audit of GitOps test helpers during PR-037 stabilization.

The goal is to identify hidden compile risks around helper functions such as:

```text
validManagedCluster
validPatchPlan
validEvaluatedProposal
```

## Confirmed Files Checked

### integrations/gitops/patch_plan_test.go

Status:

```text
checked
```

Findings:

```text
- validEvaluatedProposal is defined in this file.
- It does not depend on infra/api.ManagedCluster.
- It builds agent/proposal.ChangeProposal directly.
- The previous local variable shadowing issue was fixed by using changeProposal instead of proposal.
```

API risk:

```text
low
```

### integrations/gitops/manifest_writer_test.go

Status:

```text
checked
```

Findings:

```text
- This file calls validManagedCluster(3).
- This file calls validPatchPlan(3, 6).
- The helper definitions are not in this file.
```

API risk:

```text
unknown until helper definitions are found or go test ./... is run
```

### integrations/gitops/branch_plan_test.go

Status:

```text
checked
```

Findings:

```text
- This file also calls validManagedCluster(3).
- This file also calls validPatchPlan(3, 6).
- The helper definitions are not in this file.
```

API risk:

```text
unknown until helper definitions are found or go test ./... is run
```

## Files Attempted But Not Found

```text
integrations/gitops/object_patch_test.go
integrations/gitops/patch_writer_test.go
integrations/gitops/patch_test.go
integrations/gitops/manifest_patch_test.go
integrations/gitops/manifest_patch_plan_test.go
integrations/gitops/patch_planner_test.go
integrations/gitops/test_helpers_test.go
integrations/gitops/helpers_test.go
```

## Search Limitation

The GitHub code search connector returned no results for several queries that should normally match existing files:

```text
validManagedCluster(
validPatchPlan(
NewManagedCluster
NewMachineClass
package gitops
Replicas
```

Interpretation:

```text
Do not rely on connector code search as complete for this repository during PR-037.
```

## Current Risk

The main remaining static uncertainty is whether `validManagedCluster` still uses the current API shape:

```text
api.NewManagedCluster(name, namespace, environment)
cluster.Name / cluster.Namespace
WorkerGroupSpec.Replicas int32
```

If the helper still uses an older shape such as:

```text
api.NewManagedCluster(name, namespace)
cluster.Metadata.Name
cluster.Metadata.Namespace
```

then `go test ./...` will fail to compile.

## Required Verification

Run:

```bash
go test ./integrations/gitops/...
go test ./...
```

If compile fails, first inspect the file defining:

```text
validManagedCluster
validPatchPlan
```

and update the helper to the current `infra/api` shape.

## Boundary

This audit does not prove tests pass.

It only records static findings from the files that were accessible through the connector.
