# Cluster API Mapping Design

## 1. Goal

This document defines how `aicloud` maps its AI-governed infrastructure intent into Cluster API concepts.

The goal is not to implement a real Cluster API controller yet. The goal is to define a stable translation boundary that can be reviewed before any live backend integration is added.

## 2. Product Boundary

`aicloud` is the policy-aware planning and governance layer.

Cluster API remains the Kubernetes-native cluster lifecycle backend.

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
aicloud ManagedCluster
  ↓
Cluster API mapping layer
  ↓
Cluster API resources
```

The model does not write Cluster API resources directly.

## 3. Source Object

The source object is:

```text
infra.aicloud.dev/ManagedCluster
```

Important fields:

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

## 4. Target Cluster API Resources

The initial mapping targets these Cluster API concepts:

```text
Cluster
MachineDeployment
MachineTemplate / InfrastructureMachineTemplate
KubeadmControlPlane or equivalent control plane resource
```

Provider-specific infrastructure resources are intentionally abstracted:

```text
AWSCluster / AWSMachineTemplate
AzureCluster / AzureMachineTemplate
VSphereCluster / VSphereMachineTemplate
OpenStackCluster / OpenStackMachineTemplate
BareMetalCluster / Metal3MachineTemplate
```

The first implementation should not hard-code a cloud provider.

## 5. Mapping Table

| aicloud field | Cluster API target | Notes |
|---|---|---|
| `metadata.name` | `Cluster.metadata.name` | Stable cluster name. |
| `metadata.namespace` | target namespace | Same namespace unless policy overrides. |
| `spec.environment` | labels / annotations | Used for policy and routing. |
| `spec.workers[].name` | `MachineDeployment.metadata.name` | One MachineDeployment per worker group. |
| `spec.workers[].replicas` | `MachineDeployment.spec.replicas` | Main scale-out field. |
| `spec.workers[].machineClassRef.name` | Machine template selector | Resolved through a provider-specific template mapping. |
| `spec.networkRef` | infra cluster network fields | Provider-specific. |
| `spec.policyRef` | labels / annotations | Used by governance; not a backend credential. |

## 6. First Supported Operation

The first operation remains:

```text
ManagedCluster workers[gpu-workers].replicas 3 -> 6
```

Mapping:

```text
ManagedCluster.spec.workers[name=gpu-workers].replicas
  ↓
MachineDeployment/<cluster-name>-gpu-workers.spec.replicas
```

This operation is safe to model as a GitOps patch.

## 7. Ownership and Labels

Generated Cluster API resources should carry stable ownership labels:

```text
aicloud.dev/managed-by: aicloud
aicloud.dev/managedcluster-name: <managedcluster-name>
aicloud.dev/environment: <environment>
aicloud.dev/worker-group: <worker-group-name>
```

OwnerReferences should be added only after the real controller ownership model is designed.

For now, GitOps mapping should prefer labels and annotations over ownerReferences.

## 8. Credentials Boundary

Cluster API provider credentials must not be stored in `ManagedCluster` or model output.

Allowed:

```text
providerConfigRef
secretRef
identityRef
```

Forbidden:

```text
raw access key
raw secret key
raw kubeconfig
raw token
raw password
private key material
```

## 9. GitOps Patch Boundary

The model and agent may propose:

```text
ManagedCluster patch
```

The mapper may derive:

```text
MachineDeployment patch
```

But live apply remains outside the current MVP.

Current allowed GitOps output:

```text
ManifestPatchPlan
DryRunManifestWriter
BranchPlan
CommitPlan
PullRequestPlan
```

Not allowed yet:

```text
kubectl apply
clusterctl move
clusterctl init
controller-runtime reconcile
provider credential mutation
```

## 10. Reconciliation Boundary

The future controller should follow this responsibility split:

```text
aicloud controller:
  - observe ManagedCluster
  - derive desired Cluster API resource shape
  - compare desired vs observed
  - update status and conditions

Cluster API controllers:
  - create/update machines
  - reconcile infrastructure resources
  - manage bootstrap/control plane lifecycle
```

`aicloud` should not duplicate Cluster API machine lifecycle logic.

## 11. Status Mapping

Cluster API status can be summarized into `ManagedCluster.status`.

| Cluster API source | aicloud status |
|---|---|
| `Cluster.status.conditions` | `ManagedCluster.status.conditions` |
| `MachineDeployment.status.readyReplicas` | worker group ready replica summary |
| infrastructure resource ready condition | `Ready` / `Degraded` |
| reconcile in progress | `Reconciling` |
| blocked by policy | `PolicyBlocked` |
| waiting approval | `ApprovalPending` |

## 12. Failure Modes

The mapping layer should fail closed on:

```text
unknown worker group
missing machineClassRef
unsupported provider mapping
attempted credential mutation
attempted status patch from model output
replica delta above policy threshold
missing approval for high-risk change
```

## 13. Future Package Layout

Recommended future package layout:

```text
infra/mapping/clusterapi/
  README.md
  mapper.go
  mapper_test.go
  types.go
```

The first implementation should be pure mapping logic with fake structs or minimal interfaces.

Do not import Cluster API dependencies until the mapping contract is stable.

## 14. Not Done Yet

```text
- real Cluster API client
- real controller-runtime reconciliation
- provider-specific templates
- clusterctl integration
- YAML round-trip writer
- live apply
```

## 15. Next Step

The next engineering step is a pure mapper skeleton that converts:

```text
ManagedCluster + worker group
```

into a provider-neutral desired MachineDeployment-like shape.
