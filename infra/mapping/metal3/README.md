# Metal3 Mapping Skeleton

## Goal

`infra/mapping/metal3` defines a provider-neutral mapping boundary from `aicloud` infrastructure intent to Metal3-style bare metal host desired state.

This package intentionally does not import Metal3 APIs, controller-runtime, or provider-specific bare metal APIs yet.

## Current Input

```text
infra/api.ManagedCluster
infra/api.MachineClass[]
```

Important fields:

```text
ManagedCluster.metadata.name
ManagedCluster.metadata.namespace
ManagedCluster.spec.environment
ManagedCluster.spec.workers[].name
ManagedCluster.spec.workers[].replicas
ManagedCluster.spec.workers[].machineClassRef.name
MachineClass.metadata.name
MachineClass.spec.cpu
MachineClass.spec.memory
MachineClass.spec.gpu
MachineClass.spec.storage
```

## Current Output

```text
DesiredBareMetalHostClaim[]
```

The output is provider-neutral and safe for unit testing.

## Mapping Flow

```text
ManagedCluster + MachineClass[]
  ↓
Mapper.MapManagedCluster
  ↓
MappingResult
  ↓
DesiredBareMetalHostClaim[]
```

## Replica Expansion

Each worker group replica expands into one desired host claim identity.

Example:

```text
ManagedCluster dev-gpu-cluster
worker group gpu-workers
replicas 3
```

Maps to:

```text
dev-gpu-cluster-gpu-workers-host-0001
dev-gpu-cluster-gpu-workers-host-0002
dev-gpu-cluster-gpu-workers-host-0003
```

## MachineClass Mapping

Each worker group must reference a MachineClass.

```text
workers[].machineClassRef.name
  ↓
MachineClass.metadata.name
  ↓
DesiredBareMetalHostClaim CPU / Memory / GPUProfile / StorageProfile
```

Missing MachineClass fails closed.

## Labels

Generated desired host claims include labels:

```text
aicloud.dev/managed-by: aicloud
aicloud.dev/managedcluster-name: <cluster-name>
aicloud.dev/environment: <environment>
aicloud.dev/worker-group: <worker-group-name>
aicloud.dev/machine-class: <machineclass-name>
```

## Fail-Closed Behavior

The mapper rejects invalid ManagedCluster input before producing output.

It also rejects worker groups that reference unknown MachineClass names.

## Not Done Yet

```text
- real Metal3 API types
- real BareMetalHost YAML generation
- live host lifecycle operation
- hardware inventory discovery
- management endpoint integration
```

## Next Step

A future implementation can convert `DesiredBareMetalHostClaim` into real Metal3 manifests only after dependency, RBAC and provider choices are finalized.
