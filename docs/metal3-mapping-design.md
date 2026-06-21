# Metal3 Mapping Design

## 1. Goal

This document defines how `aicloud` can map governed infrastructure intent into Metal3-style bare metal desired state.

This is a design-only step. It does not implement a Metal3 controller, import Metal3 APIs, or apply live manifests.

## 2. Product Boundary

`aicloud` remains the policy-aware planning and governance layer.

Metal3 remains the Kubernetes-native bare metal host lifecycle layer.

```text
User Intent
  ↓
Model Gateway
  ↓
ChangePlan
  ↓
PolicyChecker
  ↓
ChangeProposal
  ↓
GitOps ManifestPatchPlan
  ↓
aicloud ManagedCluster / MachineClass
  ↓
Metal3 mapping layer
  ↓
Metal3-style bare metal desired state
```

The model does not create or mutate BareMetalHost resources directly.

## 3. Source Objects

Primary source objects:

```text
infra.aicloud.dev/ManagedCluster
infra.aicloud.dev/MachineClass
```

Important ManagedCluster fields:

```text
metadata.name
metadata.namespace
spec.environment
spec.workers[].name
spec.workers[].replicas
spec.workers[].machineClassRef.name
spec.networkRef
spec.policyRef
```

Important MachineClass fields:

```text
metadata.name
metadata.namespace
spec.cpu
spec.memory
spec.gpu
spec.storage
spec.labels
```

## 4. Target Metal3 Concepts

Initial mapping targets Metal3-style desired state for:

```text
BareMetalHost selection intent
BareMetalMachineTemplate-like profile
image reference
network data reference
host capability selector
```

Provider-specific details remain abstract until a real Metal3 dependency is introduced.

## 5. Mapping Table

| aicloud field | Metal3 target concept | Notes |
|---|---|---|
| `ManagedCluster.metadata.name` | host claim prefix | Stable ownership prefix. |
| `ManagedCluster.metadata.namespace` | target namespace | Same namespace unless policy overrides. |
| `spec.environment` | labels / annotations | Used for governance and routing. |
| `workers[].name` | host group label | One host set per worker group. |
| `workers[].replicas` | desired host count | Expansion selects additional hosts. |
| `workers[].machineClassRef.name` | MachineClass lookup | Defines CPU, memory, GPU and storage profile. |
| `MachineClass.spec.cpu` | host CPU capability selector | Must come from validated inventory/profile. |
| `MachineClass.spec.memory` | host memory capability selector | Must pass policy limits. |
| `MachineClass.spec.gpu` | GPU capability selector | Must be explicit and policy-approved. |
| `MachineClass.spec.storage` | storage capability selector | No inline credential material. |

## 6. First Supported Operation

The first operation remains scale-out:

```text
ManagedCluster workers[gpu-workers].replicas 3 -> 6
```

Metal3 mapping interpretation:

```text
3 existing host claims
  ↓
6 desired host claims
  ↓
3 additional host selection intents
```

Example generated host claim identity pattern:

```text
<cluster-name>-<worker-group-name>-host-0001
<cluster-name>-<worker-group-name>-host-0002
<cluster-name>-<worker-group-name>-host-0003
```

The mapper should produce desired host claims, not provision live bare metal hosts.

## 7. Labels

Generated bare metal desired state should include:

```text
aicloud.dev/managed-by: aicloud
aicloud.dev/managedcluster-name: <managedcluster-name>
aicloud.dev/environment: <environment>
aicloud.dev/worker-group: <worker-group-name>
aicloud.dev/machine-class: <machineclass-name>
```

## 8. Credential Boundary

Metal3 mapping must use references only.

Allowed reference style:

```text
managementAccessRef
imageRef
networkDataRef
userDataRef
```

Inline credential material is forbidden in model output, mapping output, GitOps patch plans and desired host shapes.

## 9. Hardware Selection Boundary

Hardware selection must be explicit and policy-controlled.

The model may propose:

```text
worker group uses gpu-large MachineClass
replicas increase from 3 to 6
```

The mapper may derive:

```text
bare metal host selection requires GPU-capable hardware profile
```

The mapper must not invent hardware inventory values, node identities, management endpoints, device identifiers or host access details.

Those must come from a validated MachineClass, host inventory source or operator-owned template.

## 10. GitOps Boundary

Allowed current output:

```text
DesiredBareMetalHostClaim-like shape
ManifestPatchPlan
BranchPlan
CommitPlan
PullRequestPlan
```

Not allowed yet:

```text
live power action
live provisioning
live deprovisioning
firmware mutation
management access mutation
storage wipe action
```

## 11. Status Mapping

Future status summarization can map:

| Metal3 source | aicloud status |
|---|---|
| host online / ready | Ready |
| host provisioning | Reconciling |
| host inspection running | Reconciling |
| host error condition | Degraded |
| host unavailable | Degraded |
| insufficient matching hosts | PolicyBlocked or Degraded |

## 12. Failure Modes

The mapper should fail closed on:

```text
missing MachineClass
unsupported hardware profile
replica delta above policy threshold
attempted inline credential material
attempted direct host pinning from model output
attempted live host lifecycle action
unknown network data policy
missing approval for high-risk hardware expansion
```

## 13. Future Package Layout

Recommended future layout:

```text
infra/mapping/metal3/
  README.md
  types.go
  mapper.go
  mapper_test.go
```

The first implementation should define provider-neutral host desired shapes and pure mapping tests.

Do not import Metal3 dependencies until the mapping contract is stable.

## 14. Not Done Yet

```text
- real Metal3 API types
- real BareMetalHost YAML generation
- live host lifecycle operation
- hardware inventory discovery
- management endpoint integration
```

## 15. Next Step

The next engineering step is a pure mapper skeleton that converts:

```text
ManagedCluster + MachineClass
```

into:

```text
DesiredBareMetalHostClaim[]
```
