package db

import (
	"context"
	"database/sql"
	"time"

	"perdimidocumento/src/domain/models"
	"perdimidocumento/src/infrastructure/out/db/internal/dbo"
	"perdimidocumento/src/infrastructure/out/db/internal/mapper"
)

// observer is a persistent subscriber receiving found-document events.
type observer struct{ userID int64 }

func (o observer) update(ctx context.Context, tx *sql.Tx, documentID, now int64) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO notifications(user_id,document_id,created_at) VALUES(?,?,?) ON CONFLICT(user_id,document_id) DO NOTHING`, o.userID, documentID, now)
	return err
}

type subject struct{}

func (subject) notify(ctx context.Context, tx *sql.Tx, doc dbo.DocumentDBO, now int64) error {
	rows, err := tx.QueryContext(ctx, `SELECT user_id FROM subscriptions WHERE number=?`, doc.Number)
	if err != nil {
		return err
	}
	var observers []observer
	for rows.Next() {
		var o observer
		if err := rows.Scan(&o.userID); err != nil {
			rows.Close()
			return err
		}
		observers = append(observers, o)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, o := range observers {
		if err := o.update(ctx, tx, doc.ID, now); err != nil {
			return err
		}
	}
	return nil
}

func (r *SQLiteRepository) Subscribe(ctx context.Context, userID int64, number string) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO subscriptions(user_id,number) VALUES(?,?) ON CONFLICT(user_id,number) DO NOTHING`, userID, number); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO notifications(user_id,document_id,created_at)
		SELECT ?,id,? FROM documents WHERE number=? ON CONFLICT(user_id,document_id) DO NOTHING`, userID, time.Now().UTC().Unix(), number)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *SQLiteRepository) Notifications(ctx context.Context, userID int64) ([]models.Notification, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT n.id,n.created_at,d.id,d.number,d.type,d.owner_name,d.address,d.created_at
		FROM notifications n JOIN documents d ON d.id=n.document_id WHERE n.user_id=? ORDER BY n.id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	notifications := []models.Notification{}
	for rows.Next() {
		var n dbo.NotificationDBO
		if err := rows.Scan(&n.ID, &n.CreatedAt, &n.Document.ID, &n.Document.Number, &n.Document.Type, &n.Document.OwnerName, &n.Document.Address, &n.Document.CreatedAt); err != nil {
			return nil, err
		}
		notifications = append(notifications, mapper.NotificationToDomain(n))
	}
	return notifications, rows.Err()
}
