package router

import (
	"sync"

	"github.com/tommyxie2026-tech/aicloud/internal/domain"
)

var modelVersionBindings sync.Map

func (r *Router) WithModelVersions(repo domain.ModelVersionRepository) *Router {
	if r != nil {
		if repo == nil {
			modelVersionBindings.Delete(r)
		} else {
			modelVersionBindings.Store(r, repo)
		}
	}
	return r
}

func (r *Router) modelVersionRepository() domain.ModelVersionRepository {
	if r == nil {
		return nil
	}
	value, ok := modelVersionBindings.Load(r)
	if !ok {
		return nil
	}
	repo, _ := value.(domain.ModelVersionRepository)
	return repo
}
