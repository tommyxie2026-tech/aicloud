package router

import (
	"testing"
	"time"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

func TestDeploymentCandidate(t *testing.T) {
	now := time.Now().UTC()
	m := domain.Model{ID: "m1", Version: "v1"}
	d := domain.Deployment{ID: "d1", ModelID: "m1", ModelVersion: "v1", Lifecycle: domain.DeploymentReady, RoutingEligible: true, Health: domain.HealthHealthy, HealthCheckedAt: &now}
	got := DeploymentCandidate(m, d, Request{RequireFreshSignals: true, SignalMaxAge: time.Minute}, now)
	if got.DeploymentID != "d1" || len(got.RejectionReasons) != 0 {
		t.Fatalf("unexpected candidate: %+v", got)
	}

	old := now.Add(-2 * time.Minute)
	d.HealthCheckedAt = &old
	got = DeploymentCandidate(m, d, Request{RequireFreshSignals: true, SignalMaxAge: time.Minute}, now)
	if len(got.RejectionReasons) == 0 {
		t.Fatal("expected candidate to be ineligible")
	}
}
