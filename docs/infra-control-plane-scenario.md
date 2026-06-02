# Infrastructure Control Plane Scenario

## 1. Purpose

This document defines the first high-value scenario for `aicloud`:

```text
AI-assisted infrastructure change planning and control.
```

The concrete infrastructure direction is:

```text
Use Kubernetes-style APIs to manage clusters, virtual machines, and eventually physical machines.
```

This scenario should not redefine `aicloud` as only a Kubernetes controller.

Instead, it should use the core `aicloud` platform capabilities:

```text
Model Gateway
Provider Routing
Structured Output
Safety Guard
Evaluation
Policy Checker
Agent Workflow
GitOps / Controller Reconciliation
```

## 2. Product Positioning

`aicloud` remains:

```text
Hybrid Private AI Cloud Platform + AI-native Infrastructure Control Plane
```

The infrastructure control plane is the first product scenario, not the whole product boundary.

## 3. Scenario Statement

Platform engineers should be able to say:

```text
Scale dev-gpu-cluster gpu-workers from 3 to 6.
```

The system should produce:

```text
- structured ChangePlan
- deterministic policy result
- rollback plan
- validation checklist
- PR-ready change proposal
```

It should not directly execute destructive infrastructure actions from model output.

## 4. Control Flow

```text
User intent
  ↓
AIGateway.GeneratePlan
  ↓
ModelRouter selects provider
  ↓
Provider returns structured ChangePlan
  ↓
SafetyGuard validates boundary
  ↓
SchemaValidator validates output
  ↓
PolicyChecker calculates risk and approval
  ↓
Agent workflow creates ChangeProposal
  ↓
PR draft is generated
  ↓
Human review if required
  ↓
GitOps applies desired state
  ↓
Controller reconciles infrastructure
```

## 5. Why Kubernetes API Style

Kubernetes-style APIs are useful because they provide:

```text
- declarative desired state
- reconciliation loop
- CRD extensibility
- status and conditions
- GitOps compatibility
- RBAC and admission control
- ecosystem integration
```

This makes them a good control-plane substrate for VM and physical-machine management.

## 6. Target Resource Model

Initial resource concepts:

```text
ManagedCluster
MachineClass
ManagedMachine
AgentOperation
RiskPolicy
ApprovalPolicy
```

Future resource concepts:

```text
VirtualMachine
BareMetalHost
MachinePool
GPUNodePool
NetworkProfile
StorageProfile
ModelProviderConfig
ModelEndpoint
```

## 7. MVP Resource: ManagedCluster

Example desired state:

```yaml
apiVersion: infra.aicloud.dev/v1alpha1
kind: ManagedCluster
metadata:
  name: dev-gpu-cluster
  namespace: default
spec:
  workers:
    - name: gpu-workers
      replicas: 3
      machineClassRef:
        name: gpu-large
status:
  phase: Running
  workerReadyReplicas: 3
  conditions:
    - type: Ready
      status: "True"
```

MVP change:

```text
spec.workers[name=gpu-workers].replicas: 3 -> 6
```

## 8. MVP Structured ChangePlan

The model should only generate structured proposal output:

```text
ChangePlan
- target: ManagedCluster/dev-gpu-cluster
- field: spec.workers[name=gpu-workers].replicas
- from: 3
- to: 6
- riskHint: Medium
- rollback: set replicas back to 3
```

The model should not directly apply the change.

## 9. Policy Rules

The deterministic policy layer should decide risk and approval.

Example rules:

```text
dev ManagedCluster workers replicas +3:
  riskLevel = Medium
  approvalRequired = false

staging ManagedCluster workers replicas +3:
  riskLevel = Medium
  approvalRequired = true

production ManagedCluster workers replicas change:
  riskLevel = High
  approvalRequired = true

unknown field mutation:
  fail closed
  approvalRequired = true

status / finalizers / credentials mutation:
  blocked
```

## 10. Safety Boundary

Model and agent must not perform:

```text
- direct kubectl apply
- direct machine power operation
- credential read
- BMC secret access
- production delete
- automatic approval
- automatic merge
```

All execution must go through:

```text
PR / approval / GitOps / controller reconciliation
```

## 11. Community Architecture Mapping

This scenario should map to mature cloud-native building blocks instead of inventing everything.

## 11.1 Cluster API

Use case:

```text
Manage Kubernetes clusters and MachineDeployments through declarative APIs.
```

Mapping:

```text
ManagedCluster -> Cluster API Cluster / MachineDeployment
MachineClass -> infrastructure provider machine template
```

Role in aicloud:

```text
Primary target for cluster and node-pool lifecycle integration.
```

## 11.2 KubeVirt

Use case:

```text
Run and manage virtual machines on Kubernetes.
```

Mapping:

```text
VirtualMachine -> KubeVirt VirtualMachine
ManagedMachine -> VM-backed machine abstraction
```

Role in aicloud:

```text
Primary VM management candidate.
```

## 11.3 Metal3

Use case:

```text
Manage bare metal hosts through Kubernetes APIs.
```

Mapping:

```text
BareMetalHost -> Metal3 BareMetalHost
MachineClass -> bare-metal hardware profile
```

Role in aicloud:

```text
Physical machine management candidate.
```

## 11.4 Crossplane

Use case:

```text
Compose infrastructure resources across providers using Kubernetes APIs.
```

Mapping:

```text
ManagedCluster / MachineClass -> Crossplane composite resources
```

Role in aicloud:

```text
Multi-cloud and platform composition layer.
```

## 11.5 Kamaji / vCluster

Use case:

```text
Run or manage Kubernetes control planes with lighter isolation models.
```

Mapping:

```text
ManagedCluster -> virtual/control-plane abstraction
```

Role in aicloud:

```text
Potential lightweight cluster management option.
```

## 12. aicloud Differentiation

`aicloud` should not compete with Cluster API, KubeVirt, Metal3, or Crossplane directly.

Instead, it should sit above them as:

```text
AI-native planning, governance, policy, and workflow layer.
```

Community tools provide:

```text
- actual infrastructure reconciliation
- CRDs
- controllers
- provider-specific execution
```

`aicloud` provides:

```text
- intent understanding
- model-based proposal generation
- structured ChangePlan
- policy-aware approval workflow
- evaluation and model governance
- PR/GitOps-ready output
- audit and explainability
```

## 13. MVP Scope

MVP should include:

```text
- ManagedCluster design
- MachineClass design
- ChangePlan generation
- PolicyChecker for small scale-out
- PR draft generation
- fake controller status simulation
```

MVP should not include:

```text
- real production cluster mutation
- destructive workflows
- broad CRD surface
- physical machine power operation
- BMC credential handling
- automatic merge or approval
```

## 14. Future Expansion

Future stages:

```text
R1 ManagedCluster planning only
R2 GitOps manifest generation
R3 fake controller status loop
R4 Cluster API adapter
R5 KubeVirt VM adapter
R6 Metal3 bare-metal adapter
R7 Crossplane composition adapter
R8 production-grade approval/audit/RBAC
```

## 15. Current Engineering Mapping

Current implemented pieces:

```text
model/gateway      GeneratePlan
model/schema       ChangePlan
model/safety       boundary validation
policy/checker     deterministic risk and approval
agent/proposal     ChangeProposal
agent/prdraft      PR draft generation
agent/pipeline     end-to-end planning pipeline
```

Missing pieces:

```text
infra/api          ManagedCluster / MachineClass types
infra/controller   fake reconciliation loop
infra/adapter      Cluster API / KubeVirt / Metal3 adapters
integrations/gitops manifest generation
integrations/github real PR creation
```

## 16. Next Design Tasks

```text
1. Add ManagedCluster API design.
2. Add MachineClass API design.
3. Add AgentOperation API design.
4. Add fake controller reconciliation design.
5. Add Cluster API adapter mapping.
6. Add KubeVirt adapter mapping.
7. Add Metal3 adapter mapping.
8. Add GitOps manifest generation design.
```
