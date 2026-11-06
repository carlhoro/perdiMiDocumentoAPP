package db

import (
	"context"
	"database/sql"
	"errors"

	"perdimidocumento/src/domain/models"
	"perdimidocumento/src/domain/ports"
	"perdimidocumento/src/infrastructure/out/db/internal/dbo"
	"perdimidocumento/src/infrastructure/out/db/internal/mapper"
)

func (r *SQLiteRepository) CreateUser(ctx context.Context, username, passwordHash string) error {
	result, err := r.DB.ExecContext(ctx, `INSERT INTO users(username,password_hash) VALUES(?,?) ON CONFLICT(username) DO NOTHING`, username, passwordHash)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ports.ErrConflict
	}
	return nil
}

func (r *SQLiteRepository) UserByName(ctx context.Context, username string) (models.User, error) {
	var u dbo.UserDBO
	err := r.DB.QueryRowContext(ctx, `SELECT id,username,password_hash FROM users WHERE username=?`, username).Scan(&u.ID, &u.Username, &u.PasswordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, ports.ErrNotFound
	}
	if err != nil {
		return models.User{}, err
	}
	return mapper.UserToDomain(u), nil
}
