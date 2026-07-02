package clusterapi

import (
	"testing"

	"github.com/tommyxie2026-tech/aicloud/infra/api"
)

func TestMapManagedCluster(t *testing.T) {
	cluster := validManagedCluster()
	result, err := NewMapper().MapManagedCluster(cluster)
	if err != nil {
		t.Fatalf("MapManagedCluster returned error: %v", err)
	}
	if result.Cluster.Name != "dev-gpu-cluster" {
		t.Fatalf("unexpected cluster name: %s", result.Cluster.Name)
	}
	if result.Cluster.Labels["aicloud.dev/managed-by"] != "aicloud" {
		t.Fatalf("missing managed-by label")
	}
	if len(result.MachineDeployments) != 1 {
		t.Fatalf("expected 1 machine deployment, got %d", len(result.MachineDeployments))
	}
	md := result.MachineDeployments[0]
	if md.Name != "dev-gpu-cluster-gpu-workers" {
		t.Fatalf("unexpected machine deployment name: %s", md.Name)
	}
	if md.Replicas != 6 {
		t.Fatalf("unexpected replicas: %d", md.Replicas)
	}
	if md.MachineClassName != "gpu-large" {
		t.Fatalf("unexpected machine class: %s", md.MachineClassName)
	}
	if md.Labels["aicloud.dev/worker-group"] != "gpu-workers" {
		t.Fatalf("missing worker group label")
	}
}

func TestMapManagedClusterMultipleWorkers(t *testing.T) {
	cluster := validManagedCluster()
	cluster.Spec.Workers = append(cluster.Spec.Workers, api.WorkerGroupSpec{Name: "cpu-workers", Replicas: 2, MachineClassRef: api.LocalObjectReference{Name: "cpu-large"}})
	result, err := NewMapper().MapManagedCluster(cluster)
	if err != nil {
		t.Fatalf("MapManagedCluster returned error: %v", err)
	}
	if len(result.MachineDeployments) != 2 {
		t.Fatalf("expected 2 machine deployments, got %d", len(result.MachineDeployments))
	}
	if result.MachineDeployments[1].Name != "dev-gpu-cluster-cpu-workers" {
		t.Fatalf("unexpected second machine deployment: %s", result.MachineDeployments[1].Name)
	}
}

func TestMapManagedClusterRejectsInvalidCluster(t *testing.T) {
	cluster := validManagedCluster()
	cluster.Spec.Workers[0].Replicas = -1
	_, err := NewMapper().MapManagedCluster(cluster)
	if err == nil {
		t.Fatalf("expected invalid cluster error")
	}
}

func TestMachineDeploymentPatchPath(t *testing.T) {
	path := MachineDeploymentPatchPath("Dev_GPU.Cluster", "gpu/workers")
	expected := "clusters/dev-gpu-cluster/machinedeployments/dev-gpu-cluster-gpu-workers.yaml"
	if path != expected {
		t.Fatalf("unexpected path: %s", path)
	}
}

func validManagedCluster() api.ManagedCluster {
	cluster := api.NewManagedCluster("dev-gpu-cluster", "aicloud-system", "dev")
	cluster.Spec.Workers = []api.WorkerGroupSpec{{Name: "gpu-workers", Replicas: 6, MachineClassRef: api.LocalObjectReference{Name: "gpu-large"}}}
	return cluster
}
