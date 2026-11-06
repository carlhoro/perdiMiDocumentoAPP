package db

import (
	"database/sql"
	"errors"
	"fmt"
	"sync"

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
