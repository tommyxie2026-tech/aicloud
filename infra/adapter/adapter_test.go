package adapter

import (
	"context"
	"testing"

	infraapi "github.com/tommyxie2026-tech/aicloud/infra/api"
)

func TestFakeClusterAdapterObserveUsesClusterStatusWhenNoState(t *testing.T) {
	adapter := NewFakeClusterAdapter()
	cluster := validCluster(3)
	cluster.Status.WorkerReadyReplicas = 2
	cluster.Status.Phase = infraapi.PhaseReconciling

	observed, err := adapter.Observe(context.Background(), cluster)
	if err != nil {
		t.Fatalf("Observe returned error: %v", err)
	}
	if observed.ReadyReplicas != 2 {
		t.Fatalf("expected ready replicas 2, got %d", observed.ReadyReplicas)
	}
	if observed.Phase != infraapi.PhaseReconciling {
		t.Fatalf("expected Reconciling, got %s", observed.Phase)
	}
}

func TestFakeClusterAdapterApplyDesiredStateScaleOutOneStep(t *testing.T) {
	adapter := NewFakeClusterAdapter()
	cluster := validCluster(6)
	cluster.Status.WorkerReadyReplicas = 3

	if err := adapter.ApplyDesiredState(context.Background(), cluster); err != nil {
		t.Fatalf("ApplyDesiredState returned error: %v", err)
	}
	observed, err := adapter.Observe(context.Background(), cluster)
	if err != nil {
		t.Fatalf("Observe returned error: %v", err)
	}
	if observed.ReadyReplicas != 4 {
		t.Fatalf("expected ready replicas 4, got %d", observed.ReadyReplicas)
	}
	if observed.Phase != infraapi.PhaseReconciling {
		t.Fatalf("expected Reconciling, got %s", observed.Phase)
	}
}

func TestFakeClusterAdapterApplyDesiredStateScaleDownOneStep(t *testing.T) {
	adapter := NewFakeClusterAdapter()
	cluster := validCluster(3)
	cluster.Status.WorkerReadyReplicas = 6

	if err := adapter.ApplyDesiredState(context.Background(), cluster); err != nil {
		t.Fatalf("ApplyDesiredState returned error: %v", err)
	}
	observed, err := adapter.Observe(context.Background(), cluster)
	if err != nil {
		t.Fatalf("Observe returned error: %v", err)
	}
	if observed.ReadyReplicas != 5 {
		t.Fatalf("expected ready replicas 5, got %d", observed.ReadyReplicas)
	}
}

func TestFakeClusterAdapterApplyDesiredStateReady(t *testing.T) {
	adapter := NewFakeClusterAdapter()
	cluster := validCluster(3)
	cluster.Status.WorkerReadyReplicas = 3

	if err := adapter.ApplyDesiredState(context.Background(), cluster); err != nil {
		t.Fatalf("ApplyDesiredState returned error: %v", err)
	}
	observed, err := adapter.Observe(context.Background(), cluster)
	if err != nil {
		t.Fatalf("Observe returned error: %v", err)
	}
	if observed.Phase != infraapi.PhaseRunning {
		t.Fatalf("expected Running, got %s", observed.Phase)
	}
}

func TestFakeClusterAdapterRejectsInvalidCluster(t *testing.T) {
	adapter := NewFakeClusterAdapter()
	cluster := validCluster(3)
	cluster.Name = ""

	if err := adapter.ApplyDesiredState(context.Background(), cluster); err == nil {
		t.Fatalf("expected validation error")
	}
}

func validCluster(replicas int32) infraapi.ManagedCluster {
	cluster := infraapi.NewManagedCluster("dev-gpu-cluster", "default", "dev")
	cluster.Spec.Workers = []infraapi.WorkerGroupSpec{
		{Name: "gpu-workers", Replicas: replicas, MachineClassRef: infraapi.LocalObjectReference{Name: "gpu-large"}},
	}
	return cluster
}
