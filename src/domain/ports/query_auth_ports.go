package ports

import (
	"context"
	"errors"
	"time"

	"perdimidocumento/src/domain/models"
)

var ErrInvalidCredentials = errors.New("credenciales_incorrectas")

type Session struct {
	Token     string
	ExpiresAt time.Time
	User      models.User
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
