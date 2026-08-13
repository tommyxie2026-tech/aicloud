package router

import (
	"sync"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

var deploymentBindings sync.Map

func (r *Router) WithDeployments(repo domain.DeploymentRepository) *Router {
	if r != nil {
		if repo == nil {
			deploymentBindings.Delete(r)
		} else {
			deploymentBindings.Store(r, repo)
		}
	}
	return r
}

func (r *Router) deploymentRepository() domain.DeploymentRepository {
	if r == nil {
		return nil
	}
	value, ok := deploymentBindings.Load(r)
	if !ok {
		return nil
	}
	repo, _ := value.(domain.DeploymentRepository)
	return repo
}
