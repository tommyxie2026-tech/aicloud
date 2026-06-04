# Infrastructure Adapter Layer

## 1. Goal

The `infra/adapter` package defines the backend boundary for infrastructure reconciliation.

It keeps `aicloud` independent from any single infrastructure implementation.

The adapter layer allows `ManagedCluster` and future infrastructure APIs to map to different backends:

```text
FakeClusterAdapter
ClusterAPIAdapter
KubeVirtAdapter
Metal3Adapter
CrossplaneAdapter
InternalCloudAdapter
```

Current implementation:

```text
FakeClusterAdapter only
```

## 2. Design Principle

```text
API objects describe desired state.
Controller reconciles state.
Adapter talks to backend.
Backend performs actual infrastructure operations.
```

The model and agent layers must not call adapters directly.

Allowed flow:

```text
Model -> ChangePlan -> Policy -> PR/GitOps -> Controller -> Adapter -> Backend
```

Not allowed:

```text
Model -> Adapter -> Backend
```

## 3. Current Interface

```go
type ClusterAdapter interface {
    Observe(ctx context.Context, cluster infraapi.ManagedCluster) (ObservedClusterState, error)
    ApplyDesiredState(ctx context.Context, cluster infraapi.ManagedCluster) error
}
```

## 4. ObservedClusterState

`ObservedClusterState` is a normalized backend observation.

Current fields:

```text
ReadyReplicas
Phase
Conditions
```

It intentionally does not expose provider-specific implementation details.

Future provider-specific details should be added through typed adapter-specific structs or status extensions, not by leaking backend objects into the core API too early.

## 5. FakeClusterAdapter

`FakeClusterAdapter` is the first implementation.

Purpose:

```text
- deterministic tests
- demos
- local development
- no external side effects
```

Behavior:

```text
- Observe returns in-memory state when present
- Observe falls back to ManagedCluster.status when no state exists
- ApplyDesiredState moves ReadyReplicas one step toward desired replicas
- ApplyDesiredState sets Phase to Reconciling or Running
```

Non-goals:

```text
- no Kubernetes client
- no Cluster API call
- no KubeVirt call
- no Metal3 call
- no Crossplane call
- no credential handling
- no destructive operation
```

## 6. Cluster API Adapter Boundary

Future `ClusterAPIAdapter` should map:

```text
ManagedCluster -> Cluster API Cluster
WorkerGroupSpec -> MachineDeployment
MachineClass -> MachineTemplate / provider-specific machine template
```

Initial adapter responsibilities:

```text
- observe MachineDeployment ready replicas
- observe Cluster conditions
- generate or apply desired MachineDeployment replica count through GitOps/controller path
```

Non-goals for first adapter:

```text
- creating every Cluster API provider integration
- handling cloud credentials directly
- bypassing approval workflow
- direct production mutation from model output
```

## 7. KubeVirt Adapter Boundary

Future `KubeVirtAdapter` should map:

```text
VirtualMachine -> KubeVirt VirtualMachine
MachineClass -> VM flavor / template
ManagedMachine -> VM-backed machine abstraction
```

Initial adapter responsibilities:

```text
- observe VM readiness
- map machine profile to VM template
- expose VM-backed machine status
```

Non-goals for early stage:

```text
- full VM lifecycle management
- live migration policy
- storage/network deep integration
- production destructive operations
```

## 8. Metal3 Adapter Boundary

Future `Metal3Adapter` should map:

```text
BareMetalHost -> Metal3 BareMetalHost
MachineClass -> hardware profile
ManagedMachine -> bare-metal machine abstraction
```

Initial adapter responsibilities:

```text
- observe bare-metal host provisioning state
- map hardware profile to machine class
- expose host readiness
```

Strict boundary:

```text
- no BMC credential exposure to model layer
- no direct power operation from model or agent
- destructive bare-metal actions require explicit policy and approval
```

## 9. Crossplane Adapter Boundary

Future `CrossplaneAdapter` should map:

```text
ManagedCluster -> CompositeResource
MachineClass -> Composition parameters
ProviderRef -> Crossplane ProviderConfig
```

Initial adapter responsibilities:

```text
- observe composite resource readiness
- map high-level desired state to composition parameters
- keep provider-specific resources behind Crossplane abstractions
```

## 10. Adapter Error Strategy

Adapters should return normalized errors to the controller layer.

Suggested categories:

```text
ValidationError
BackendUnavailable
BackendConflict
Unauthorized
RateLimited
ReconcileInProgress
UnknownBackendError
```

The controller should convert adapter errors into status conditions:

```text
Degraded=True
Ready=False
Reconciling=False or True depending on retryability
```

## 11. Security Boundary

Adapters must never expose sensitive values upward.

Forbidden upward data:

```text
raw credentials
kubeconfig contents
BMC passwords
tokens
private keys
provider secrets
```

Allowed upward data:

```text
readiness
phase
condition reason
condition message
resource identifiers
sanitized error category
```

## 12. Current Tests

Current tests:

```text
infra/adapter/adapter_test.go
```

Covered behavior:

```text
- Observe falls back to ManagedCluster.status
- ApplyDesiredState scale-out one step
- ApplyDesiredState scale-down one step
- ApplyDesiredState sets Running when ready
- invalid ManagedCluster is rejected
```

## 13. Next Steps

Recommended next steps:

```text
1. Add adapter error types.
2. Wire FakeClusterAdapter into FakeController.
3. Add adapter-driven fake reconcile tests.
4. Add Cluster API mapping design doc.
5. Add KubeVirt mapping design doc.
6. Add Metal3 mapping design doc.
```
