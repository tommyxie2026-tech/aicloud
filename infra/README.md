# Infrastructure Layer

## 1. Goal

The `infra/` directory contains the first infrastructure control-plane scenario for `aicloud`.

Its purpose is to explore:

```text
How Kubernetes-style APIs can manage clusters, virtual machines, and eventually physical machines under AI-assisted planning and deterministic governance.
```

This layer should not redefine `aicloud` as only a Kubernetes controller.

`aicloud` remains:

```text
Hybrid Private AI Cloud Platform + AI-native Infrastructure Control Plane
```

Infrastructure control is the first high-value scenario.

## 2. Core Principle

```text
Models propose.
Policy decides.
Humans approve when required.
Controllers execute.
```

The infrastructure layer receives approved desired state.

It must not directly trust model output.

## 3. Current Packages

```text
infra/api
infra/controller
```

## 4. api

Path:

```text
infra/api/types.go
infra/api/validation.go
infra/api/*_test.go
```

Purpose:

```text
Define Kubernetes-style infrastructure API types and static validation.
```

Current group/version:

```text
infra.aicloud.dev/v1alpha1
```

Current kinds:

```text
ManagedCluster
MachineClass
```

Current type concepts:

```text
TypeMeta
ObjectMeta
OwnerReference
LocalObjectReference
ManagedClusterSpec
ManagedClusterStatus
WorkerGroupSpec
MachineClassSpec
Condition
```

## 5. ManagedCluster

`ManagedCluster` is the first desired-state API.

It represents a cluster-like infrastructure unit that may later map to:

```text
Cluster API Cluster / MachineDeployment
KubeVirt VM-backed cluster
Metal3 bare-metal cluster
Crossplane composite resource
Internal cloud platform cluster
```

Current MVP field of interest:

```text
spec.workers[name=gpu-workers].replicas
```

First scenario:

```text
3 -> 6 replicas
```

## 6. MachineClass

`MachineClass` defines reusable machine profiles.

It may later map to:

```text
Cluster API machine template
KubeVirt VM flavor / template
Metal3 hardware profile
Cloud instance type
Internal platform machine pool
```

Current example:

```text
gpu-large
32 CPU
128Gi memory
4 x A100
```

## 7. Validation

Current static validation rules:

```text
ManagedCluster:
- kind must be ManagedCluster
- metadata.name is required
- spec.environment is required
- worker group name is required
- worker group name must be unique
- replicas must be >= 0
- machineClassRef.name is required

MachineClass:
- kind must be MachineClass
- metadata.name is required
- spec.provider is required
- gpu.count must be >= 0
- gpu.type is required when gpu.count > 0
```

## 8. controller

Path:

```text
infra/controller/README.md
infra/controller/fake.go
infra/controller/fake_test.go
```

Purpose:

```text
Provide a deterministic fake reconciliation loop for the first ManagedCluster scenario.
```

The fake controller has no external side effects.

It does not:

```text
- install CRDs
- connect to Kubernetes
- call Cluster API
- call KubeVirt
- call Metal3
- handle real credentials
- perform destructive operations
```

## 9. Fake Reconcile Flow

```text
ManagedCluster input
  ↓
ValidateManagedCluster
  ↓
Read desired worker replicas from spec
  ↓
Read observed workerReadyReplicas from status
  ↓
Move workerReadyReplicas one step toward desired replicas
  ↓
Set Phase
  ↓
Set Ready / Reconciling conditions
  ↓
Update observedGeneration when ready
  ↓
Persist in FakeStateStore
```

## 10. Current Example Manifests

```text
examples/infra/managedcluster-dev-gpu.yaml
examples/infra/machineclass-gpu-large.yaml
```

These examples correspond to the first scenario:

```text
scale dev-gpu-cluster gpu-workers from 3 to 6
```

## 11. Community Project Mapping

`aicloud` should not replace mature infrastructure projects.

It should sit above them as:

```text
AI-native planning, governance, policy, workflow, and audit layer.
```

Mapping:

```text
Cluster API:
  ManagedCluster -> Cluster
  WorkerGroupSpec -> MachineDeployment
  MachineClass -> MachineTemplate

KubeVirt:
  VirtualMachine -> KubeVirt VirtualMachine
  MachineClass -> VM flavor / template

Metal3:
  BareMetalHost -> Metal3 BareMetalHost
  MachineClass -> hardware profile

Crossplane:
  ManagedCluster -> CompositeResource
  ProviderRef -> ProviderConfig
```

## 12. Safety Boundary

Forbidden model-generated mutations:

```text
status
metadata.finalizers
metadata.ownerReferences
spec.credentials
spec.secretRef
spec.bmcSecretRef
```

Execution boundary:

```text
Model output -> ChangePlan -> Policy -> PR/GitOps -> Controller
```

Not allowed:

```text
Model output -> direct infrastructure execution
```

## 13. Current Tests

```text
infra/api/types_test.go
infra/api/validation_test.go
infra/controller/fake_test.go
```

Current tested behavior:

```text
- ManagedCluster type construction
- MachineClass type construction
- API static validation
- fake reconcile already-ready state
- fake reconcile scale-out one step
- fake reconcile scale-out until ready
- fake reconcile scale-down one step
- invalid object rejection
- FakeStateStore get/set
```

## 14. Not Done Yet

```text
- real CRD generation
- controller-runtime manager
- Kubernetes client integration
- status subresource
- admission validation
- Cluster API adapter
- KubeVirt adapter
- Metal3 adapter
- Crossplane adapter
- GitOps manifest generator
```

## 15. Recommended Next Steps

Recommended next steps:

```text
1. Add infra/adapter design interfaces.
2. Add FakeClusterAdapter boundary.
3. Add GitOps manifest generation design.
4. Extend ChangeProposal -> ManagedCluster patch mapping.
5. Add controller-runtime design only after fake controller flow is stable.
```
