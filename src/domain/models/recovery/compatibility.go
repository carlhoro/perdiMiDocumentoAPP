// Package recovery conserva los nombres usados por los consumidores del flujo de recuperación.
package recovery

import (
	"time"

	"perdimidocumento/src/domain/factories"
	"perdimidocumento/src/domain/models/document"
	"perdimidocumento/src/domain/models/location"
	"perdimidocumento/src/domain/models/notification"
	"perdimidocumento/src/domain/models/report"
	"perdimidocumento/src/domain/models/valueobjects"
	"perdimidocumento/src/domain/models/watch"
	"perdimidocumento/src/domain/recoveryerrors"
)

type (
	Document            = document.DocumentIdentity
	Location            = location.Location
	FoundDocumentReport = report.FoundDocumentReport
	Availability        = report.Availability
	WatchedDocument     = watch.WatchedDocument
	Notification        = notification.MatchNotification
	NotificationView    = notification.NotificationView

	IdentityInput      = valueobjects.IdentityInput
	ProtectedReference = valueobjects.ProtectedReference
	Visibility         = valueobjects.Visibility
	LocationSummary    = valueobjects.LocationSummary
	ReportStatus       = valueobjects.ReportStatus
	Transition         = valueobjects.Transition
	VerifiedOwnership  = valueobjects.VerifiedOwnership
	WatchStatus        = valueobjects.WatchStatus
)

const (
	Restricted = valueobjects.Restricted
	Public     = valueobjects.Public

	Reported  = valueobjects.Reported
	Confirmed = valueobjects.Confirmed
	Recovered = valueobjects.Recovered
	Stale     = valueobjects.Stale
	Rejected  = valueobjects.Rejected

	Active    = valueobjects.Active
	Suspended = valueobjects.Suspended
	Stopped   = valueobjects.Stopped
)

var (
	ErrInvalid            = recoveryerrors.ErrInvalid
	ErrUnauthorized       = recoveryerrors.ErrUnauthorized
	ErrForbidden          = recoveryerrors.ErrForbidden
	ErrConflict           = recoveryerrors.ErrConflict
	ErrNotFound           = recoveryerrors.ErrNotFound
	ErrOwnership          = recoveryerrors.ErrOwnership
	ErrCapacity           = recoveryerrors.ErrCapacity
	ErrCustodyUnavailable = recoveryerrors.ErrCustodyUnavailable
)

// NewReport delega la creación del reporte a factories.NewReport.
func NewReport(documentID, locationID, actorID int64, observed, now time.Time) (FoundDocumentReport, error) {
	return factories.NewReport(documentID, locationID, actorID, observed, now)
}

// PotentialContradiction aplica la regla de contradicción del paquete report.
func PotentialContradiction(reports []FoundDocumentReport, now time.Time, ttl time.Duration) bool {
	return report.PotentialContradiction(reports, now, ttl)
}
