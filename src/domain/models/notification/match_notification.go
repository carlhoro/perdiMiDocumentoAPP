package notification

import (
	"time"

	"perdimidocumento/src/domain/models/report"
)

// MatchNotification guarda la coincidencia por sus IDs para armar la vista con los datos vigentes al consultarla.
type MatchNotification struct {
	ID, UserID, WatchID, ReportID int64
	Kind                          string
	CreatedAt                     time.Time
}

// NotificationView combina la fecha del aviso con la disponibilidad calculada al consultar.
type NotificationView struct {
	ID        int64
	Match     report.Availability
	CreatedAt time.Time
}
