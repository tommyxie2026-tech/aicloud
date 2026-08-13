package main

import "github.com/tommyxie2026-tech/aicloud/internal/domain"

func modelVersionRepository(models domain.ModelRepository) domain.ModelVersionRepository {
	if value, ok := models.(interface {
		ModelVersionRepository() domain.ModelVersionRepository
	}); ok {
		return value.ModelVersionRepository()
	}
	return nil
}
