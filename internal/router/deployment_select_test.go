package router

import (
	"testing"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

func TestSelectDeploymentCandidateKeepsFallbackOrder(t *testing.T) {
	selected, fallback, err := SelectDeploymentCandidate([]domain.RouteCandidate{
		{DeploymentID: "d-low", Score: 70, EstimatedCost: 1},
		{DeploymentID: "d-rejected", Score: 100, RejectionReasons: []string{"not eligible"}},
		{DeploymentID: "d-high", Score: 90, EstimatedCost: 2},
		{DeploymentID: "d-mid", Score: 80, EstimatedCost: 1.5},
	})
	if err != nil {
		t.Fatalf("SelectDeploymentCandidate returned error: %v", err)
	}
	if selected.DeploymentID != "d-high" {
		t.Fatalf("selected deployment = %q", selected.DeploymentID)
	}
	if len(fallback) != 2 || fallback[0].DeploymentID != "d-mid" || fallback[1].DeploymentID != "d-low" {
		t.Fatalf("unexpected fallback chain: %#v", fallback)
	}
}
