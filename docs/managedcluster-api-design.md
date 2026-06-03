# ManagedCluster API Design

## 1. Purpose

This document defines the first infrastructure API design for `aicloud`.

The goal is to design Kubernetes-style APIs for:

```text
AI-assisted infrastructure change planning and control
```

The initial scenario is:

```text
ManagedCluster workers replicas 3 -> 6
```

The API should borrow from Kubernetes project conventions:

```text
- Group / Version / Kind
- Spec / Status separation
- declarative desired state
- controller reconciliation
- status conditions
- observedGeneration
- finalizers
- owner references
- subresource boundaries
- validation and admission style guardrails
```

## 2. Design Principles

```text
1. Spec is desired state.
2. Status is observed state.
3. Controllers reconcile actual state toward desired state.
4. Status must not be directly written by users or models.
5. Risk and approval are policy decisions, not model decisions.
6. Destructive or sensitive actions must fail closed.
7. Infrastructure execution must go through GitOps / controller reconciliation.
8. The model only proposes structured ChangePlan objects.
```

## 3. API Groups

Suggested API groups:

```text
infra.aicloud.dev       infrastructure desired-state APIs
agent.aicloud.dev       agent operation and workflow APIs
policy.aicloud.dev      risk and approval policy APIs
model.aicloud.dev       model provider and routing APIs
```

Initial API versions:

```text
v1alpha1
```

## 4. Initial Kinds

```text
ManagedCluster
MachineClass
AgentOperation
RiskPolicy
ApprovalPolicy
```

Future kinds:

```text
ManagedMachine
VirtualMachine
BareMetalHost
MachinePool
GPUNodePool
NetworkProfile
StorageProfile
```

## 5. ManagedCluster

### 5.1 Purpose

`ManagedCluster` represents a Kubernetes-style desired-state object for a managed infrastructure cluster.

It is not necessarily a Kubernetes workload cluster only. In future stages, it may map to:

```text
- Cluster API Cluster / MachineDeployment
- KubeVirt VM-backed clusters
- Metal3 bare-metal clusters
- Crossplane composite infrastructure
- internal cloud platform clusters
```

### 5.2 Example YAML

```yaml
apiVersion: infra.aicloud.dev/v1alpha1
kind: ManagedCluster
metadata:
  name: dev-gpu-cluster
  namespace: default
spec:
  environment: dev
  providerRef:
    kind: ClusterProvider
    name: capi-dev
  workers:
    - name: gpu-workers
      replicas: 3
      machineClassRef:
        name: gpu-large
      labels:
        workload: gpu
status:
  observedGeneration: 1
  phase: Running
  workerReadyReplicas: 3
  conditions:
    - type: Ready
      status: "True"
      reason: WorkersReady
      message: all worker groups are ready
```

### 5.3 Spec

Suggested Go-like shape:

```go
type ManagedClusterSpec struct {
    Environment string `json:"environment"`
    ProviderRef LocalObjectReference `json:"providerRef,omitempty"`
    Workers []WorkerGroupSpec `json:"workers,omitempty"`
}

type WorkerGroupSpec struct {
    Name string `json:"name"`
    Replicas int32 `json:"replicas"`
    MachineClassRef LocalObjectReference `json:"machineClassRef"`
    Labels map[string]string `json:"labels,omitempty"`
}
```

Spec rules:

```text
- spec.environment is required.
- spec.workers[].name must be unique.
- spec.workers[].replicas must be >= 0.
- spec.workers[].machineClassRef is required.
- model-generated changes are only allowed through reviewed workflow.
```

### 5.4 Status

Suggested Go-like shape:

```go
type ManagedClusterStatus struct {
    ObservedGeneration int64 `json:"observedGeneration,omitempty"`
    Phase string `json:"phase,omitempty"`
    WorkerReadyReplicas int32 `json:"workerReadyReplicas,omitempty"`
    Conditions []Condition `json:"conditions,omitempty"`
}
```

Status rules:

```text
- status is controller-owned.
- users and models must not mutate status.
- status.observedGeneration tracks the latest reconciled metadata.generation.
- status.conditions explain readiness, progress, degradation, and policy blocking.
```

### 5.5 Conditions

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

Example:

```yaml
status:
  conditions:
    - type: Ready
      status: "False"
      reason: ScalingWorkers
      message: gpu-workers is scaling from 3 to 6
    - type: Reconciling
      status: "True"
      reason: DesiredStateChanged
      message: controller observed a new desired replica count
```

## 6. MachineClass

### 6.1 Purpose

`MachineClass` describes reusable machine profiles.

It is similar in spirit to a class/template object and can map to different backends:

```text
- Cluster API machine templates
- KubeVirt VM flavor templates
- Metal3 hardware profiles
- cloud instance types
- internal platform machine pools
```

### 6.2 Example YAML

```yaml
apiVersion: infra.aicloud.dev/v1alpha1
kind: MachineClass
metadata:
  name: gpu-large
spec:
  provider: internal-cloud
  cpu: "32"
  memory: 128Gi
  gpu:
    count: 4
    type: A100
  labels:
    workload: gpu
```

### 6.3 Spec

```go
type MachineClassSpec struct {
    Provider string `json:"provider"`
    CPU string `json:"cpu,omitempty"`
    Memory string `json:"memory,omitempty"`
    GPU *GPUSpec `json:"gpu,omitempty"`
    Labels map[string]string `json:"labels,omitempty"`
}

type GPUSpec struct {
    Count int32 `json:"count"`
    Type string `json:"type"`
}
```

Rules:

```text
- MachineClass is referenced by ManagedCluster worker groups.
- MachineClass changes may be higher risk than replica count changes.
- GPU profile changes should require explicit policy review.
```

## 7. AgentOperation

### 7.1 Purpose

`AgentOperation` records an AI-assisted infrastructure workflow.

It should be status-oriented and auditable.

### 7.2 Example YAML

```yaml
apiVersion: agent.aicloud.dev/v1alpha1
kind: AgentOperation
metadata:
  name: scale-dev-gpu-cluster
  namespace: default
spec:
  intent: scale dev-gpu-cluster gpu-workers from 3 to 6
  targetRef:
    apiVersion: infra.aicloud.dev/v1alpha1
    kind: ManagedCluster
    name: dev-gpu-cluster
  operationType: ScaleOut
  proposedChanges:
    - field: spec.workers[name=gpu-workers].replicas
      from: 3
      to: 6
  rollback:
    summary: set gpu-workers replicas back to 3
  validation:
    expected:
      - workerReadyReplicas=6
status:
  phase: ProposalGenerated
  policyResult:
    riskLevel: Medium
    approvalRequired: false
    matchedRule: dev-managedcluster-small-scale
  conditions:
    - type: PolicyEvaluated
      status: "True"
      reason: RuleMatched
```

### 7.3 Phases

Recommended phases:

```text
Pending
PlanGenerated
PolicyEvaluated
ApprovalPending
PRCreated
Merged
Reconciling
Validated
Failed
RollbackProposed
RolledBack
```

## 8. RiskPolicy

### 8.1 Purpose

`RiskPolicy` defines deterministic risk rules.

It should not be generated by a model.

### 8.2 Example YAML

```yaml
apiVersion: policy.aicloud.dev/v1alpha1
kind: RiskPolicy
metadata:
  name: default-risk-policy
spec:
  rules:
    - name: dev-managedcluster-small-scale
      match:
        environment: dev
        targetKind: ManagedCluster
        operationType: ScaleOut
        field: spec.workers[name=gpu-workers].replicas
      constraints:
        maxReplicaDelta: 3
      result:
        riskLevel: Medium
        approvalRequired: false
    - name: production-managedcluster-scale
      match:
        environment: production
        targetKind: ManagedCluster
        operationType: ScaleOut
      result:
        riskLevel: High
        approvalRequired: true
```

## 9. ApprovalPolicy

### 9.1 Purpose

`ApprovalPolicy` defines who must approve a proposed change.

### 9.2 Example YAML

```yaml
apiVersion: policy.aicloud.dev/v1alpha1
kind: ApprovalPolicy
metadata:
  name: default-approval-policy
spec:
  rules:
    - name: high-risk-production
      match:
        riskLevel: High
        environment: production
      requiredApprovers:
        - platform-admin
        - sre-lead
    - name: medium-risk-staging
      match:
        riskLevel: Medium
        environment: staging
      requiredApprovers:
        - platform-engineer
```

## 10. Reconciliation Model

The controller should follow a Kubernetes-style reconciliation model:

```text
watch ManagedCluster
  ↓
compare spec with observed external state
  ↓
call adapter if approved desired state changed
  ↓
update status
  ↓
set conditions
  ↓
requeue if still reconciling
```

Important rules:

```text
- Reconcile must be idempotent.
- Status updates must be separate from spec updates.
- Controller should update observedGeneration after processing current generation.
- External side effects must be guarded by policy and approval state.
```

## 11. Finalizers

Finalizers should be used only when external resource cleanup is needed.

Example finalizer:

```text
infra.aicloud.dev/managedcluster-finalizer
```

Rules:

```text
- no finalizer mutation from model output
- finalizer changes are controller-owned
- deletion must respect approval policy for destructive resources
```

## 12. Owner References

Owner references can connect generated child resources to parent objects.

Examples:

```text
AgentOperation owned by ManagedCluster change request context
Provider-specific child object owned by ManagedCluster
```

Rules:

```text
- model output must not mutate ownerReferences
- controller owns ownerReferences for generated children
```

## 13. Subresources

Recommended boundaries:

```text
/status      controller-owned observed state
/approval    optional future approval transition subresource
/rollback    optional future rollback proposal subresource
```

MVP should implement only conceptual status handling first.

## 14. Validation and Admission Guardrails

Validation rules:

```text
- block status mutation from spec path
- block metadata.finalizers mutation from model proposals
- block metadata.ownerReferences mutation from model proposals
- block credential references in model-generated patch
- enforce worker group name uniqueness
- enforce replica count bounds
- enforce immutable fields where needed
```

Future admission policies:

```text
- CEL validation rules
- ValidatingAdmissionPolicy
- OPA/Gatekeeper
- Kyverno
```

## 15. Backend Adapter Mapping

## 15.1 Cluster API Adapter

```text
ManagedCluster -> Cluster
WorkerGroupSpec -> MachineDeployment
MachineClass -> MachineTemplate / provider-specific template
```

## 15.2 KubeVirt Adapter

```text
VirtualMachine -> KubeVirt VirtualMachine
MachineClass -> VM flavor / template
ManagedMachine -> VM-backed machine abstraction
```

## 15.3 Metal3 Adapter

```text
BareMetalHost -> Metal3 BareMetalHost
MachineClass -> hardware profile
ManagedMachine -> bare-metal machine abstraction
```

## 15.4 Crossplane Adapter

```text
ManagedCluster -> CompositeResource
MachineClass -> Composition parameter
ProviderRef -> Crossplane ProviderConfig
```

## 16. MVP Scope

MVP should include:

```text
- ManagedCluster API design
- MachineClass API design
- AgentOperation design
- deterministic RiskPolicy design
- ApprovalPolicy design
- fake controller status flow
- PR-ready manifest generation design
```

MVP should not include:

```text
- direct production execution
- real BMC credential operations
- destructive physical machine workflows
- automatic merge
- automatic approval
- broad cloud-provider implementation
```

## 17. Next Engineering Tasks

```text
1. Create infra/api package with Go structs for ManagedCluster and MachineClass.
2. Create agent/api package or reuse agent/proposal for AgentOperation design.
3. Add YAML examples under examples/infra/.
4. Add fake controller design under infra/controller/README.md.
5. Add adapter mapping docs for Cluster API / KubeVirt / Metal3.
6. Extend SafetyGuard forbidden fields based on API design.
7. Extend PolicyChecker to read policy-like structs instead of hardcoded rules.
```
