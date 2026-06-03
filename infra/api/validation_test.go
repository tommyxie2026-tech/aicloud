package api

import "testing"

func TestValidateManagedClusterPasses(t *testing.T) {
	cluster := NewManagedCluster("dev-gpu-cluster", "default", "dev")
	cluster.Spec.Workers = []WorkerGroupSpec{
		{Name: "gpu-workers", Replicas: 3, MachineClassRef: LocalObjectReference{Name: "gpu-large"}},
	}

	if err := ValidateManagedCluster(cluster); err != nil {
		t.Fatalf("expected valid ManagedCluster, got error: %v", err)
	}
}

func TestValidateManagedClusterRequiresName(t *testing.T) {
	cluster := NewManagedCluster("", "default", "dev")

	if err := ValidateManagedCluster(cluster); err == nil {
		t.Fatalf("expected missing name validation error")
	}
}

func TestValidateManagedClusterRequiresEnvironment(t *testing.T) {
	cluster := NewManagedCluster("dev-gpu-cluster", "default", "")

	if err := ValidateManagedCluster(cluster); err == nil {
		t.Fatalf("expected missing environment validation error")
	}
}

func TestValidateManagedClusterRejectsDuplicateWorkerNames(t *testing.T) {
	cluster := NewManagedCluster("dev-gpu-cluster", "default", "dev")
	cluster.Spec.Workers = []WorkerGroupSpec{
		{Name: "gpu-workers", Replicas: 3, MachineClassRef: LocalObjectReference{Name: "gpu-large"}},
		{Name: "gpu-workers", Replicas: 6, MachineClassRef: LocalObjectReference{Name: "gpu-large"}},
	}

	if err := ValidateManagedCluster(cluster); err == nil {
		t.Fatalf("expected duplicate worker name validation error")
	}
}

func TestValidateManagedClusterRejectsNegativeReplicas(t *testing.T) {
	cluster := NewManagedCluster("dev-gpu-cluster", "default", "dev")
	cluster.Spec.Workers = []WorkerGroupSpec{
		{Name: "gpu-workers", Replicas: -1, MachineClassRef: LocalObjectReference{Name: "gpu-large"}},
	}

	if err := ValidateManagedCluster(cluster); err == nil {
		t.Fatalf("expected negative replicas validation error")
	}
}

func TestValidateManagedClusterRequiresMachineClassRef(t *testing.T) {
	cluster := NewManagedCluster("dev-gpu-cluster", "default", "dev")
	cluster.Spec.Workers = []WorkerGroupSpec{
		{Name: "gpu-workers", Replicas: 3},
	}

	if err := ValidateManagedCluster(cluster); err == nil {
		t.Fatalf("expected missing machineClassRef validation error")
	}
}

func TestValidateMachineClassPasses(t *testing.T) {
	machineClass := NewMachineClass("gpu-large", "internal-cloud")
	machineClass.Spec.GPU = &GPUSpec{Count: 4, Type: "A100"}

	if err := ValidateMachineClass(machineClass); err != nil {
		t.Fatalf("expected valid MachineClass, got error: %v", err)
	}
}

func TestValidateMachineClassRequiresProvider(t *testing.T) {
	machineClass := NewMachineClass("gpu-large", "")

	if err := ValidateMachineClass(machineClass); err == nil {
		t.Fatalf("expected missing provider validation error")
	}
}

func TestValidateMachineClassRejectsNegativeGPUCount(t *testing.T) {
	machineClass := NewMachineClass("gpu-large", "internal-cloud")
	machineClass.Spec.GPU = &GPUSpec{Count: -1, Type: "A100"}

	if err := ValidateMachineClass(machineClass); err == nil {
		t.Fatalf("expected negative gpu count validation error")
	}
}

func TestValidateMachineClassRequiresGPUTypeWhenCountPositive(t *testing.T) {
	machineClass := NewMachineClass("gpu-large", "internal-cloud")
	machineClass.Spec.GPU = &GPUSpec{Count: 1}

	if err := ValidateMachineClass(machineClass); err == nil {
		t.Fatalf("expected missing gpu type validation error")
	}
}
