package gitops

import (
	"fmt"
	"math"

	infraapi "github.com/tommyxie2026-tech/aicloud/infra/api"
)

const managedClusterWorkerReplicasField = "spec.workers[name=gpu-workers].replicas"

// ApplyManagedClusterPatch applies a ManifestPatchPlan to a ManagedCluster object in memory.
// It does not write files, create commits, call kubectl, or talk to a live cluster.
func ApplyManagedClusterPatch(plan ManifestPatchPlan, cluster infraapi.ManagedCluster) (infraapi.ManagedCluster, error) {
	if err := infraapi.ValidateManagedCluster(cluster); err != nil {
		return cluster, err
	}
	if plan.Target.Kind != "" && plan.Target.Kind != infraapi.KindManagedCluster {
		return cluster, NewGitOpsError("TargetKindMismatch", fmt.Sprintf("expected target kind %s, got %s", infraapi.KindManagedCluster, plan.Target.Kind))
	}
	if plan.Target.Name != "" && plan.Target.Name != cluster.Name {
		return cluster, NewGitOpsError("TargetNameMismatch", fmt.Sprintf("plan targets %s but manifest is %s", plan.Target.Name, cluster.Name))
	}
	if plan.Target.Namespace != "" && plan.Target.Namespace != cluster.Namespace {
		return cluster, NewGitOpsError("TargetNamespaceMismatch", fmt.Sprintf("plan targets namespace %s but manifest is namespace %s", plan.Target.Namespace, cluster.Namespace))
	}
	if len(plan.Changes) == 0 {
		return cluster, NewGitOpsError("MissingChanges", "manifest patch plan changes must not be empty")
	}

	updated := cluster
	updated.Spec.Workers = append([]infraapi.WorkerGroupSpec(nil), cluster.Spec.Workers...)
	for _, change := range plan.Changes {
		if change.Field != managedClusterWorkerReplicasField {
			return cluster, NewGitOpsError("UnsupportedManagedClusterField", fmt.Sprintf("unsupported ManagedCluster patch field %s", change.Field))
		}
		from, ok := toInt32(change.From)
		if !ok {
			return cluster, NewGitOpsError("InvalidFromValue", fmt.Sprintf("from value for %s must be an integer", change.Field))
		}
		to, ok := toInt32(change.To)
		if !ok {
			return cluster, NewGitOpsError("InvalidToValue", fmt.Sprintf("to value for %s must be an integer", change.Field))
		}
		if to < 0 {
			return cluster, NewGitOpsError("InvalidReplicaValue", "replicas must be >= 0")
		}

		index := findWorkerGroup(updated, "gpu-workers")
		if index < 0 {
			return cluster, NewGitOpsError("WorkerGroupNotFound", "worker group gpu-workers not found")
		}
		current := updated.Spec.Workers[index].Replicas
		if current != from {
			return cluster, NewGitOpsError("CurrentValueMismatch", fmt.Sprintf("expected current replicas %d, got %d", from, current))
		}
		updated.Spec.Workers[index].Replicas = to
	}

	if err := infraapi.ValidateManagedCluster(updated); err != nil {
		return cluster, err
	}
	return updated, nil
}

func findWorkerGroup(cluster infraapi.ManagedCluster, name string) int {
	for i, worker := range cluster.Spec.Workers {
		if worker.Name == name {
			return i
		}
	}
	return -1
}

func toInt32(value any) (int32, bool) {
	switch v := value.(type) {
	case int:
		if v < math.MinInt32 || v > math.MaxInt32 {
			return 0, false
		}
		return int32(v), true
	case int32:
		return v, true
	case int64:
		if v < math.MinInt32 || v > math.MaxInt32 {
			return 0, false
		}
		return int32(v), true
	case float64:
		if math.Trunc(v) != v || v < math.MinInt32 || v > math.MaxInt32 {
			return 0, false
		}
		return int32(v), true
	case float32:
		f := float64(v)
		if math.Trunc(f) != f || f < math.MinInt32 || f > math.MaxInt32 {
			return 0, false
		}
		return int32(v), true
	default:
		return 0, false
	}
}
