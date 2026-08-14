package domain

import "testing"

func TestDeploymentLifecycleTransitions(t *testing.T) {
	allowed := [][2]DeploymentLifecycle{
		{DeploymentDiscovered, DeploymentReady},
		{DeploymentReady, DeploymentDegraded},
		{DeploymentReady, DeploymentDraining},
		{DeploymentDegraded, DeploymentReady},
		{DeploymentDraining, DeploymentRetired},
	}
	for _, transition := range allowed {
		if err := ValidateDeploymentTransition(transition[0], transition[1]); err != nil {
			t.Fatalf("expected transition %s -> %s to be allowed: %v", transition[0], transition[1], err)
		}
	}

	for _, state := range []DeploymentLifecycle{DeploymentRetired, DeploymentBlocked} {
		if err := ValidateDeploymentTransition(state, DeploymentReady); err == nil {
			t.Fatalf("terminal deployment state %s must not return to ready", state)
		}
	}
}
