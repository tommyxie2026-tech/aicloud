package domain

import (
	"testing"
	"time"
)

func TestDeploymentRoutingEligibility(t *testing.T) {
	now := time.Now().UTC()
	fresh := now.Add(-time.Minute)
	base := Deployment{
		Lifecycle:       DeploymentReady,
		RoutingEligible: true,
		Health:          HealthHealthy,
		HealthCheckedAt: &fresh,
	}
	if !base.IsRoutingEligible(now, 5*time.Minute) {
		t.Fatal("expected fresh ready deployment to be eligible")
	}

	cases := []struct {
		name string
		mut  func(*Deployment)
	}{
		{"disabled", func(d *Deployment) { d.RoutingEligible = false }},
		{"draining", func(d *Deployment) { d.Lifecycle = DeploymentDraining }},
		{"retired", func(d *Deployment) { d.Lifecycle = DeploymentRetired }},
		{"blocked", func(d *Deployment) { d.Lifecycle = DeploymentBlocked }},
		{"unhealthy", func(d *Deployment) { d.Health = HealthUnhealthy }},
		{"missing-signal", func(d *Deployment) { d.HealthCheckedAt = nil }},
		{"stale-signal", func(d *Deployment) { stale := now.Add(-10 * time.Minute); d.HealthCheckedAt = &stale }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := base
			tc.mut(&d)
			if d.IsRoutingEligible(now, 5*time.Minute) {
				t.Fatalf("expected %s deployment to be ineligible", tc.name)
			}
		})
	}
}
