package executionbinding

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	execution "github.com/tommyxie2026-tech/aicloud/execution"
	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

// TargetFromRouteDecision converts an already policy/admission-filtered routing
// decision into the immutable target snapshot consumed by the Execution domain.
// It does not grant permission and must never be used as a policy decision.
func TargetFromRouteDecision(decision domain.RouteDecision, deployment *domain.Deployment) (execution.ExecutionTarget, error) {
	if strings.TrimSpace(decision.ID) == "" || strings.TrimSpace(decision.TaskID) == "" {
		return execution.ExecutionTarget{}, fmt.Errorf("route decision id and task id are required")
	}
	selected := decision.Selected
	if strings.TrimSpace(selected.ModelID) == "" || strings.TrimSpace(selected.ModelVersion) == "" {
		return execution.ExecutionTarget{}, fmt.Errorf("route decision does not select a model target")
	}

	if selected.DeploymentID != "" {
		if deployment == nil {
			return execution.ExecutionTarget{}, fmt.Errorf("selected deployment %q requires a deployment snapshot", selected.DeploymentID)
		}
		if deployment.ID != selected.DeploymentID {
			return execution.ExecutionTarget{}, fmt.Errorf("deployment snapshot does not match selected deployment")
		}
		if deployment.ModelID != "" && deployment.ModelID != selected.ModelID {
			return execution.ExecutionTarget{}, fmt.Errorf("deployment model does not match selected model")
		}
		if deployment.ModelVersion != "" && deployment.ModelVersion != selected.ModelVersion {
			return execution.ExecutionTarget{}, fmt.Errorf("deployment model version does not match selected model version")
		}
		return targetFromDeployment(decision, *deployment)
	}
	return targetFromModelRoute(decision)
}

func targetFromDeployment(decision domain.RouteDecision, deployment domain.Deployment) (execution.ExecutionTarget, error) {
	selected := decision.Selected
	digest, err := routeTargetDigest(struct {
		RouteID         string                `json:"routeId"`
		EvidenceVersion string                `json:"evidenceVersion,omitempty"`
		PolicyVersion   string                `json:"policyVersion,omitempty"`
		Candidate       domain.RouteCandidate `json:"candidate"`
		Deployment      domain.Deployment     `json:"deployment"`
	}{
		RouteID: decision.ID, EvidenceVersion: decision.EvidenceVersion,
		PolicyVersion: decision.PolicyVersion, Candidate: selected, Deployment: deployment,
	})
	if err != nil {
		return execution.ExecutionTarget{}, err
	}

	observedAt := deployment.UpdatedAt
	if deployment.HealthCheckedAt != nil {
		observedAt = deployment.HealthCheckedAt.UTC()
	}
	return execution.ExecutionTarget{
		ID:       execution.TargetID("deployment/" + deployment.ID),
		Revision: 1, // Digest is authoritative until Deployment gets an explicit monotonic revision.
		Type:     execution.TargetModel,
		Capabilities: map[string]string{
			"provider":       deployment.Provider,
			"model.id":       selected.ModelID,
			"model.version":  selected.ModelVersion,
			"deployment.id":  deployment.ID,
			"deployment.mode": string(deployment.Mode),
			"region":         deployment.Region,
			"runtime":        deployment.Runtime,
			"quantization":   deployment.Quantization,
		},
		Policy: execution.TargetPolicy{
			DataLocality: deployment.DataResidency,
		},
		Health: execution.TargetHealth{
			State:      string(deployment.Health),
			ObservedAt: observedAt,
		},
		Snapshot: execution.TargetSnapshot{
			Digest:         digest,
			EndpointClass:  string(deployment.Mode),
			ModelVersion:   selected.ModelVersion,
			RuntimeVersion: deployment.Runtime,
			DeploymentRef:  deployment.ID,
		},
	}, nil
}

func targetFromModelRoute(decision domain.RouteDecision) (execution.ExecutionTarget, error) {
	selected := decision.Selected
	digest, err := routeTargetDigest(struct {
		RouteID         string                `json:"routeId"`
		EvidenceVersion string                `json:"evidenceVersion,omitempty"`
		PolicyVersion   string                `json:"policyVersion,omitempty"`
		Candidate       domain.RouteCandidate `json:"candidate"`
	}{
		RouteID: decision.ID, EvidenceVersion: decision.EvidenceVersion,
		PolicyVersion: decision.PolicyVersion, Candidate: selected,
	})
	if err != nil {
		return execution.ExecutionTarget{}, err
	}
	return execution.ExecutionTarget{
		ID:       execution.TargetID("model/" + selected.ModelID + "@" + selected.ModelVersion),
		Revision: 1,
		Type:     execution.TargetModel,
		Capabilities: map[string]string{
			"model.id":      selected.ModelID,
			"model.version": selected.ModelVersion,
			"route.class":   string(selected.RouteClass),
		},
		Snapshot: execution.TargetSnapshot{
			Digest:       digest,
			ModelVersion: selected.ModelVersion,
		},
	}, nil
}

func routeTargetDigest(value any) (string, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode execution target snapshot: %w", err)
	}
	sum := sha256.Sum256(body)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
