# KubeVirt Mapping Skeleton

## Goal

`infra/mapping/kubevirt` defines a provider-neutral mapping boundary from `aicloud` infrastructure intent to KubeVirt-style virtual machine desired state.

This package intentionally does not import KubeVirt APIs, controller-runtime, or provider-specific VM APIs yet.

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
DesiredVirtualMachine[]
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
DesiredVirtualMachine[]
```

## Replica Expansion

Each worker group replica expands into one desired VM identity.

Example:

```text
ManagedCluster dev-gpu-cluster
worker group gpu-workers
replicas 3
```

Maps to:

```text
dev-gpu-cluster-gpu-workers-0001
dev-gpu-cluster-gpu-workers-0002
dev-gpu-cluster-gpu-workers-0003
```

## MachineClass Mapping

Each worker group must reference a MachineClass.

```text
workers[].machineClassRef.name
  ↓
MachineClass.metadata.name
  ↓
DesiredVirtualMachine CPU / Memory / GPUProfile / StorageProfile
```

Missing MachineClass fails closed.

## Labels

Generated desired VMs include labels:

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
- real KubeVirt API types
- real VirtualMachine YAML generation
- live VM lifecycle operation
- node/device discovery
- provider-specific GPU passthrough implementation
```

## Next Step

A future implementation can convert `DesiredVirtualMachine` into real KubeVirt manifests only after dependency, RBAC and provider choices are finalized.
