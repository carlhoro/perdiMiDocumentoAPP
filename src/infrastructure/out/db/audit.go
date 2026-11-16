package db

import (
	"context"
	"time"
)

func (r *SQLiteRepository) LogRejectedDocument(ctx context.Context, userID int64, number string) error {
	_, err := r.DB.ExecContext(ctx, `INSERT INTO logs(user_id,document_number,operation,created_at) VALUES(?,?,?,?)`, userID, number, "registro_rechazado", time.Now().UTC().Unix())
	return err
}
