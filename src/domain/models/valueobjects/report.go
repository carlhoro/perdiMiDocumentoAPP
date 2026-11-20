package valueobjects

import "time"

// ReportStatus identifica el estado de un reporte de hallazgo.
type ReportStatus string

// Transition registra el motivo y el actor de un cambio de estado.
type Transition struct {
	From, To ReportStatus
	ActorID  int64
	Reason   string
	At       time.Time
}

const (
	Reported  ReportStatus = "reported"
	Confirmed ReportStatus = "confirmed"
	Recovered ReportStatus = "recovered"
	Stale     ReportStatus = "stale"
	Rejected  ReportStatus = "rejected"
)
