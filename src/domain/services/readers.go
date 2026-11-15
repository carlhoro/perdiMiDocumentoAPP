package services

import (
	"context"

	"perdimidocumento/src/domain/models"
	"perdimidocumento/src/domain/ports"
)

type DocumentSearchService struct {
	Repository ports.Repository
}

func (s DocumentSearchService) Search(ctx context.Context, number string) ([]models.Document, error) {
	return s.Repository.SearchDocuments(ctx, number)
}

type SubscriptionService struct {
	Repository ports.Repository
}

func (s SubscriptionService) Subscribe(ctx context.Context, userID int64, number string) error {
	return s.Repository.Subscribe(ctx, userID, number)
}

type NotificationService struct {
	Repository ports.Repository
}

func (s NotificationService) Notifications(ctx context.Context, userID int64) ([]models.Notification, error) {
	return s.Repository.Notifications(ctx, userID)
}
