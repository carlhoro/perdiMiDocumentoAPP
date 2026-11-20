package factories

import (
	"time"

	"perdimidocumento/src/domain/models/report"
	"perdimidocumento/src/domain/models/valueobjects"
	"perdimidocumento/src/domain/recoveryerrors"
)

// NewReport crea un reporte inicial y valida que el avistamiento tenga una fecha válida.
func NewReport(documentID, locationID, actorID int64, observed, now time.Time) (report.FoundDocumentReport, error) {
	if documentID <= 0 || locationID <= 0 || actorID <= 0 || observed.IsZero() || observed.After(now) {
		return report.FoundDocumentReport{}, recoveryerrors.ErrInvalid
	}
	return report.FoundDocumentReport{
		DocumentID: documentID,
		LocationID: locationID,
		ReportedBy: actorID,
		ObservedAt: observed.UTC(),
		CreatedAt:  now.UTC(),
		Status:     valueobjects.Reported,
		Version:    1,
	}, nil
}
