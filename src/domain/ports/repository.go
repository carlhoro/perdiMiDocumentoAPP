package ports

import (
	"context"
	"errors"
	"time"

	"perdimidocumento/src/domain/models"
)

// Repository define las operaciones de persistencia de cuentas, sesiones y documentos.
type Repository interface {
	CreateUser(context.Context, string, string) error
	UserByName(context.Context, string) (models.User, error)
	CreateSession(context.Context, int64, string, time.Time) error
	UserBySession(context.Context, string) (models.User, error)
	DeleteSession(context.Context, string) error
	RegisterDocument(context.Context, models.Document) (models.Document, error)
	LogRejectedDocument(context.Context, int64, string) error
	SearchDocuments(context.Context, string) ([]models.Document, error)
	Subscribe(context.Context, int64, string) error
	Notifications(context.Context, int64) ([]models.Notification, error)
}

// ErrConflict indica que ya existe un registro incompatible con la operación.
var ErrConflict = errors.New("ya existe")

// ErrNotFound indica que el registro solicitado no existe.
var ErrNotFound = errors.New("no encontrado")
