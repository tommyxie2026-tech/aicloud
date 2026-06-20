# KubeVirt Mapping Design

## 1. Goal

This document defines how `aicloud` can map governed infrastructure intent into KubeVirt-style virtual machine desired state.

This is a design-only step. It does not implement a KubeVirt controller, import KubeVirt APIs, or apply live manifests.

## 2. Product Boundary

`aicloud` remains the policy-aware planning and governance layer.

KubeVirt remains the Kubernetes-native VM runtime layer.

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
KubeVirt mapping layer
  ↓
KubeVirt-style VM desired state
```

The model does not create or mutate VirtualMachine resources directly.

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

## 4. Target KubeVirt Concepts

Initial mapping targets KubeVirt-style desired state for:

```text
VirtualMachine
VirtualMachineInstance template
DataVolume or PVC reference
Network attachment reference
Cloud-init reference
```

Provider-specific details remain abstract until a real KubeVirt dependency is introduced.

## 5. Mapping Table

| aicloud field | KubeVirt target concept | Notes |
|---|---|---|
| `ManagedCluster.metadata.name` | VM name prefix | Stable ownership prefix. |
| `ManagedCluster.metadata.namespace` | VM namespace | Same namespace unless policy overrides. |
| `spec.environment` | labels / annotations | Used for governance and routing. |
| `workers[].name` | VM group label | One VM set per worker group. |
| `workers[].replicas` | desired VM count | Expansion produces additional VM desired states. |
| `workers[].machineClassRef.name` | MachineClass lookup | Defines CPU, memory, GPU, storage profile. |
| `MachineClass.spec.cpu` | VM CPU profile | Provider-neutral until KubeVirt schema is imported. |
| `MachineClass.spec.memory` | VM memory profile | Must pass policy limits. |
| `MachineClass.spec.gpu` | GPU profile reference | Must be explicit and policy-approved. |
| `MachineClass.spec.storage` | volume profile reference | No inline credential material. |

## 6. First Supported Operation

The first operation remains scale-out:

```text
ManagedCluster workers[gpu-workers].replicas 3 -> 6
```

KubeVirt mapping interpretation:

```text
3 existing VM desired identities
  ↓
6 desired VM identities
  ↓
3 additional VM desired states
```

Example generated identity pattern:

```text
<cluster-name>-<worker-group-name>-0001
<cluster-name>-<worker-group-name>-0002
<cluster-name>-<worker-group-name>-0003
```

The mapper should produce desired identities, not create live VMs.

## 7. Labels

Generated VM desired state should include:

```text
aicloud.dev/managed-by: aicloud
aicloud.dev/managedcluster-name: <managedcluster-name>
aicloud.dev/environment: <environment>
aicloud.dev/worker-group: <worker-group-name>
aicloud.dev/machine-class: <machineclass-name>
```

## 8. Credential Boundary

KubeVirt mapping must use references only.

Allowed reference style:

```text
cloudInitRef
imagePullRef
storageClassRef
networkAttachmentRef
```

Inline credential material is forbidden in model output, mapping output, GitOps patch plans and desired VM shapes.

## 9. GPU Boundary

GPU scheduling must be explicit and policy-controlled.

The model may propose:

```text
worker group uses gpu-large MachineClass
replicas increase from 3 to 6
```

The mapper may derive:

```text
VM desired state requires GPU-capable scheduling profile
```

The mapper must not invent provider-specific device identifiers, node names, host device rules or passthrough details.

Those must come from a validated MachineClass or operator-owned template.

## 10. GitOps Boundary

Allowed current output:

```text
DesiredVirtualMachine-like shape
ManifestPatchPlan
BranchPlan
CommitPlan
PullRequestPlan
```

Not allowed yet:

```text
live VM start
live VM stop
live VM migration
host device mutation
storage credential mutation
```

## 11. Status Mapping

Future status summarization can map:

| KubeVirt source | aicloud status |
|---|---|
| VM ready condition | worker ready count |
| VMI phase Running | Ready |
| VMI phase Pending / Scheduling | Reconciling |
| VMI phase Failed | Degraded |
| DataVolume import pending | Reconciling |
| node/device unavailable | Degraded |

## 12. Failure Modes

The mapper should fail closed on:

```text
missing MachineClass
unsupported GPU request
replica delta above policy threshold
attempted inline credential material
attempted node pinning from model output
attempted live VM lifecycle action
unknown network attachment policy
missing approval for high-risk GPU expansion
```

## 13. Future Package Layout

Recommended future layout:

```text
infra/mapping/kubevirt/
  README.md
  types.go
  mapper.go
  mapper_test.go
```

The first implementation should define provider-neutral VM desired shapes and pure mapping tests.

Do not import KubeVirt dependencies until the mapping contract is stable.

## 14. Not Done Yet

```text
- real KubeVirt API types
- real VirtualMachine YAML generation
- live VM lifecycle operation
- node/device discovery
- GPU passthrough implementation
```

## 15. Next Step

The next engineering step is a pure mapper skeleton that converts:

```text
ManagedCluster + MachineClass
```

into:

```text
DesiredVirtualMachine[]
```
