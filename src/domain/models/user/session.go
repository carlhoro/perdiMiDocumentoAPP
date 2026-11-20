package user

import "time"

// Session contiene el token de acceso, su vencimiento y el usuario autenticado.
type Session struct {
	Token     string
	ExpiresAt time.Time
	User      User
}
