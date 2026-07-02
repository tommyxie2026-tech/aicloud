package metal3

import (
	"testing"

	"github.com/tommyxie2026-tech/aicloud/infra/api"
)

func TestMapManagedClusterToDesiredBareMetalHostClaims(t *testing.T) {
	cluster := validManagedCluster()
	classes := []api.MachineClass{validMachineClass("gpu-large")}
	result, err := NewMapper().MapManagedCluster(cluster, classes)
	if err != nil {
		t.Fatalf("MapManagedCluster returned error: %v", err)
	}
	if len(result.HostClaims) != 3 {
		t.Fatalf("expected 3 host claims, got %d", len(result.HostClaims))
	}
	first := result.HostClaims[0]
	if first.Name != "dev-gpu-cluster-gpu-workers-host-0001" {
		t.Fatalf("unexpected host claim name: %s", first.Name)
	}
	if first.Ordinal != 1 {
		t.Fatalf("unexpected ordinal: %d", first.Ordinal)
	}
	if first.CPU != "8" || first.Memory != "32Gi" || first.GPUProfile != "1xnvidia-a10" {
		t.Fatalf("unexpected hardware profile: %+v", first)
	}
	if first.Labels["aicloud.dev/machine-class"] != "gpu-large" {
		t.Fatalf("missing machine class label")
	}
}

func TestMapManagedClusterRejectsMissingMachineClass(t *testing.T) {
	cluster := validManagedCluster()
	_, err := NewMapper().MapManagedCluster(cluster, nil)
	if err == nil {
		t.Fatalf("expected missing machine class error")
	}
}

func TestMapManagedClusterRejectsInvalidManagedCluster(t *testing.T) {
	cluster := validManagedCluster()
	cluster.Spec.Workers[0].Replicas = -1
	_, err := NewMapper().MapManagedCluster(cluster, []api.MachineClass{validMachineClass("gpu-large")})
	if err == nil {
		t.Fatalf("expected invalid managed cluster error")
	}
}

func TestMapManagedClusterMultipleWorkerGroups(t *testing.T) {
	cluster := validManagedCluster()
	cluster.Spec.Workers = append(cluster.Spec.Workers, api.WorkerGroupSpec{Name: "cpu-workers", Replicas: 2, MachineClassRef: api.LocalObjectReference{Name: "cpu-large"}})
	classes := []api.MachineClass{validMachineClass("gpu-large"), validMachineClass("cpu-large")}
	result, err := NewMapper().MapManagedCluster(cluster, classes)
	if err != nil {
		t.Fatalf("MapManagedCluster returned error: %v", err)
	}
	if len(result.HostClaims) != 5 {
		t.Fatalf("expected 5 host claims, got %d", len(result.HostClaims))
	}
	last := result.HostClaims[4]
	if last.Name != "dev-gpu-cluster-cpu-workers-host-0002" {
		t.Fatalf("unexpected last host claim name: %s", last.Name)
	}
}

func validManagedCluster() api.ManagedCluster {
	cluster := api.NewManagedCluster("dev-gpu-cluster", "aicloud-system", "dev")
	cluster.Spec.Workers = []api.WorkerGroupSpec{{Name: "gpu-workers", Replicas: 3, MachineClassRef: api.LocalObjectReference{Name: "gpu-large"}}}
	return cluster
}

func validMachineClass(name string) api.MachineClass {
	class := api.NewMachineClass(name, "metal3")
	class.Spec.CPU = "8"
	class.Spec.Memory = "32Gi"
	class.Spec.GPU = &api.GPUSpec{Count: 1, Type: "nvidia-a10"}
	return class
}
