package ports

import (
	"context"
	"errors"
	"time"

	"perdimidocumento/src/domain/models/document"
	"perdimidocumento/src/domain/models/notification"
	"perdimidocumento/src/domain/models/user"
)

// Repository reúne los contratos de persistencia que implementa el repositorio de la aplicación.
type Repository interface {
	AuthRepository
	DocumentRegistrationRepository
	DocumentSearchRepository
	SubscriptionRepository
	NotificationRepository
}

// AuthRepository persiste las cuentas de usuario y sus sesiones.
type AuthRepository interface {
	CreateUser(context.Context, string, string) error
	UserByName(context.Context, string) (user.User, error)
	CreateSession(context.Context, int64, string, time.Time) error
	UserBySession(context.Context, string) (user.User, error)
	DeleteSession(context.Context, string) error
}

// DocumentRegistrationRepository guarda hallazgos y audita los intentos rechazados.
type DocumentRegistrationRepository interface {
	RegisterDocument(context.Context, document.FoundDocument) (document.FoundDocument, error)
	LogRejectedDocument(context.Context, int64, string) error
}

// DocumentSearchRepository consulta los hallazgos por número de documento.
type DocumentSearchRepository interface {
	SearchDocuments(context.Context, string) ([]document.FoundDocument, error)
}

// SubscriptionRepository registra el interés del usuario por un número documental.
type SubscriptionRepository interface {
	Subscribe(context.Context, int64, string) error
}

// NotificationRepository consulta los avisos de un usuario.
type NotificationRepository interface {
	Notifications(context.Context, int64) ([]notification.DocumentNotification, error)
}

// ErrConflict indica que ya existe un registro incompatible con la operación.
var ErrConflict = errors.New("ya existe")

// ErrNotFound indica que el registro solicitado no existe.
var ErrNotFound = errors.New("no encontrado")
