package router

import (
	"fmt"
	"sort"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

func SelectDeploymentCandidate(items []domain.RouteCandidate) (domain.RouteCandidate, []domain.RouteCandidate, error) {
	eligible := make([]domain.RouteCandidate, 0, len(items))
	for _, item := range items {
		if item.DeploymentID != "" && len(item.RejectionReasons) == 0 {
			eligible = append(eligible, item)
		}
	}
	if len(eligible) == 0 {
		return domain.RouteCandidate{}, nil, fmt.Errorf("no eligible deployment candidate")
	}
	sort.SliceStable(eligible, func(i, j int) bool {
		if eligible[i].Score == eligible[j].Score {
			return eligible[i].EstimatedCost < eligible[j].EstimatedCost
		}
		return eligible[i].Score > eligible[j].Score
	})
	return eligible[0], append([]domain.RouteCandidate(nil), eligible[1:]...), nil
}
