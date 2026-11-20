package ports

import (
	"context"
	"errors"
	"time"

	"perdimidocumento/src/domain/models"
)

// Session contiene el token de acceso, su vencimiento y el usuario autenticado.
type Session struct {
	Token     string
	ExpiresAt time.Time
	User      models.User
}

// UserRegistrar crea una cuenta local.
type UserRegistrar interface {
	Register(context.Context, string, string) error
}

// SessionManager controla el inicio, la consulta y el cierre de sesiones.
type SessionManager interface {
	Login(context.Context, string, string) (Session, error)
	UserByToken(context.Context, string) (models.User, error)
	Logout(context.Context, string) error
}

// DocumentSearcher busca documentos por su número de identificación.
type DocumentSearcher interface {
	Search(context.Context, string) ([]models.Document, error)
}

// SubscriptionManager registra el interés de un usuario en un número documental.
type SubscriptionManager interface {
	Subscribe(context.Context, int64, string) error
}

// NotificationReader consulta los avisos que corresponden al usuario.
type NotificationReader interface {
	Notifications(context.Context, int64) ([]models.Notification, error)
}

// ErrInvalidCredentials indica que las credenciales presentadas no son válidas.
var ErrInvalidCredentials = errors.New("credenciales_incorrectas")
