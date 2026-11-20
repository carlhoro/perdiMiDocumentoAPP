package document

import "time"

// FoundDocument reúne los datos registrados de un documento encontrado.
type FoundDocument struct {
	ID         int64
	Number     string
	Type       string
	OwnerName  string
	Address    string
	ReportedBy int64
	CreatedAt  time.Time
}
