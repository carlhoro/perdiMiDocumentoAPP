package ports

import (
	"context"
	"errors"
	"time"

	"perdimidocumento/src/domain/models"
)

var ErrConflict = errors.New("ya existe")
var ErrNotFound = errors.New("no encontrado")

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

type DocumentRegistrar interface {
	Register(context.Context, int64, string, string, string, string) (models.Document, error)
}
