package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"

	"golang.org/x/crypto/bcrypt"
)

// BcryptPasswordHasher usa bcrypt para proteger y verificar contraseñas.
type BcryptPasswordHasher struct{}

// SHA256TokenProvider genera tokens aleatorios y deriva su hash SHA-256 para persistirlos.
type SHA256TokenProvider struct{}

func (BcryptPasswordHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func (BcryptPasswordHasher) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

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
