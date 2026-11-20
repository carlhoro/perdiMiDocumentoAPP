package models

import "time"

// Document reóne los datos registrados de un documento encontrado.
type Document struct {
	ID         int64
	Number     string
	Type       string
	OwnerName  string
	Address    string
	ReportedBy int64
	CreatedAt  time.Time
}
