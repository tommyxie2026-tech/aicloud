package gitops

import (
	"testing"

	"github.com/tommyxie2026-tech/aicloud/integrations/gitops/yamlio"
)

func TestManagedClusterManifestBytesWriterPatchesAndRendersBytes(t *testing.T) {
	writer := NewManagedClusterManifestBytesWriter(NewDryRunManifestWriter())
	result, err := writer.WriteManagedClusterBytes(validPatchPlan(3, 6), []byte(validManagedClusterManifestBytes()))
	if err != nil {
		t.Fatalf("WriteManagedClusterBytes returned error: %v", err)
	}
	if result == nil {
		t.Fatalf("expected result")
	}
	if result.WriteResult == nil {
		t.Fatalf("expected write result")
	}
	if len(result.Manifest) == 0 {
		t.Fatalf("expected rendered manifest bytes")
	}
	if result.WriteResult.Updated.Spec.Workers[0].Replicas != 6 {
		t.Fatalf("expected updated replicas 6, got %d", result.WriteResult.Updated.Spec.Workers[0].Replicas)
	}
	parsed, err := yamlio.ReadManagedCluster(result.Manifest)
	if err != nil {
		t.Fatalf("rendered manifest did not parse: %v", err)
	}
	if parsed.Spec.Workers[0].Replicas != 6 {
		t.Fatalf("expected rendered replicas 6, got %d", parsed.Spec.Workers[0].Replicas)
	}
}

func TestManagedClusterManifestBytesWriterRejectsInvalidInputBytes(t *testing.T) {
	writer := NewManagedClusterManifestBytesWriter(NewDryRunManifestWriter())
	_, err := writer.WriteManagedClusterBytes(validPatchPlan(3, 6), []byte("metadata: ["))
	if err == nil {
		t.Fatalf("expected invalid input error")
	}
}

func TestManagedClusterManifestBytesWriterPropagatesPatchErrors(t *testing.T) {
	writer := NewManagedClusterManifestBytesWriter(NewDryRunManifestWriter())
	_, err := writer.WriteManagedClusterBytes(validPatchPlan(4, 6), []byte(validManagedClusterManifestBytes()))
	assertGitOpsError(t, err, "CurrentValueMismatch")
}

func TestManagedClusterManifestBytesWriterRequiresObjectWriter(t *testing.T) {
	writer := &ManagedClusterManifestBytesWriter{}
	_, err := writer.WriteManagedClusterBytes(validPatchPlan(3, 6), []byte(validManagedClusterManifestBytes()))
	assertGitOpsError(t, err, "MissingObjectWriter")
}

func validManagedClusterManifestBytes() string {
	return "apiVersion: infra.aicloud.dev/v1alpha1\nkind: ManagedCluster\nmetadata:\n  name: dev-gpu-cluster\n  namespace: default\nspec:\n  environment: dev\n  workers:\n    - name: gpu-workers\n      replicas: 3\n      machineClassRef:\n        name: gpu-large\n"
}
