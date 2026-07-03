# go test Stabilization Notes

## Goal

This note tracks the `PR-037 go test ./... stabilization` work.

The current workflow has not executed `go test ./...` or `go mod tidy`.

This document records static compile-risk fixes made before a real test run.

## Current Status

```text
Status: static stabilization in progress
Real go test ./... run: not confirmed
Real go mod tidy run: not confirmed
go.sum: not present yet
```

## Why This Was Needed

Several recently added mapping packages were written against an assumed `infra/api` shape.

The real API shape is:

```go
ManagedCluster embeds ObjectMeta
cluster.Name
cluster.Namespace

NewManagedCluster(name, namespace, environment)

WorkerGroupSpec.Replicas int32

MachineClass embeds ObjectMeta
class.Name
class.Spec.GPU *GPUSpec
```

There is no:

```text
cluster.Metadata
class.Metadata
class.Spec.Storage
```

## Static Fixes Completed

### Cluster API mapper

Updated files:

```text
infra/mapping/clusterapi/types.go
infra/mapping/clusterapi/mapper.go
infra/mapping/clusterapi/mapper_test.go
```

Fixes:

```text
- DesiredMachineDeployment.Replicas changed to int32
- cluster.Metadata usage replaced with cluster.Name / cluster.Namespace
- NewManagedCluster test helper updated to pass environment
```

### KubeVirt mapper

Updated files:

```text
infra/mapping/kubevirt/types.go
infra/mapping/kubevirt/mapper.go
infra/mapping/kubevirt/mapper_test.go
```

Fixes:

```text
- DesiredVirtualMachine.Ordinal changed to int32
- removed StorageProfile because MachineClassSpec has no Storage field
- class.Metadata usage replaced with class.Name
- class.Spec.GPU string usage replaced with *api.GPUSpec helper
- NewManagedCluster test helper updated to pass environment
```

### Metal3 mapper

Updated files:

```text
infra/mapping/metal3/types.go
infra/mapping/metal3/mapper.go
infra/mapping/metal3/mapper_test.go
```

Fixes:

```text
- DesiredBareMetalHostClaim.Ordinal changed to int32
- removed StorageProfile because MachineClassSpec has no Storage field
- class.Metadata usage replaced with class.Name
- class.Spec.GPU string usage replaced with *api.GPUSpec helper
- NewManagedCluster test helper updated to pass environment
```

### yamlio

Updated files:

```text
integrations/gitops/yamlio/managedcluster.go
integrations/gitops/yamlio/managedcluster_test.go
```

Fixes:

```text
- cluster.Metadata usage replaced with cluster.Name / cluster.Namespace / cluster.Labels
- NewManagedCluster call updated to include environment
- WorkerGroupYAML.Replicas changed to int32
```

### GitOps patch plan tests

Updated file:

```text
integrations/gitops/patch_plan_test.go
```

Fix:

```text
- renamed local variable proposal to changeProposal to avoid shadowing the imported proposal package
```

## Known Remaining Work

```text
1. Run go mod tidy.
2. Commit generated go.sum if needed.
3. Run go test ./....
4. Fix any remaining compile errors.
5. Only after tests stabilize, wire yamlio into DryRunManifestWriter.
```

## Important Boundary

Do not claim `go test ./...` passes until a real test run has been observed.

Do not hand-write go.sum hashes unless they are produced by Go tooling.
