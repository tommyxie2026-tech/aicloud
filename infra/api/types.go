package api

// This package intentionally keeps the first infrastructure API types
// dependency-light. Kubernetes runtime/object metadata can be introduced later
// when CRD generation is added.

const (
	GroupName = "infra.aicloud.dev"
	Version   = "v1alpha1"
)

const (
	KindManagedCluster = "ManagedCluster"
	KindMachineClass   = "MachineClass"
)

// TypeMeta is a lightweight Kubernetes-style type header.
type TypeMeta struct {
	APIVersion string `json:"apiVersion" yaml:"apiVersion"`
	Kind       string `json:"kind" yaml:"kind"`
}

// ObjectMeta is a lightweight metadata subset for early design and tests.
type ObjectMeta struct {
	Name              string            `json:"name" yaml:"name"`
	Namespace         string            `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Labels            map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Annotations       map[string]string `json:"annotations,omitempty" yaml:"annotations,omitempty"`
	Generation        int64             `json:"generation,omitempty" yaml:"generation,omitempty"`
	Finalizers        []string          `json:"finalizers,omitempty" yaml:"finalizers,omitempty"`
	OwnerReferences   []OwnerReference  `json:"ownerReferences,omitempty" yaml:"ownerReferences,omitempty"`
}

type OwnerReference struct {
	APIVersion string `json:"apiVersion" yaml:"apiVersion"`
	Kind       string `json:"kind" yaml:"kind"`
	Name       string `json:"name" yaml:"name"`
}

type LocalObjectReference struct {
	Kind string `json:"kind,omitempty" yaml:"kind,omitempty"`
	Name string `json:"name" yaml:"name"`
}

// ManagedCluster represents the desired and observed state of a managed cluster.
type ManagedCluster struct {
	TypeMeta   `json:",inline" yaml:",inline"`
	ObjectMeta `json:"metadata" yaml:"metadata"`
	Spec       ManagedClusterSpec   `json:"spec" yaml:"spec"`
	Status     ManagedClusterStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

type ManagedClusterSpec struct {
	Environment string               `json:"environment" yaml:"environment"`
	ProviderRef LocalObjectReference `json:"providerRef,omitempty" yaml:"providerRef,omitempty"`
	Workers     []WorkerGroupSpec    `json:"workers,omitempty" yaml:"workers,omitempty"`
}

type WorkerGroupSpec struct {
	Name            string               `json:"name" yaml:"name"`
	Replicas        int32                `json:"replicas" yaml:"replicas"`
	MachineClassRef LocalObjectReference `json:"machineClassRef" yaml:"machineClassRef"`
	Labels          map[string]string    `json:"labels,omitempty" yaml:"labels,omitempty"`
}

type ManagedClusterStatus struct {
	ObservedGeneration int64       `json:"observedGeneration,omitempty" yaml:"observedGeneration,omitempty"`
	Phase              string      `json:"phase,omitempty" yaml:"phase,omitempty"`
	WorkerReadyReplicas int32      `json:"workerReadyReplicas,omitempty" yaml:"workerReadyReplicas,omitempty"`
	Conditions         []Condition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

// MachineClass describes a reusable machine profile.
type MachineClass struct {
	TypeMeta   `json:",inline" yaml:",inline"`
	ObjectMeta `json:"metadata" yaml:"metadata"`
	Spec       MachineClassSpec `json:"spec" yaml:"spec"`
}

type MachineClassSpec struct {
	Provider string            `json:"provider" yaml:"provider"`
	CPU      string            `json:"cpu,omitempty" yaml:"cpu,omitempty"`
	Memory   string            `json:"memory,omitempty" yaml:"memory,omitempty"`
	GPU      *GPUSpec          `json:"gpu,omitempty" yaml:"gpu,omitempty"`
	Labels   map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

type GPUSpec struct {
	Count int32  `json:"count" yaml:"count"`
	Type  string `json:"type" yaml:"type"`
}

// Condition follows the Kubernetes condition style.
type Condition struct {
	Type               string `json:"type" yaml:"type"`
	Status             string `json:"status" yaml:"status"`
	ObservedGeneration int64  `json:"observedGeneration,omitempty" yaml:"observedGeneration,omitempty"`
	Reason             string `json:"reason,omitempty" yaml:"reason,omitempty"`
	Message            string `json:"message,omitempty" yaml:"message,omitempty"`
}

const (
	ConditionReady             = "Ready"
	ConditionReconciling       = "Reconciling"
	ConditionDegraded          = "Degraded"
	ConditionPolicyBlocked     = "PolicyBlocked"
	ConditionApprovalPending   = "ApprovalPending"
	ConditionValidated         = "Validated"
	ConditionRollbackAvailable = "RollbackAvailable"
)

const (
	PhasePending     = "Pending"
	PhaseRunning     = "Running"
	PhaseReconciling = "Reconciling"
	PhaseDegraded    = "Degraded"
	PhaseFailed      = "Failed"
)

func NewManagedCluster(name string, namespace string, environment string) ManagedCluster {
	return ManagedCluster{
		TypeMeta: TypeMeta{APIVersion: GroupName + "/" + Version, Kind: KindManagedCluster},
		ObjectMeta: ObjectMeta{Name: name, Namespace: namespace, Generation: 1},
		Spec: ManagedClusterSpec{Environment: environment},
	}
}

func NewMachineClass(name string, provider string) MachineClass {
	return MachineClass{
		TypeMeta: TypeMeta{APIVersion: GroupName + "/" + Version, Kind: KindMachineClass},
		ObjectMeta: ObjectMeta{Name: name},
		Spec: MachineClassSpec{Provider: provider},
	}
}
