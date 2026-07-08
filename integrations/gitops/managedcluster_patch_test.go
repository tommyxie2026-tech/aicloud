package gitops

import (
	"testing"

	infraapi "github.com/tommyxie2026-tech/aicloud/infra/api"
	"github.com/tommyxie2026-tech/aicloud/model/schema"
)

func TestApplyManagedClusterPatch(t *testing.T) {
	cluster := validManagedCluster(3)
	plan := validPatchPlan(3, 6)

	updated, err := ApplyManagedClusterPatch(plan, cluster)
	if err != nil {
		t.Fatalf("ApplyManagedClusterPatch returned error: %v", err)
	}
	if updated.Spec.Workers[0].Replicas != 6 {
		t.Fatalf("expected replicas 6, got %d", updated.Spec.Workers[0].Replicas)
	}
	if cluster.Spec.Workers[0].Replicas != 3 {
		t.Fatalf("original cluster should not be mutated")
	}
}

func TestApplyManagedClusterPatchRejectsCurrentValueMismatch(t *testing.T) {
	cluster := validManagedCluster(4)
	plan := validPatchPlan(3, 6)

	_, err := ApplyManagedClusterPatch(plan, cluster)
	if err == nil {
		t.Fatalf("expected current value mismatch error")
	}
}

func TestApplyManagedClusterPatchRejectsTargetNameMismatch(t *testing.T) {
	cluster := validManagedCluster(3)
	plan := validPatchPlan(3, 6)
	plan.Target.Name = "other-cluster"

	_, err := ApplyManagedClusterPatch(plan, cluster)
	if err == nil {
		t.Fatalf("expected target name mismatch error")
	}
}

func TestApplyManagedClusterPatchRejectsUnsupportedField(t *testing.T) {
	cluster := validManagedCluster(3)
	plan := validPatchPlan(3, 6)
	plan.Changes[0].Field = "spec.network.cidr"

	_, err := ApplyManagedClusterPatch(plan, cluster)
	if err == nil {
		t.Fatalf("expected unsupported field error")
	}
}

func TestApplyManagedClusterPatchRejectsMissingWorkerGroup(t *testing.T) {
	cluster := validManagedCluster(3)
	cluster.Spec.Workers[0].Name = "cpu-workers"
	plan := validPatchPlan(3, 6)

	_, err := ApplyManagedClusterPatch(plan, cluster)
	if err == nil {
		t.Fatalf("expected missing worker group error")
	}
}

func TestApplyManagedClusterPatchRejectsNegativeReplicas(t *testing.T) {
	cluster := validManagedCluster(3)
	plan := validPatchPlan(3, -1)

	_, err := ApplyManagedClusterPatch(plan, cluster)
	if err == nil {
		t.Fatalf("expected negative replicas error")
	}
}

func TestApplyManagedClusterPatchRejectsFractionalReplicaValue(t *testing.T) {
	cluster := validManagedCluster(3)
	plan := validPatchPlan(3, 6.5)

	_, err := ApplyManagedClusterPatch(plan, cluster)
	assertGitOpsError(t, err, "InvalidToValue")
}

func TestApplyManagedClusterPatchRejectsReplicaOverflow(t *testing.T) {
	cluster := validManagedCluster(3)
	plan := validPatchPlan(3, int64(2147483648))

	_, err := ApplyManagedClusterPatch(plan, cluster)
	assertGitOpsError(t, err, "InvalidToValue")
}

func validManagedCluster(replicas int32) infraapi.ManagedCluster {
	cluster := infraapi.NewManagedCluster("dev-gpu-cluster", "default", "dev")
	cluster.Spec.Workers = []infraapi.WorkerGroupSpec{
		{Name: "gpu-workers", Replicas: replicas, MachineClassRef: infraapi.LocalObjectReference{Name: "gpu-large"}},
	}
	return cluster
}

func validPatchPlan(from any, to any) ManifestPatchPlan {
	return ManifestPatchPlan{
		RequestID:  "request-001",
		ProposalID: "proposal-001",
		Target:    schema.ResourceRef{Kind: infraapi.KindManagedCluster, Namespace: "default", Name: "dev-gpu-cluster"},
		SourcePath: "examples/infra/managedcluster-dev-gpu.yaml",
		OutputPath: "examples/infra/managedcluster-dev-gpu.yaml",
		Changes: []ManifestFieldChange{
			{Field: managedClusterWorkerReplicasField, From: from, To: to, Reason: "scale out"},
		},
	}
}

func assertGitOpsError(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %s", code)
	}
	gitopsErr, ok := err.(*GitOpsError)
	if !ok {
		t.Fatalf("expected GitOpsError, got %T", err)
	}
	if gitopsErr.Code != code {
		t.Fatalf("expected code %s, got %s", code, gitopsErr.Code)
	}
}
