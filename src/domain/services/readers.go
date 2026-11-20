package services

import (
	"context"

	"perdimidocumento/src/domain/models"
	"perdimidocumento/src/domain/ports"
)

// DocumentSearchService delega al repositorio la búsqueda por número.
type DocumentSearchService struct {
	Repository ports.Repository
}

// SubscriptionService registra en el repositorio el interés del usuario por un número documental.
type SubscriptionService struct {
	Repository ports.Repository
}

// NotificationService obtiene del repositorio los avisos del usuario.
type NotificationService struct {
	Repository ports.Repository
}

func (s DocumentSearchService) Search(ctx context.Context, number string) ([]models.Document, error) {
	return s.Repository.SearchDocuments(ctx, number)
}

func (s SubscriptionService) Subscribe(ctx context.Context, userID int64, number string) error {
	return s.Repository.Subscribe(ctx, userID, number)
}

func (s NotificationService) Notifications(ctx context.Context, userID int64) ([]models.Notification, error) {
	return s.Repository.Notifications(ctx, userID)
}
