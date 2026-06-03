package controller

import (
	"context"
	"fmt"

	infraapi "github.com/tommyxie2026-tech/aicloud/infra/api"
)

// FakeController is a deterministic in-memory reconciler for the first
// ManagedCluster scenario. It has no external side effects.
type FakeController struct {
	store *FakeStateStore
}

func NewFakeController() *FakeController {
	return &FakeController{store: NewFakeStateStore()}
}

func NewFakeControllerWithStore(store *FakeStateStore) *FakeController {
	if store == nil {
		store = NewFakeStateStore()
	}
	return &FakeController{store: store}
}

// Reconcile validates the cluster and moves status one deterministic step toward desired state.
func (c *FakeController) Reconcile(ctx context.Context, cluster infraapi.ManagedCluster) (infraapi.ManagedCluster, error) {
	if err := infraapi.ValidateManagedCluster(cluster); err != nil {
		return cluster, err
	}

	key := NamespacedName(cluster.Namespace, cluster.Name)
	current, exists := c.store.Get(key)
	if exists {
		cluster.Status = current.Status
	}

	desiredReady := desiredWorkerReplicas(cluster)
	ready := cluster.Status.WorkerReadyReplicas

	if ready < desiredReady {
		ready++
		cluster.Status.WorkerReadyReplicas = ready
		cluster.Status.Phase = infraapi.PhaseReconciling
		cluster.Status.Conditions = []infraapi.Condition{
			{Type: infraapi.ConditionReady, Status: "False", ObservedGeneration: cluster.Generation, Reason: "ScalingWorkers", Message: "workers are scaling toward desired replicas"},
			{Type: infraapi.ConditionReconciling, Status: "True", ObservedGeneration: cluster.Generation, Reason: "DesiredStateChanged", Message: "desired state has not been fully reconciled"},
		}
		c.store.Set(key, cluster)
		return cluster, nil
	}

	if ready > desiredReady {
		ready--
		cluster.Status.WorkerReadyReplicas = ready
		cluster.Status.Phase = infraapi.PhaseReconciling
		cluster.Status.Conditions = []infraapi.Condition{
			{Type: infraapi.ConditionReady, Status: "False", ObservedGeneration: cluster.Generation, Reason: "ScalingWorkers", Message: "workers are scaling toward desired replicas"},
			{Type: infraapi.ConditionReconciling, Status: "True", ObservedGeneration: cluster.Generation, Reason: "DesiredStateChanged", Message: "desired state has not been fully reconciled"},
		}
		c.store.Set(key, cluster)
		return cluster, nil
	}

	cluster.Status.Phase = infraapi.PhaseRunning
	cluster.Status.ObservedGeneration = cluster.Generation
	cluster.Status.WorkerReadyReplicas = desiredReady
	cluster.Status.Conditions = []infraapi.Condition{
		{Type: infraapi.ConditionReady, Status: "True", ObservedGeneration: cluster.Generation, Reason: "WorkersReady", Message: "all worker groups are ready"},
		{Type: infraapi.ConditionReconciling, Status: "False", ObservedGeneration: cluster.Generation, Reason: "ReconcileComplete", Message: "desired state has been reconciled"},
	}
	c.store.Set(key, cluster)
	return cluster, nil
}

func desiredWorkerReplicas(cluster infraapi.ManagedCluster) int32 {
	var total int32
	for _, worker := range cluster.Spec.Workers {
		total += worker.Replicas
	}
	return total
}

func NamespacedName(namespace string, name string) string {
	if namespace == "" {
		return name
	}
	return namespace + "/" + name
}

// FakeStateStore is an in-memory state store for fake controller tests and demos.
type FakeStateStore struct {
	clusters map[string]infraapi.ManagedCluster
}

func NewFakeStateStore() *FakeStateStore {
	return &FakeStateStore{clusters: map[string]infraapi.ManagedCluster{}}
}

func (s *FakeStateStore) Get(key string) (infraapi.ManagedCluster, bool) {
	cluster, ok := s.clusters[key]
	return cluster, ok
}

func (s *FakeStateStore) Set(key string, cluster infraapi.ManagedCluster) {
	s.clusters[key] = cluster
}

func (s *FakeStateStore) MustGet(key string) infraapi.ManagedCluster {
	cluster, ok := s.Get(key)
	if !ok {
		panic(fmt.Sprintf("cluster %s not found", key))
	}
	return cluster
}
