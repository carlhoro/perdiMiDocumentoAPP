package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
	"perdimidocumento/src/domain/models"
	"perdimidocumento/src/domain/ports"
)

type PasswordHasher interface {
	Hash(string) (string, error)
	Compare(string, string) error
}

type TokenProvider interface {
	New() (string, error)
	Hash(string) string
}

type BcryptPasswordHasher struct{}

func (BcryptPasswordHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func (BcryptPasswordHasher) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

type SHA256TokenProvider struct{}

func (SHA256TokenProvider) New() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (SHA256TokenProvider) Hash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

type AuthService struct {
	Repository ports.Repository
	Hasher     PasswordHasher
	Tokens     TokenProvider
	SessionTTL time.Duration
}

func (s AuthService) Register(ctx context.Context, username, password string) error {
	hash, err := s.Hasher.Hash(password)
	if err != nil {
		return err
	}
	return s.Repository.CreateUser(ctx, username, hash)
}

func (s AuthService) Login(ctx context.Context, username, password string) (ports.Session, error) {
	user, err := s.Repository.UserByName(ctx, username)
	if errors.Is(err, ports.ErrNotFound) {
		return ports.Session{}, ports.ErrInvalidCredentials
	}
	if err != nil {
		return ports.Session{}, err
	}
	if err := s.Hasher.Compare(user.PasswordHash, password); err != nil {
		return ports.Session{}, ports.ErrInvalidCredentials
	}
	token, err := s.Tokens.New()
	if err != nil {
		return ports.Session{}, err
	}
	ttl := s.SessionTTL
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	expiresAt := time.Now().Add(ttl)
	if err := s.Repository.CreateSession(ctx, user.ID, s.Tokens.Hash(token), expiresAt); err != nil {
		return ports.Session{}, err
	}
	return ports.Session{Token: token, ExpiresAt: expiresAt, User: user}, nil
}

func (s AuthService) UserByToken(ctx context.Context, token string) (models.User, error) {
	return s.Repository.UserBySession(ctx, s.Tokens.Hash(token))
}

func (s AuthService) Logout(ctx context.Context, token string) error {
	return s.Repository.DeleteSession(ctx, s.Tokens.Hash(token))
}
