package modelruntime

import (
	"context"
	"sync"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

var executorDeploymentBindings sync.Map

func (e *Executor) WithDeployments(repo domain.DeploymentRepository) *Executor {
	if e != nil {
		if repo == nil {
			executorDeploymentBindings.Delete(e)
		} else {
			executorDeploymentBindings.Store(e, repo)
		}
	}
	return e
}

func (e *Executor) deploymentRepository() domain.DeploymentRepository {
	if e == nil {
		return nil
	}
	value, ok := executorDeploymentBindings.Load(e)
	if !ok {
		return nil
	}
	repo, _ := value.(domain.DeploymentRepository)
	return repo
}

func (e *Executor) deploymentEvidence(ctx context.Context, candidate domain.RouteCandidate) (domain.Deployment, bool, error) {
	if candidate.DeploymentID == "" {
		return domain.Deployment{}, false, nil
	}
	repo := e.deploymentRepository()
	if repo == nil {
		return domain.Deployment{}, false, nil
	}
	item, err := repo.Get(ctx, candidate.DeploymentID)
	if err != nil {
		return domain.Deployment{}, false, err
	}
	return item, true, nil
}
