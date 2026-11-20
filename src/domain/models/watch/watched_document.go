package watch

import "perdimidocumento/src/domain/models/valueobjects"

// WatchedDocument guarda el seguimiento del usuario y la evidencia de titularidad asociada.
type WatchedDocument struct {
	ID, UserID, DocumentID int64
	Status                 valueobjects.WatchStatus
	Ownership              valueobjects.VerifiedOwnership
}
