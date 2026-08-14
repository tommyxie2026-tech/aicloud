package domain

import (
	"testing"
	"time"
)

func TestDeploymentRoutingEligibility(t *testing.T) {
	now := time.Now().UTC()
	deployment := Deployment{
		ID: "dep-1", Lifecycle: DeploymentReady, RoutingEligible: true,
		Health: HealthHealthy, HealthCheckedAt: &now,
	}
	if !deployment.IsRoutingEligible(now, time.Minute) {
		t.Fatal("ready healthy deployment should be eligible")
	}
}

func TestDeploymentLifecycleBlocksRouting(t *testing.T) {
	now := time.Now().UTC()
	for _, state := range []DeploymentLifecycle{DeploymentDraining, DeploymentRetired, DeploymentBlocked} {
		deployment := Deployment{
			ID: "dep-1", Lifecycle: state, RoutingEligible: true,
			Health: HealthHealthy, HealthCheckedAt: &now,
		}
		if deployment.IsRoutingEligible(now, time.Minute) {
			t.Fatalf("deployment lifecycle %s must not be routable", state)
		}
	}
}

func TestDeploymentRejectsStaleSignals(t *testing.T) {
	now := time.Now().UTC()
	old := now.Add(-10 * time.Minute)
	deployment := Deployment{
		ID: "dep-1", Lifecycle: DeploymentReady, RoutingEligible: true,
		Health: HealthHealthy, HealthCheckedAt: &old,
	}
	if deployment.IsRoutingEligible(now, time.Minute) {
		t.Fatal("stale runtime signal must not be treated as current")
	}
}
