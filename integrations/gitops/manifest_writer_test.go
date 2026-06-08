package gitops

import "testing"

func TestDryRunManifestWriterWriteManagedCluster(t *testing.T) {
	writer := NewDryRunManifestWriter()
	cluster := validManagedCluster(3)
	plan := validPatchPlan(3, 6)

	result, err := writer.WriteManagedCluster(plan, cluster)
	if err != nil {
		t.Fatalf("WriteManagedCluster returned error: %v", err)
	}
	if result == nil {
		t.Fatalf("expected result")
	}
	if result.Updated.Spec.Workers[0].Replicas != 6 {
		t.Fatalf("expected replicas 6, got %d", result.Updated.Spec.Workers[0].Replicas)
	}
	if result.SourcePath != plan.SourcePath {
		t.Fatalf("unexpected source path: %s", result.SourcePath)
	}
	if result.OutputPath != plan.OutputPath {
		t.Fatalf("unexpected output path: %s", result.OutputPath)
	}
	if len(result.Changes) != 1 {
		t.Fatalf("expected one change, got %d", len(result.Changes))
	}
	if len(result.Rollback) != 0 {
		t.Fatalf("expected zero rollback entries because validPatchPlan does not set rollback, got %d", len(result.Rollback))
	}
	if result.Summary == "" {
		t.Fatalf("expected summary")
	}
}

func TestDryRunManifestWriterPropagatesPatchError(t *testing.T) {
	writer := NewDryRunManifestWriter()
	cluster := validManagedCluster(4)
	plan := validPatchPlan(3, 6)

	result, err := writer.WriteManagedCluster(plan, cluster)
	if err == nil {
		t.Fatalf("expected current value mismatch error")
	}
	if result != nil {
		t.Fatalf("expected nil result on error")
	}
}
