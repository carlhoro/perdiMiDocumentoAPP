package ports

import (
	"context"
	"errors"
	"time"

	"perdimidocumento/src/domain/models"
)

var ErrConflict = errors.New("ya existe")
var ErrNotFound = errors.New("no encontrado")
var ErrInvalidCredentials = errors.New("credenciales_incorrectas")

type Session struct {
	Token     string
	ExpiresAt time.Time
	User      models.User
}

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

type UserRegistrar interface {
	Register(context.Context, string, string) error
}

type SessionManager interface {
	Login(context.Context, string, string) (Session, error)
	UserByToken(context.Context, string) (models.User, error)
	Logout(context.Context, string) error
}

type DocumentSearcher interface {
	Search(context.Context, string) ([]models.Document, error)
}

type SubscriptionManager interface {
	Subscribe(context.Context, int64, string) error
}

type NotificationReader interface {
	Notifications(context.Context, int64) ([]models.Notification, error)
}
