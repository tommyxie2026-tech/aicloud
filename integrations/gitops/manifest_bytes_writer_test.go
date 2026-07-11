package gitops

import (
	"testing"

	"github.com/tommyxie2026-tech/aicloud/integrations/gitops/yamlio"
)

func TestManagedClusterManifestBytesWriterPatchesAndRenders(t *testing.T) {
	writer := NewManagedClusterManifestBytesWriter(NewDryRunManifestWriter())
	plan := validPatchPlan(3, 6)

	result, err := writer.WriteManagedClusterBytes(plan, []byte(validManagedClusterManifestYAML()))
	if err != nil {
		t.Fatalf("WriteManagedClusterBytes returned error: %v", err)
	}
	if result == nil || result.WriteResult == nil {
		t.Fatalf("expected result and write result")
	}
	if len(result.Manifest) == 0 {
		t.Fatalf("expected rendered manifest bytes")
	}
	parsed, err := yamlio.ReadManagedCluster(result.Manifest)
	if err != nil {
		t.Fatalf("rendered manifest did not parse: %v", err)
	}
	if parsed.Spec.Workers[0].Replicas != 6 {
		t.Fatalf("expected rendered replicas 6, got %d", parsed.Spec.Workers[0].Replicas)
	}
	if result.WriteResult.Updated.Spec.Workers[0].Replicas != 6 {
		t.Fatalf("expected write result replicas 6, got %d", result.WriteResult.Updated.Spec.Workers[0].Replicas)
	}
}

func TestManagedClusterManifestBytesWriterRejectsInvalidInputBytes(t *testing.T) {
	writer := NewManagedClusterManifestBytesWriter(NewDryRunManifestWriter())
	_, err := writer.WriteManagedClusterBytes(validPatchPlan(3, 6), []byte("metadata: ["))
	assertGitOpsError(t, err, "ReadManagedClusterManifestFailed")
}

func TestManagedClusterManifestBytesWriterPropagatesPatchFailure(t *testing.T) {
	writer := NewManagedClusterManifestBytesWriter(NewDryRunManifestWriter())
	_, err := writer.WriteManagedClusterBytes(validPatchPlan(4, 6), []byte(validManagedClusterManifestYAML()))
	if err == nil {
		t.Fatalf("expected patch failure")
	}
}

func TestManagedClusterManifestBytesWriterUsesDefaultObjectWriter(t *testing.T) {
	writer := NewManagedClusterManifestBytesWriter(nil)
	result, err := writer.WriteManagedClusterBytes(validPatchPlan(3, 6), []byte(validManagedClusterManifestYAML()))
	if err != nil {
		t.Fatalf("WriteManagedClusterBytes returned error: %v", err)
	}
	if result.WriteResult.Updated.Spec.Workers[0].Replicas != 6 {
		t.Fatalf("expected default object writer to patch replicas to 6")
	}
}

func validManagedClusterManifestYAML() string {
	return "apiVersion: infra.aicloud.dev/v1alpha1\nkind: ManagedCluster\nmetadata:\n  name: dev-gpu-cluster\n  namespace: default\nspec:\n  environment: dev\n  workers:\n    - name: gpu-workers\n      replicas: 3\n      machineClassRef:\n        name: gpu-large\n"
}
