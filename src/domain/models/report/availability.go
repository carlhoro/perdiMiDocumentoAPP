package report

import (
	"time"

	"perdimidocumento/src/domain/models/valueobjects"
)

// Availability resume el estado, el vencimiento y la ubicación visible de un hallazgo.
type Availability struct {
	Location  valueobjects.LocationSummary
	Status    valueobjects.ReportStatus
	ExpiresAt time.Time
}
