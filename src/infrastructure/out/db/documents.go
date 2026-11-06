package db

import (
	"context"
	"database/sql"
	"time"

	"perdimidocumento/src/domain/models"
	"perdimidocumento/src/infrastructure/out/db/internal/dbo"
	"perdimidocumento/src/infrastructure/out/db/internal/mapper"
)

func (r *SQLiteRepository) RegisterDocument(ctx context.Context, doc models.Document) (models.Document, error) {
	record := mapper.DocumentToDBO(doc)
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return models.Document{}, err
	}
	defer tx.Rollback()
	now := time.Now().UTC().Unix()
	record.CreatedAt = now
	result, err := tx.ExecContext(ctx, `INSERT INTO documents(number,type,owner_name,address,reported_by,created_at) VALUES(?,?,?,?,?,?)`, record.Number, record.Type, record.OwnerName, record.Address, record.ReportedBy, record.CreatedAt)
	if err != nil {
		return models.Document{}, err
	}
	record.ID, err = result.LastInsertId()
	if err != nil {
		return models.Document{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO logs(user_id,document_number,document_id,operation,created_at) VALUES(?,?,?,?,?)`, record.ReportedBy, record.Number, record.ID, "registro", now); err != nil {
		return models.Document{}, err
	}
	if err = (subject{}).notify(ctx, tx, record, now); err != nil {
		return models.Document{}, err
	}
	if err = tx.Commit(); err != nil {
		return models.Document{}, err
	}
	return mapper.DocumentToDomain(record), nil
}

func scanDocument(rows *sql.Rows) (dbo.DocumentDBO, error) {
	var d dbo.DocumentDBO
	if err := rows.Scan(&d.ID, &d.Number, &d.Type, &d.OwnerName, &d.Address, &d.CreatedAt); err != nil {
		return dbo.DocumentDBO{}, err
	}
	return d, nil
}

func (r *SQLiteRepository) SearchDocuments(ctx context.Context, number string) ([]models.Document, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id,number,type,owner_name,address,created_at FROM documents WHERE number=? ORDER BY id DESC`, number)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	docs := []models.Document{}
	for rows.Next() {
		d, err := scanDocument(rows)
		if err != nil {
			return nil, err
		}
		docs = append(docs, mapper.DocumentToDomain(d))
	}
	return docs, rows.Err()
}
