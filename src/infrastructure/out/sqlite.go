package out

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"perdimidocumento/src/domain/models"
	"perdimidocumento/src/domain/ports"

	_ "modernc.org/sqlite"
)

var singleton struct {
	once sync.Once
	db   *sql.DB
	err  error
	path string
}

// OpenSingleton establishes one database connection pool for the application.
func OpenSingleton(path string) (*sql.DB, error) {
	singleton.once.Do(func() {
		singleton.path = path
		singleton.db, singleton.err = Open(path)
	})
	if singleton.path != path {
		return nil, errors.New("la conexión singleton ya fue configurada con otra ruta")
	}
	return singleton.db, singleton.err
}

// Open is also available for isolated tests.
func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	// Serialize SQLite writes and subscription/registration transactions.
	db.SetMaxOpenConns(1)
	if _, err = db.Exec(`PRAGMA foreign_keys = ON; PRAGMA busy_timeout = 5000;`); err != nil {
		db.Close()
		return nil, err
	}
	if _, err = db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("crear esquema: %w", err)
	}
	return db, nil
}

const schema = `
CREATE TABLE IF NOT EXISTS users (
 id INTEGER PRIMARY KEY, username TEXT NOT NULL UNIQUE, password_hash TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS sessions (
 token_hash TEXT PRIMARY KEY, user_id INTEGER NOT NULL REFERENCES users(id), expires_at INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS documents (
 id INTEGER PRIMARY KEY, number TEXT NOT NULL, type TEXT NOT NULL, owner_name TEXT NOT NULL,
 address TEXT NOT NULL, reported_by INTEGER NOT NULL REFERENCES users(id), created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS documents_number ON documents(number);
CREATE TABLE IF NOT EXISTS subscriptions (
 user_id INTEGER NOT NULL REFERENCES users(id), number TEXT NOT NULL, PRIMARY KEY(user_id, number)
);
CREATE INDEX IF NOT EXISTS subscriptions_number ON subscriptions(number);
CREATE TABLE IF NOT EXISTS notifications (
 id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL REFERENCES users(id),
 document_id INTEGER NOT NULL REFERENCES documents(id), created_at INTEGER NOT NULL,
 UNIQUE(user_id, document_id)
);
CREATE TABLE IF NOT EXISTS logs (
 id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL REFERENCES users(id), document_number TEXT NOT NULL,
 document_id INTEGER REFERENCES documents(id), operation TEXT NOT NULL, created_at INTEGER NOT NULL
);
`

type SQLiteRepository struct{ DB *sql.DB }

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
	var u models.User
	err := r.DB.QueryRowContext(ctx, `SELECT id,username,password_hash FROM users WHERE username=?`, username).Scan(&u.ID, &u.Username, &u.PasswordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return u, ports.ErrNotFound
	}
	return u, err
}

func (r *SQLiteRepository) CreateSession(ctx context.Context, userID int64, hash string, expiry time.Time) error {
	_, err := r.DB.ExecContext(ctx, `INSERT INTO sessions(token_hash,user_id,expires_at) VALUES(?,?,?)`, hash, userID, expiry.Unix())
	return err
}

func (r *SQLiteRepository) UserBySession(ctx context.Context, hash string) (models.User, error) {
	var u models.User
	err := r.DB.QueryRowContext(ctx, `SELECT u.id,u.username FROM users u JOIN sessions s ON s.user_id=u.id WHERE s.token_hash=? AND s.expires_at>?`, hash, time.Now().Unix()).Scan(&u.ID, &u.Username)
	if errors.Is(err, sql.ErrNoRows) {
		return u, ports.ErrNotFound
	}
	return u, err
}

func (r *SQLiteRepository) DeleteSession(ctx context.Context, hash string) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash=?`, hash)
	return err
}

// observer is a persistent subscriber receiving found-document events.
type observer struct{ userID int64 }

func (o observer) update(ctx context.Context, tx *sql.Tx, documentID, now int64) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO notifications(user_id,document_id,created_at) VALUES(?,?,?) ON CONFLICT(user_id,document_id) DO NOTHING`, o.userID, documentID, now)
	return err
}

type subject struct{}

func (subject) notify(ctx context.Context, tx *sql.Tx, doc models.Document, now int64) error {
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

func (r *SQLiteRepository) RegisterDocument(ctx context.Context, doc models.Document) (models.Document, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return models.Document{}, err
	}
	defer tx.Rollback()
	now := time.Now().UTC().Unix()
	result, err := tx.ExecContext(ctx, `INSERT INTO documents(number,type,owner_name,address,reported_by,created_at) VALUES(?,?,?,?,?,?)`, doc.Number, doc.Type, doc.OwnerName, doc.Address, doc.ReportedBy, now)
	if err != nil {
		return models.Document{}, err
	}
	doc.ID, err = result.LastInsertId()
	if err != nil {
		return models.Document{}, err
	}
	doc.CreatedAt = time.Unix(now, 0).UTC()
	if _, err = tx.ExecContext(ctx, `INSERT INTO logs(user_id,document_number,document_id,operation,created_at) VALUES(?,?,?,?,?)`, doc.ReportedBy, doc.Number, doc.ID, "registro", now); err != nil {
		return models.Document{}, err
	}
	if err = (subject{}).notify(ctx, tx, doc, now); err != nil {
		return models.Document{}, err
	}
	if err = tx.Commit(); err != nil {
		return models.Document{}, err
	}
	return doc, nil
}

func (r *SQLiteRepository) LogRejectedDocument(ctx context.Context, userID int64, number string) error {
	_, err := r.DB.ExecContext(ctx, `INSERT INTO logs(user_id,document_number,operation,created_at) VALUES(?,?,?,?)`, userID, number, "registro_rechazado", time.Now().UTC().Unix())
	return err
}

func scanDocument(rows *sql.Rows) (models.Document, error) {
	var d models.Document
	var created int64
	err := rows.Scan(&d.ID, &d.Number, &d.Type, &d.OwnerName, &d.Address, &created)
	d.CreatedAt = time.Unix(created, 0).UTC()
	return d, err
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
		docs = append(docs, d)
	}
	return docs, rows.Err()
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
		var n models.Notification
		var created, docCreated int64
		if err := rows.Scan(&n.ID, &created, &n.Document.ID, &n.Document.Number, &n.Document.Type, &n.Document.OwnerName, &n.Document.Address, &docCreated); err != nil {
			return nil, err
		}
		n.CreatedAt = time.Unix(created, 0).UTC()
		n.Document.CreatedAt = time.Unix(docCreated, 0).UTC()
		notifications = append(notifications, n)
	}
	return notifications, rows.Err()
}
