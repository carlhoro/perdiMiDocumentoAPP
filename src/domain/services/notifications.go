package services

import (
	"context"

	"perdimidocumento/src/domain/models/notification"
	"perdimidocumento/src/domain/ports"
)

// NotificationService obtiene del repositorio los avisos del usuario.
type NotificationService struct {
	Repository ports.NotificationRepository
}

func (s NotificationService) Notifications(ctx context.Context, userID int64) ([]notification.DocumentNotification, error) {
	return s.Repository.Notifications(ctx, userID)
}
