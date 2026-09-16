package executionbinding

import (
	"testing"
	"time"

	execution "github.com/tommyxie2026-tech/aicloud/execution"
	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

func TestTargetFromRouteDecisionFreezesDeploymentSnapshot(t *testing.T) {
	healthAt := time.Date(2026, 9, 16, 8, 0, 0, 0, time.UTC)
	decision := domain.RouteDecision{
		ID: "route-1", TaskID: "task-1", EvidenceVersion: "ev-7", PolicyVersion: "policy-3",
		Selected: domain.RouteCandidate{
			ModelID: "model-a", ModelVersion: "v2", DeploymentID: "dep-a",
			RouteClass: domain.RouteEfficient,
		},
	}
	deployment := domain.Deployment{
		ID: "dep-a", ModelID: "model-a", ModelVersion: "v2", Provider: "provider-a",
		Mode: domain.DeploymentMode("remote-api"), Region: "ap-northeast", DataResidency: "apac",
		Runtime: "provider-runtime-v1", Quantization: "none", Health: domain.HealthHealthy,
		HealthCheckedAt: &healthAt, UpdatedAt: healthAt,
	}

	target, err := TargetFromRouteDecision(decision, &deployment)
	if err != nil {
		t.Fatal(err)
	}
	if target.ID != "deployment/dep-a" || target.Type != execution.TargetModel {
		t.Fatalf("unexpected target identity: %+v", target)
	}
	if target.Snapshot.DeploymentRef != "dep-a" || target.Snapshot.ModelVersion != "v2" {
		t.Fatalf("route snapshot lost deployment lineage: %+v", target.Snapshot)
	}
	if target.Policy.DataLocality != "apac" || target.Health.ObservedAt != healthAt {
		t.Fatalf("target policy/health evidence not frozen: %+v", target)
	}
	if target.Snapshot.Digest == "" {
		t.Fatal("target snapshot digest is required")
	}

	targetAgain, err := TargetFromRouteDecision(decision, &deployment)
	if err != nil {
		t.Fatal(err)
	}
	if targetAgain.Snapshot.Digest != target.Snapshot.Digest {
		t.Fatal("same routing evidence must produce a deterministic target digest")
	}
}

func TestTargetFromRouteDecisionRequiresSelectedDeploymentSnapshot(t *testing.T) {
	decision := domain.RouteDecision{
		ID: "route-2", TaskID: "task-2",
		Selected: domain.RouteCandidate{ModelID: "model-a", ModelVersion: "v1", DeploymentID: "dep-a"},
	}
	if _, err := TargetFromRouteDecision(decision, nil); err == nil {
		t.Fatal("selected deployment must be resolved before execution")
	}
}

func TestTargetFromRouteDecisionSupportsModelOnlyRoute(t *testing.T) {
	decision := domain.RouteDecision{
		ID: "route-3", TaskID: "task-3",
		Selected: domain.RouteCandidate{
			ModelID: "model-a", ModelVersion: "v1", RouteClass: domain.RouteFlagship,
		},
	}
	target, err := TargetFromRouteDecision(decision, nil)
	if err != nil {
		t.Fatal(err)
	}
	if target.ID != "model/model-a@v1" || target.Snapshot.Digest == "" {
		t.Fatalf("unexpected model target: %+v", target)
	}
}

func TestTargetFromRouteDecisionDoesNotTreatDeterministicRouteAsModelPermission(t *testing.T) {
	decision := domain.RouteDecision{
		ID: "route-4", TaskID: "task-4",
		Selected: domain.RouteCandidate{RouteClass: domain.RouteDeterministic},
	}
	if _, err := TargetFromRouteDecision(decision, nil); err == nil {
		t.Fatal("deterministic/non-model route must not become a model execution target")
	}
}
