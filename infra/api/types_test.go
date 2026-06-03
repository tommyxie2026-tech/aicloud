package api

import "testing"

func TestNewManagedCluster(t *testing.T) {
	cluster := NewManagedCluster("dev-gpu-cluster", "default", "dev")

	if cluster.APIVersion != "infra.aicloud.dev/v1alpha1" {
		t.Fatalf("unexpected apiVersion: %s", cluster.APIVersion)
	}
	if cluster.Kind != KindManagedCluster {
		t.Fatalf("unexpected kind: %s", cluster.Kind)
	}
	if cluster.Name != "dev-gpu-cluster" {
		t.Fatalf("unexpected name: %s", cluster.Name)
	}
	if cluster.Namespace != "default" {
		t.Fatalf("unexpected namespace: %s", cluster.Namespace)
	}
	if cluster.Spec.Environment != "dev" {
		t.Fatalf("unexpected environment: %s", cluster.Spec.Environment)
	}
}

func TestManagedClusterWorkerGroup(t *testing.T) {
	cluster := NewManagedCluster("dev-gpu-cluster", "default", "dev")
	cluster.Spec.Workers = []WorkerGroupSpec{
		{Name: "gpu-workers", Replicas: 3, MachineClassRef: LocalObjectReference{Name: "gpu-large"}},
	}

	if len(cluster.Spec.Workers) != 1 {
		t.Fatalf("expected one worker group, got %d", len(cluster.Spec.Workers))
	}
	if cluster.Spec.Workers[0].Replicas != 3 {
		t.Fatalf("expected replicas 3, got %d", cluster.Spec.Workers[0].Replicas)
	}
	if cluster.Spec.Workers[0].MachineClassRef.Name != "gpu-large" {
		t.Fatalf("expected gpu-large, got %s", cluster.Spec.Workers[0].MachineClassRef.Name)
	}
}

func TestNewMachineClass(t *testing.T) {
	machineClass := NewMachineClass("gpu-large", "internal-cloud")
	machineClass.Spec.CPU = "32"
	machineClass.Spec.Memory = "128Gi"
	machineClass.Spec.GPU = &GPUSpec{Count: 4, Type: "A100"}

	if machineClass.APIVersion != "infra.aicloud.dev/v1alpha1" {
		t.Fatalf("unexpected apiVersion: %s", machineClass.APIVersion)
	}
	if machineClass.Kind != KindMachineClass {
		t.Fatalf("unexpected kind: %s", machineClass.Kind)
	}
	if machineClass.Spec.GPU == nil || machineClass.Spec.GPU.Count != 4 {
		t.Fatalf("expected gpu count 4, got %#v", machineClass.Spec.GPU)
	}
}

func TestConditionConstants(t *testing.T) {
	condition := Condition{Type: ConditionReady, Status: "True", ObservedGeneration: 1, Reason: "WorkersReady"}

	if condition.Type != "Ready" {
		t.Fatalf("unexpected condition type: %s", condition.Type)
	}
	if condition.Status != "True" {
		t.Fatalf("unexpected condition status: %s", condition.Status)
	}
}
