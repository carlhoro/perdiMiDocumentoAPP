package db

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"perdimidocumento/src/domain/models"
	"perdimidocumento/src/domain/ports"
	"perdimidocumento/src/infrastructure/out/db/internal/dbo"
	"perdimidocumento/src/infrastructure/out/db/internal/mapper"
)

func (r *SQLiteRepository) CreateSession(ctx context.Context, userID int64, hash string, expiry time.Time) error {
	_, err := r.DB.ExecContext(ctx, `INSERT INTO sessions(token_hash,user_id,expires_at) VALUES(?,?,?)`, hash, userID, expiry.Unix())
	return err
}

func (r *SQLiteRepository) UserBySession(ctx context.Context, hash string) (models.User, error) {
	var u dbo.UserDBO
	err := r.DB.QueryRowContext(ctx, `SELECT u.id,u.username FROM users u JOIN sessions s ON s.user_id=u.id WHERE s.token_hash=? AND s.expires_at>?`, hash, time.Now().Unix()).Scan(&u.ID, &u.Username)
	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, ports.ErrNotFound
	}
	if err != nil {
		return models.User{}, err
	}
	return mapper.UserToDomain(u), nil
}

func (r *SQLiteRepository) DeleteSession(ctx context.Context, hash string) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash=?`, hash)
	return err
}
