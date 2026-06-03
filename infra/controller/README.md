# Infra Controller Design

## 1. Purpose

This directory will contain the infrastructure reconciliation layer for `aicloud`.

The first implementation should be a fake controller, not a production Kubernetes controller.

The goal is to prove the control-loop semantics before introducing controller-runtime, CRDs, webhooks, or real infrastructure adapters.

## 2. Design Principle

```text
Spec is desired state.
Status is observed state.
Controller reconciles actual state toward desired state.
Model only proposes.
Policy decides.
GitOps / controller executes.
```

The controller must not trust model output directly.

It should only observe already-approved desired state.

## 3. Initial Scope

Initial fake controller scope:

```text
ManagedCluster
MachineClass
```

Initial scenario:

```text
ManagedCluster dev-gpu-cluster gpu-workers replicas 3 -> 6
```

The fake controller should simulate:

```text
- reading ManagedCluster.spec
- comparing desired replicas with observed ready replicas
- setting Reconciling condition
- updating workerReadyReplicas toward desired replicas
- setting Ready condition when complete
- updating observedGeneration
```

## 4. Non-goals

Do not implement these in the first fake controller:

```text
- real Kubernetes CRD installation
- controller-runtime manager
- admission webhook
- real Cluster API calls
- real KubeVirt calls
- real Metal3/BMC operations
- destructive actions
- credential handling
- production reconciliation
```

## 5. Reconcile Flow

```text
Input: ManagedCluster object
  ↓
Validate ManagedCluster
  ↓
Read desired worker replicas from spec
  ↓
Read observed workerReadyReplicas from status
  ↓
If observedGeneration < metadata.generation:
    mark Reconciling=True
  ↓
If ready replicas < desired replicas:
    increment ready replicas in fake state
    phase=Reconciling
    Ready=False
  ↓
If ready replicas == desired replicas:
    phase=Running
    Ready=True
    Reconciling=False
    observedGeneration=metadata.generation
  ↓
Return updated status
```

## 6. Status Update Rules

The controller owns status fields:

```text
status.phase
status.workerReadyReplicas
status.conditions
status.observedGeneration
```

The controller must not mutate:

```text
spec
metadata.name
metadata.namespace
metadata.finalizers from model input
metadata.ownerReferences from model input
```

## 7. Condition Strategy

Recommended condition types:

```text
Ready
Reconciling
Degraded
PolicyBlocked
ApprovalPending
Validated
RollbackAvailable
```

MVP fake controller should update only:

```text
Ready
Reconciling
Degraded
```

Example while scaling:

```yaml
conditions:
  - type: Ready
    status: "False"
    observedGeneration: 2
    reason: ScalingWorkers
    message: gpu-workers is scaling toward desired replicas
  - type: Reconciling
    status: "True"
    observedGeneration: 2
    reason: DesiredStateChanged
    message: observed generation is behind desired generation
```

Example when ready:

```yaml
conditions:
  - type: Ready
    status: "True"
    observedGeneration: 2
    reason: WorkersReady
    message: all worker groups are ready
  - type: Reconciling
    status: "False"
    observedGeneration: 2
    reason: ReconcileComplete
    message: desired state has been reconciled
```

## 8. Fake State Model

The fake controller can keep state in memory:

```go
type FakeStateStore struct {
    clusters map[string]infraapi.ManagedCluster
}
```

Key format:

```text
namespace/name
```

This state store is only for tests and demos.

It is not a replacement for Kubernetes API server persistence.

## 9. Adapter Boundary

The controller should call an adapter interface rather than hardcoding backend logic.

Suggested interface:

```go
type ClusterAdapter interface {
    Observe(ctx context.Context, cluster ManagedCluster) (ObservedClusterState, error)
    ApplyDesiredState(ctx context.Context, cluster ManagedCluster) error
}
```

Initial adapter:

```text
FakeClusterAdapter
```

Future adapters:

```text
ClusterAPIAdapter
KubeVirtAdapter
Metal3Adapter
CrossplaneAdapter
```

## 10. Safety Boundary

The controller should reject or ignore changes that try to mutate controller-owned fields through model-generated proposals.

Forbidden model-generated fields:

```text
status
metadata.finalizers
metadata.ownerReferences
spec.credentials
spec.secretRef
spec.bmcSecretRef
```

These should remain guarded by:

```text
SafetyGuard
API validation
PolicyChecker
Controller ownership boundaries
```

## 11. MVP Test Cases

Recommended fake controller tests:

```text
1. Valid ManagedCluster with replicas=3 and ready=3 remains Running.
2. Generation increases and replicas change 3 -> 6 enters Reconciling.
3. Repeated reconcile moves ready replicas toward 6.
4. When ready replicas reaches 6, Ready=True and observedGeneration updates.
5. Invalid ManagedCluster returns validation error.
6. Negative replicas is rejected by API validation.
7. Status-only mutation is not accepted as desired state input.
```

## 12. Future Production Controller

Only after the fake controller proves the state machine should the project introduce:

```text
controller-runtime
CRD generation
OpenAPI schemas
Status subresource
Admission policies
Leader election
Metrics
Events
Finalizers
Real backend adapters
```

## 13. Next Engineering Step

Recommended next step:

```text
infra/controller/fake.go
infra/controller/fake_test.go
```

But the fake implementation should stay small and deterministic.
