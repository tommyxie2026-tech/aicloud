package controller

import (
	"context"
	"testing"

	infraapi "github.com/tommyxie2026-tech/aicloud/infra/api"
)

func TestFakeControllerReconcileAlreadyReady(t *testing.T) {
	controller := NewFakeController()
	cluster := validCluster(3)
	cluster.Status.WorkerReadyReplicas = 3

	updated, err := controller.Reconcile(context.Background(), cluster)
	if err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}
	if updated.Status.Phase != infraapi.PhaseRunning {
		t.Fatalf("expected Running, got %s", updated.Status.Phase)
	}
	if updated.Status.ObservedGeneration != cluster.Generation {
		t.Fatalf("expected observedGeneration %d, got %d", cluster.Generation, updated.Status.ObservedGeneration)
	}
	assertCondition(t, updated, infraapi.ConditionReady, "True")
	assertCondition(t, updated, infraapi.ConditionReconciling, "False")
}

func TestFakeControllerReconcileScaleOutOneStep(t *testing.T) {
	controller := NewFakeController()
	cluster := validCluster(6)
	cluster.Status.WorkerReadyReplicas = 3
	cluster.Generation = 2

	updated, err := controller.Reconcile(context.Background(), cluster)
	if err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}
	if updated.Status.WorkerReadyReplicas != 4 {
		t.Fatalf("expected ready replicas 4, got %d", updated.Status.WorkerReadyReplicas)
	}
	if updated.Status.Phase != infraapi.PhaseReconciling {
		t.Fatalf("expected Reconciling, got %s", updated.Status.Phase)
	}
	assertCondition(t, updated, infraapi.ConditionReady, "False")
	assertCondition(t, updated, infraapi.ConditionReconciling, "True")
}

func TestFakeControllerReconcileScaleOutToReady(t *testing.T) {
	controller := NewFakeController()
	cluster := validCluster(6)
	cluster.Status.WorkerReadyReplicas = 3
	cluster.Generation = 2

	var updated infraapi.ManagedCluster
	var err error
	for i := 0; i < 4; i++ {
		updated, err = controller.Reconcile(context.Background(), cluster)
		if err != nil {
			t.Fatalf("Reconcile returned error: %v", err)
		}
		cluster = updated
	}

	if updated.Status.WorkerReadyReplicas != 6 {
		t.Fatalf("expected ready replicas 6, got %d", updated.Status.WorkerReadyReplicas)
	}
	if updated.Status.Phase != infraapi.PhaseRunning {
		t.Fatalf("expected Running, got %s", updated.Status.Phase)
	}
	if updated.Status.ObservedGeneration != 2 {
		t.Fatalf("expected observedGeneration 2, got %d", updated.Status.ObservedGeneration)
	}
	assertCondition(t, updated, infraapi.ConditionReady, "True")
	assertCondition(t, updated, infraapi.ConditionReconciling, "False")
}

func TestFakeControllerReconcileScaleDownOneStep(t *testing.T) {
	controller := NewFakeController()
	cluster := validCluster(3)
	cluster.Status.WorkerReadyReplicas = 6
	cluster.Generation = 3

	updated, err := controller.Reconcile(context.Background(), cluster)
	if err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}
	if updated.Status.WorkerReadyReplicas != 5 {
		t.Fatalf("expected ready replicas 5, got %d", updated.Status.WorkerReadyReplicas)
	}
	if updated.Status.Phase != infraapi.PhaseReconciling {
		t.Fatalf("expected Reconciling, got %s", updated.Status.Phase)
	}
}

func TestFakeControllerRejectsInvalidCluster(t *testing.T) {
	controller := NewFakeController()
	cluster := validCluster(3)
	cluster.Name = ""

	_, err := controller.Reconcile(context.Background(), cluster)
	if err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestFakeStateStore(t *testing.T) {
	store := NewFakeStateStore()
	cluster := validCluster(3)
	key := NamespacedName(cluster.Namespace, cluster.Name)

	store.Set(key, cluster)
	got, ok := store.Get(key)
	if !ok {
		t.Fatalf("expected stored cluster")
	}
	if got.Name != cluster.Name {
		t.Fatalf("expected %s, got %s", cluster.Name, got.Name)
	}
}

func validCluster(replicas int32) infraapi.ManagedCluster {
	cluster := infraapi.NewManagedCluster("dev-gpu-cluster", "default", "dev")
	cluster.Spec.Workers = []infraapi.WorkerGroupSpec{
		{Name: "gpu-workers", Replicas: replicas, MachineClassRef: infraapi.LocalObjectReference{Name: "gpu-large"}},
	}
	return cluster
}

func assertCondition(t *testing.T, cluster infraapi.ManagedCluster, conditionType string, status string) {
	t.Helper()
	for _, condition := range cluster.Status.Conditions {
		if condition.Type == conditionType {
			if condition.Status != status {
				t.Fatalf("expected condition %s=%s, got %s", conditionType, status, condition.Status)
			}
			return
		}
	}
	t.Fatalf("condition %s not found", conditionType)
}
