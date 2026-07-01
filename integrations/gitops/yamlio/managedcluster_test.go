package yamlio

import (
	"testing"

	"github.com/tommyxie2026-tech/aicloud/infra/api"
)

func TestReadManagedCluster(t *testing.T) {
	cluster, err := ReadManagedCluster([]byte(validManagedClusterYAML()))
	if err != nil {
		t.Fatalf("ReadManagedCluster returned error: %v", err)
	}
	if cluster.Metadata.Name != "dev-gpu-cluster" {
		t.Fatalf("unexpected cluster name: %s", cluster.Metadata.Name)
	}
	if cluster.Metadata.Namespace != "aicloud-system" {
		t.Fatalf("unexpected namespace: %s", cluster.Metadata.Namespace)
	}
	if cluster.Spec.Environment != "dev" {
		t.Fatalf("unexpected environment: %s", cluster.Spec.Environment)
	}
	if len(cluster.Spec.Workers) != 1 {
		t.Fatalf("expected 1 worker, got %d", len(cluster.Spec.Workers))
	}
	worker := cluster.Spec.Workers[0]
	if worker.Name != "gpu-workers" || worker.Replicas != 3 || worker.MachineClassRef.Name != "gpu-large" {
		t.Fatalf("unexpected worker: %+v", worker)
	}
}

func TestReadManagedClusterRejectsEmptyInput(t *testing.T) {
	_, err := ReadManagedCluster(nil)
	assertYAMLIOError(t, err, "EmptyInput")
}

func TestReadManagedClusterRejectsInvalidYAML(t *testing.T) {
	_, err := ReadManagedCluster([]byte("metadata: ["))
	assertYAMLIOError(t, err, "InvalidYAML")
}

func TestReadManagedClusterRejectsInvalidObject(t *testing.T) {
	_, err := ReadManagedCluster([]byte("apiVersion: infra.aicloud.dev/v1alpha1\nkind: ManagedCluster\nmetadata:\n  name: dev\n  namespace: aicloud-system\nspec:\n  environment: dev\n  workers:\n    - name: gpu-workers\n      replicas: -1\n      machineClassRef:\n        name: gpu-large\n"))
	assertYAMLIOError(t, err, "InvalidManagedCluster")
}

func TestWriteManagedClusterRoundTrip(t *testing.T) {
	original := validManagedCluster()
	data, err := WriteManagedCluster(original)
	if err != nil {
		t.Fatalf("WriteManagedCluster returned error: %v", err)
	}
	parsed, err := ReadManagedCluster(data)
	if err != nil {
		t.Fatalf("ReadManagedCluster after write returned error: %v", err)
	}
	if parsed.Metadata.Name != original.Metadata.Name {
		t.Fatalf("unexpected name after round trip: %s", parsed.Metadata.Name)
	}
	if parsed.Spec.Workers[0].Replicas != original.Spec.Workers[0].Replicas {
		t.Fatalf("unexpected replicas after round trip: %d", parsed.Spec.Workers[0].Replicas)
	}
}

func TestWriteManagedClusterRejectsInvalidObject(t *testing.T) {
	cluster := validManagedCluster()
	cluster.Spec.Workers[0].Replicas = -1
	_, err := WriteManagedCluster(cluster)
	assertYAMLIOError(t, err, "InvalidManagedCluster")
}

func validManagedCluster() api.ManagedCluster {
	cluster := api.NewManagedCluster("dev-gpu-cluster", "aicloud-system")
	cluster.Spec.Environment = "dev"
	cluster.Spec.Workers = []api.WorkerGroupSpec{{Name: "gpu-workers", Replicas: 3, MachineClassRef: api.LocalObjectReference{Name: "gpu-large"}}}
	return cluster
}

func validManagedClusterYAML() string {
	return "apiVersion: infra.aicloud.dev/v1alpha1\nkind: ManagedCluster\nmetadata:\n  name: dev-gpu-cluster\n  namespace: aicloud-system\n  labels:\n    aicloud.dev/environment: dev\nspec:\n  environment: dev\n  workers:\n    - name: gpu-workers\n      replicas: 3\n      machineClassRef:\n        name: gpu-large\n"
}

func assertYAMLIOError(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %s", code)
	}
	yamlErr, ok := err.(*YAMLIOError)
	if !ok {
		t.Fatalf("expected YAMLIOError, got %T", err)
	}
	if yamlErr.Code != code {
		t.Fatalf("expected code %s, got %s", code, yamlErr.Code)
	}
}
