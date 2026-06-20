# Cluster API Mapping Skeleton

## Goal

`infra/mapping/clusterapi` defines a provider-neutral mapping boundary from `aicloud` infrastructure intent to Cluster API-like desired state.

This package intentionally does not import Cluster API, controller-runtime, or provider-specific APIs yet.

## Current Input

```text
infra/api.ManagedCluster
```

Important fields:

```text
metadata.name
metadata.namespace
spec.environment
spec.workers[].name
spec.workers[].replicas
spec.workers[].machineClassRef.name
```

## Current Output

```text
DesiredCluster
DesiredMachineDeployment[]
```

The output is provider-neutral and safe for unit testing.

## Mapping Flow

```text
ManagedCluster
  ↓
Mapper.MapManagedCluster
  ↓
MappingResult
  ↓
DesiredCluster
  ↓
DesiredMachineDeployment[]
```

## Worker Mapping

Each `ManagedCluster.spec.workers[]` entry maps to one desired machine deployment shape.

```text
workers[].name
  -> DesiredMachineDeployment.WorkerGroupName
  -> DesiredMachineDeployment.Name suffix

workers[].replicas
  -> DesiredMachineDeployment.Replicas

workers[].machineClassRef.name
  -> DesiredMachineDeployment.MachineClassName
```

## Naming

Machine deployment names are deterministic:

```text
<managedcluster-name>-<worker-group-name>
```

The mapper sanitizes `_`, `.`, and `/` into `-` for generated names and paths.

## Labels

Generated desired state includes labels:

```text
aicloud.dev/managed-by: aicloud
aicloud.dev/managedcluster-name: <cluster-name>
aicloud.dev/environment: <environment>
aicloud.dev/worker-group: <worker-group-name>
```

## GitOps Path Helper

`MachineDeploymentPatchPath` produces a deterministic path:

```text
clusters/<cluster-name>/machinedeployments/<cluster-name>-<worker-group-name>.yaml
```

This is only a planning helper. It does not write files.

## Fail-Closed Behavior

The mapper calls `api.ValidateManagedCluster` before producing output.

Invalid source objects fail closed with:

```text
MappingError{Code: InvalidManagedCluster}
```

## Not Done Yet

```text
- real Cluster API types
- real MachineDeployment YAML generation
- provider-specific templates
- controller-runtime reconciliation
- clusterctl integration
- live apply
```

## Next Step

A future package can map `DesiredMachineDeployment` to real Cluster API YAML after dependency and provider choices are finalized.
