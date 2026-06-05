package controller

import (
	"context"
	"fmt"

	infraadapter "github.com/tommyxie2026-tech/aicloud/infra/adapter"
	infraapi "github.com/tommyxie2026-tech/aicloud/infra/api"
)

// FakeController is a deterministic in-memory reconciler for the first
// ManagedCluster scenario. It has no external side effects.
type FakeController struct {
	store   *FakeStateStore
	adapter infraadapter.ClusterAdapter
}

func NewFakeController() *FakeController {
	return &FakeController{store: NewFakeStateStore(), adapter: infraadapter.NewFakeClusterAdapter()}
}

func NewFakeControllerWithStore(store *FakeStateStore) *FakeController {
	if store == nil {
		store = NewFakeStateStore()
	}
	return &FakeController{store: store, adapter: infraadapter.NewFakeClusterAdapter()}
}

func NewFakeControllerWithAdapter(store *FakeStateStore, adapter infraadapter.ClusterAdapter) *FakeController {
	if store == nil {
		store = NewFakeStateStore()
	}
	if adapter == nil {
		adapter = infraadapter.NewFakeClusterAdapter()
	}
	return &FakeController{store: store, adapter: adapter}
}

// Reconcile validates the cluster, delegates backend state movement to adapter,
// and summarizes observed state into ManagedCluster.status.
func (c *FakeController) Reconcile(ctx context.Context, cluster infraapi.ManagedCluster) (infraapi.ManagedCluster, error) {
	if err := infraapi.ValidateManagedCluster(cluster); err != nil {
		return cluster, err
	}

	key := NamespacedName(cluster.Namespace, cluster.Name)
	if current, exists := c.store.Get(key); exists {
		cluster.Status = current.Status
	}

	if err := c.adapter.ApplyDesiredState(ctx, cluster); err != nil {
		cluster.Status.Phase = infraapi.PhaseDegraded
		cluster.Status.Conditions = []infraapi.Condition{
			{Type: infraapi.ConditionReady, Status: "False", ObservedGeneration: cluster.Generation, Reason: "AdapterError", Message: err.Error()},
			{Type: infraapi.ConditionDegraded, Status: "True", ObservedGeneration: cluster.Generation, Reason: "AdapterError", Message: err.Error()},
		}
		c.store.Set(key, cluster)
		return cluster, err
	}

	observed, err := c.adapter.Observe(ctx, cluster)
	if err != nil {
		cluster.Status.Phase = infraapi.PhaseDegraded
		cluster.Status.Conditions = []infraapi.Condition{
			{Type: infraapi.ConditionReady, Status: "False", ObservedGeneration: cluster.Generation, Reason: "ObserveError", Message: err.Error()},
			{Type: infraapi.ConditionDegraded, Status: "True", ObservedGeneration: cluster.Generation, Reason: "ObserveError", Message: err.Error()},
		}
		c.store.Set(key, cluster)
		return cluster, err
	}

	desiredReady := desiredWorkerReplicas(cluster)
	cluster.Status.WorkerReadyReplicas = observed.ReadyReplicas

	if observed.ReadyReplicas == desiredReady {
		cluster.Status.Phase = infraapi.PhaseRunning
		cluster.Status.ObservedGeneration = cluster.Generation
		cluster.Status.Conditions = []infraapi.Condition{
			{Type: infraapi.ConditionReady, Status: "True", ObservedGeneration: cluster.Generation, Reason: "WorkersReady", Message: "all worker groups are ready"},
			{Type: infraapi.ConditionReconciling, Status: "False", ObservedGeneration: cluster.Generation, Reason: "ReconcileComplete", Message: "desired state has been reconciled"},
		}
		c.store.Set(key, cluster)
		return cluster, nil
	}

	cluster.Status.Phase = infraapi.PhaseReconciling
	cluster.Status.Conditions = []infraapi.Condition{
		{Type: infraapi.ConditionReady, Status: "False", ObservedGeneration: cluster.Generation, Reason: "ScalingWorkers", Message: "workers are scaling toward desired replicas"},
		{Type: infraapi.ConditionReconciling, Status: "True", ObservedGeneration: cluster.Generation, Reason: "DesiredStateChanged", Message: "desired state has not been fully reconciled"},
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
