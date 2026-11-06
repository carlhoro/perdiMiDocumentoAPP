package db

import (
	"database/sql"

	"perdimidocumento/src/domain/ports"
)

type SQLiteRepository struct{ DB *sql.DB }

var _ ports.Repository = (*SQLiteRepository)(nil)
