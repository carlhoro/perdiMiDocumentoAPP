package services

import (
	"context"
	"errors"
	"time"

	"perdimidocumento/src/domain/models/user"
	"perdimidocumento/src/domain/ports"
)

// AuthService coordina el registro de cuentas y el ciclo de vida de las sesiones.
type AuthService struct {
	Repository ports.AuthRepository
	Hasher     PasswordHasher
	Tokens     TokenProvider
	SessionTTL time.Duration
}

// PasswordHasher abstrae la generación y la comparación de hashes de contraseñas.
type PasswordHasher interface {
	Hash(string) (string, error)
	Compare(string, string) error
}

// TokenProvider genera tokens de sesión y calcula su representación para el almacenamiento.
type TokenProvider interface {
	New() (string, error)
	Hash(string) string
}

const defaultSessionTTL = 24 * time.Hour

func (s AuthService) Register(ctx context.Context, username, password string) error {
	hash, err := s.Hasher.Hash(password)
	if err != nil {
		return err
	}
	return s.Repository.CreateUser(ctx, username, hash)
}

// Login verifica las credenciales y entrega el token cuya representación protegida se guarda en el repositorio.
func (s AuthService) Login(ctx context.Context, username, password string) (user.Session, error) {
	account, err := s.verifyCredentials(ctx, username, password)
	if err != nil {
		return user.Session{}, err
	}

	token, err := s.Tokens.New()
	if err != nil {
		return user.Session{}, err
	}

	ttl := s.SessionTTL
	if ttl <= 0 {
		ttl = defaultSessionTTL
	}
	expiresAt := time.Now().Add(ttl)
	if err := s.Repository.CreateSession(ctx, account.ID, s.Tokens.Hash(token), expiresAt); err != nil {
		return user.Session{}, err
	}
	return user.Session{Token: token, ExpiresAt: expiresAt, User: account}, nil
}

func (s AuthService) UserByToken(ctx context.Context, token string) (user.User, error) {
	return s.Repository.UserBySession(ctx, s.Tokens.Hash(token))
}

func (s AuthService) Logout(ctx context.Context, token string) error {
	return s.Repository.DeleteSession(ctx, s.Tokens.Hash(token))
}

func (s AuthService) verifyCredentials(ctx context.Context, username, password string) (user.User, error) {
	account, err := s.Repository.UserByName(ctx, username)
	if errors.Is(err, ports.ErrNotFound) {
		return user.User{}, ports.ErrInvalidCredentials
	}
	if err != nil {
		return user.User{}, err
	}
	if err := s.Hasher.Compare(account.PasswordHash, password); err != nil {
		return user.User{}, ports.ErrInvalidCredentials
	}
	return account, nil
}
