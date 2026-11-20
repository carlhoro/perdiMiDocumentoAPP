package ports

import (
	"context"
	"errors"
	"time"

	"perdimidocumento/src/domain/models/document"
	"perdimidocumento/src/domain/models/notification"
	"perdimidocumento/src/domain/models/user"
)

// Repository define las operaciones de persistencia de cuentas, sesiones y documentos.
type Repository interface {
	CreateUser(context.Context, string, string) error
	UserByName(context.Context, string) (user.User, error)
	CreateSession(context.Context, int64, string, time.Time) error
	UserBySession(context.Context, string) (user.User, error)
	DeleteSession(context.Context, string) error
	RegisterDocument(context.Context, document.FoundDocument) (document.FoundDocument, error)
	LogRejectedDocument(context.Context, int64, string) error
	SearchDocuments(context.Context, string) ([]document.FoundDocument, error)
	Subscribe(context.Context, int64, string) error
	Notifications(context.Context, int64) ([]notification.DocumentNotification, error)
}

// ErrConflict indica que ya existe un registro incompatible con la operación.
var ErrConflict = errors.New("ya existe")

// ErrNotFound indica que el registro solicitado no existe.
var ErrNotFound = errors.New("no encontrado")
