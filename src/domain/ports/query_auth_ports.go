package ports

import (
	"context"
	"errors"

	"perdimidocumento/src/domain/models/document"
	"perdimidocumento/src/domain/models/notification"
	"perdimidocumento/src/domain/models/user"
)

// Session conserva el nombre usado por los consumidores del puerto de autenticación.
type Session = user.Session

// UserRegistrar crea una cuenta local.
type UserRegistrar interface {
	Register(context.Context, string, string) error
}

// SessionManager controla el inicio, la consulta y el cierre de sesiones.
type SessionManager interface {
	Login(context.Context, string, string) (user.Session, error)
	UserByToken(context.Context, string) (user.User, error)
	Logout(context.Context, string) error
}

// DocumentSearcher busca documentos por su número de identificación.
type DocumentSearcher interface {
	Search(context.Context, string) ([]document.FoundDocument, error)
}

// SubscriptionManager registra el interés de un usuario en un número documental.
type SubscriptionManager interface {
	Subscribe(context.Context, int64, string) error
}

// NotificationReader consulta los avisos que corresponden al usuario.
type NotificationReader interface {
	Notifications(context.Context, int64) ([]notification.DocumentNotification, error)
}

// ErrInvalidCredentials indica que las credenciales presentadas no son válidas.
var ErrInvalidCredentials = errors.New("credenciales_incorrectas")
