package domain

import "fmt"

func ValidateDeploymentTransition(from, to DeploymentLifecycle) error {
	allowed := map[DeploymentLifecycle][]DeploymentLifecycle{
		DeploymentDiscovered: {DeploymentReady, DeploymentBlocked, DeploymentRetired},
		DeploymentReady:      {DeploymentDegraded, DeploymentDraining, DeploymentBlocked, DeploymentRetired},
		DeploymentDegraded:   {DeploymentReady, DeploymentDraining, DeploymentBlocked, DeploymentRetired},
		DeploymentDraining:   {DeploymentReady, DeploymentBlocked, DeploymentRetired},
	}
	for _, target := range allowed[from] {
		if target == to {
			return nil
		}
	}
	return fmt.Errorf("invalid deployment lifecycle transition: %s -> %s", from, to)
}
