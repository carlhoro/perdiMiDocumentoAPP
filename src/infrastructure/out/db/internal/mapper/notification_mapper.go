package mapper

import (
	"time"

	"perdimidocumento/src/domain/models"
	"perdimidocumento/src/infrastructure/out/db/internal/dbo"
)

func NotificationToDomain(notification dbo.NotificationDBO) models.Notification {
	return models.Notification{
		ID: notification.ID, Document: DocumentToDomain(notification.Document),
		CreatedAt: time.Unix(notification.CreatedAt, 0).UTC(),
	}
}
