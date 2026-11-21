package services

import (
	"context"

	"perdimidocumento/src/domain/models/document"
	"perdimidocumento/src/domain/ports"
)

// DocumentSearchService delega al repositorio la búsqueda por número.
type DocumentSearchService struct {
	Repository ports.DocumentSearchRepository
}

func (s DocumentSearchService) Search(ctx context.Context, number string) ([]document.FoundDocument, error) {
	return s.Repository.SearchDocuments(ctx, number)
}
