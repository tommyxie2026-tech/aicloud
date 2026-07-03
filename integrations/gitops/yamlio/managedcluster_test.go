package yamlio

import (
	"strings"
	"testing"

	"github.com/tommyxie2026-tech/aicloud/infra/api"
)

func TestReadManagedCluster(t *testing.T) {
	cluster, err := ReadManagedCluster([]byte(validManagedClusterYAML()))
	if err != nil {
		t.Fatalf("ReadManagedCluster returned error: %v", err)
	}
	if cluster.Name != "dev-gpu-cluster" {
		t.Fatalf("unexpected cluster name: %s", cluster.Name)
	}
	if cluster.Namespace != "aicloud-system" {
		t.Fatalf("unexpected namespace: %s", cluster.Namespace)
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
	if parsed.Name != original.Name {
		t.Fatalf("unexpected name after round trip: %s", parsed.Name)
	}
	if parsed.Spec.Workers[0].Replicas != original.Spec.Workers[0].Replicas {
		t.Fatalf("unexpected replicas after round trip: %d", parsed.Spec.Workers[0].Replicas)
	}
}

func TestWriteManagedClusterSortsLabelsDeterministically(t *testing.T) {
	cluster := validManagedCluster()
	cluster.Labels = map[string]string{
		"z.example/last":  "last",
		"a.example/first": "first",
	}
	data, err := WriteManagedCluster(cluster)
	if err != nil {
		t.Fatalf("WriteManagedCluster returned error: %v", err)
	}
	output := string(data)
	first := strings.Index(output, "a.example/first: first")
	last := strings.Index(output, "z.example/last: last")
	if first == -1 || last == -1 {
		t.Fatalf("expected both labels in output: %s", output)
	}
	if first > last {
		t.Fatalf("expected labels to be sorted, got: %s", output)
	}
	parsed, err := ReadManagedCluster(data)
	if err != nil {
		t.Fatalf("ReadManagedCluster after write returned error: %v", err)
	}
	if parsed.Labels["a.example/first"] != "first" || parsed.Labels["z.example/last"] != "last" {
		t.Fatalf("labels did not round trip: %+v", parsed.Labels)
	}
}

func TestWriteManagedClusterRejectsInvalidObject(t *testing.T) {
	cluster := validManagedCluster()
	cluster.Spec.Workers[0].Replicas = -1
	_, err := WriteManagedCluster(cluster)
	assertYAMLIOError(t, err, "InvalidManagedCluster")
}

func validManagedCluster() api.ManagedCluster {
	cluster := api.NewManagedCluster("dev-gpu-cluster", "aicloud-system", "dev")
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
