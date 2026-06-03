package adapter

import (
	"context"
	"fmt"

	infraapi "github.com/tommyxie2026-tech/aicloud/infra/api"
)

// ClusterAdapter is the backend boundary for infrastructure reconciliation.
//
// Production implementations may map ManagedCluster to Cluster API, KubeVirt,
// Metal3, Crossplane, or an internal cloud platform. The MVP uses FakeClusterAdapter.
type ClusterAdapter interface {
	Observe(ctx context.Context, cluster infraapi.ManagedCluster) (ObservedClusterState, error)
	ApplyDesiredState(ctx context.Context, cluster infraapi.ManagedCluster) error
}

// ObservedClusterState is the backend-observed state normalized for controller logic.
type ObservedClusterState struct {
	ReadyReplicas int32
	Phase         string
	Conditions    []infraapi.Condition
}

// FakeClusterAdapter is a deterministic in-memory adapter for tests and demos.
// It has no external side effects.
type FakeClusterAdapter struct {
	state map[string]ObservedClusterState
}

func NewFakeClusterAdapter() *FakeClusterAdapter {
	return &FakeClusterAdapter{state: map[string]ObservedClusterState{}}
}

func (a *FakeClusterAdapter) Observe(ctx context.Context, cluster infraapi.ManagedCluster) (ObservedClusterState, error) {
	if err := infraapi.ValidateManagedCluster(cluster); err != nil {
		return ObservedClusterState{}, err
	}
	key := namespacedName(cluster.Namespace, cluster.Name)
	if observed, ok := a.state[key]; ok {
		return observed, nil
	}
	return ObservedClusterState{ReadyReplicas: cluster.Status.WorkerReadyReplicas, Phase: cluster.Status.Phase, Conditions: cluster.Status.Conditions}, nil
}

func (a *FakeClusterAdapter) ApplyDesiredState(ctx context.Context, cluster infraapi.ManagedCluster) error {
	if err := infraapi.ValidateManagedCluster(cluster); err != nil {
		return err
	}
	key := namespacedName(cluster.Namespace, cluster.Name)
	current, _ := a.Observe(ctx, cluster)
	desired := desiredReplicas(cluster)

	if current.ReadyReplicas < desired {
		current.ReadyReplicas++
		current.Phase = infraapi.PhaseReconciling
	} else if current.ReadyReplicas > desired {
		current.ReadyReplicas--
		current.Phase = infraapi.PhaseReconciling
	} else {
		current.Phase = infraapi.PhaseRunning
	}

	a.state[key] = current
	return nil
}

func (a *FakeClusterAdapter) SetObservedState(cluster infraapi.ManagedCluster, state ObservedClusterState) error {
	if cluster.Name == "" {
		return fmt.Errorf("cluster name is required")
	}
	a.state[namespacedName(cluster.Namespace, cluster.Name)] = state
	return nil
}

func desiredReplicas(cluster infraapi.ManagedCluster) int32 {
	var total int32
	for _, worker := range cluster.Spec.Workers {
		total += worker.Replicas
	}
	return total
}

func namespacedName(namespace string, name string) string {
	if namespace == "" {
		return name
	}
	return namespace + "/" + name
}
