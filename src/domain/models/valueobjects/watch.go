package valueobjects

import "time"

// WatchStatus indica si un seguimiento está activo, suspendido o detenido.
type WatchStatus string

// VerifiedOwnership representa la evidencia de titularidad emitida para una pareja de usuario y documento.
type VerifiedOwnership struct {
	UserID, DocumentID        int64
	Issuer, EvidenceReference string
	VerifiedAt, ExpiresAt     time.Time
	Revoked                   bool
}

const (
	Active    WatchStatus = "active"
	Suspended WatchStatus = "suspended"
	Stopped   WatchStatus = "stopped"
)

// Valid exige la pareja exacta y evidencia vigente. La fecha de emisión se incluye en el intervalo y la de vencimiento lo cierra.
func (v VerifiedOwnership) Valid(userID, documentID int64, now time.Time) bool {
	return !v.Revoked && v.UserID == userID && v.DocumentID == documentID && v.Issuer != "" && v.EvidenceReference != "" && !v.VerifiedAt.IsZero() && !now.Before(v.VerifiedAt) && now.Before(v.ExpiresAt)
}
