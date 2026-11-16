package in

import (
	"time"

	"perdimidocumento/src/domain/models"
)

type documentResponse struct {
	ID        int64     `json:"id"`
	Number    string    `json:"numero"`
	Type      string    `json:"tipo"`
	OwnerName string    `json:"nombre_titular"`
	Address   string    `json:"direccion"`
	CreatedAt time.Time `json:"fecha"`
}

func documentToResponse(doc models.Document) documentResponse {
	return documentResponse{
		ID: doc.ID, Number: doc.Number, Type: doc.Type, OwnerName: doc.OwnerName,
		Address: doc.Address, CreatedAt: doc.CreatedAt,
	}
}

func documentsToResponse(docs []models.Document) []documentResponse {
	if docs == nil {
		return nil
	}
	responses := make([]documentResponse, len(docs))
	for i, doc := range docs {
		responses[i] = documentToResponse(doc)
	}
	return responses
}

type notificationResponse struct {
	ID        int64            `json:"id"`
	Document  documentResponse `json:"documento"`
	CreatedAt time.Time        `json:"fecha"`
}

func notificationsToResponse(notifications []models.Notification) []notificationResponse {
	if notifications == nil {
		return nil
	}
	responses := make([]notificationResponse, len(notifications))
	for i, notification := range notifications {
		responses[i] = notificationResponse{
			ID: notification.ID, Document: documentToResponse(notification.Document),
			CreatedAt: notification.CreatedAt,
		}
	}
	return responses
}
