package report

import (
	"strings"
	"time"
	"unicode/utf8"

	"perdimidocumento/src/domain/models/valueobjects"
	"perdimidocumento/src/domain/recoveryerrors"
)

// FoundDocumentReport registra un avistamiento de un documento en una ubicación.
// ObservedAt marca cuándo se vio el documento. CreatedAt marca cuándo se registró el reporte.
type FoundDocumentReport struct {
	ID, DocumentID, LocationID, ReportedBy int64
	ObservedAt, CreatedAt                  time.Time
	Status                                 valueobjects.ReportStatus
	Version                                int64
	History                                []valueobjects.Transition
}

// Available calcula la vigencia desde el avistamiento. El reporte vence justo al cumplirse el plazo.
func (r FoundDocumentReport) Available(now time.Time, ttl time.Duration) bool {
	return (r.Status == valueobjects.Reported || r.Status == valueobjects.Confirmed) && now.Before(r.ObservedAt.Add(ttl))
}

// Change valida la transición y su motivo antes de registrar el cambio.
func (r *FoundDocumentReport) Change(to valueobjects.ReportStatus, actorID int64, moderator, owner bool, reason string, now time.Time) error {
	if err := r.validateTransition(to, actorID, moderator, owner); err != nil {
		return err
	}
	reason, err := normalizeTransitionReason(reason)
	if err != nil {
		return err
	}
	r.recordTransition(to, actorID, reason, now)
	return nil
}

// validateTransition comprueba el estado del reporte y los permisos del actor para el destino solicitado.
func (r *FoundDocumentReport) validateTransition(to valueobjects.ReportStatus, actorID int64, moderator, owner bool) error {
	if actorID <= 0 {
		return recoveryerrors.ErrUnauthorized
	}
	// La confirmación de custodia requiere acreditar establecimientos y sigue pendiente de implementación.
	if to == valueobjects.Confirmed {
		return recoveryerrors.ErrCustodyUnavailable
	}
	if r.Status != valueobjects.Reported && r.Status != valueobjects.Confirmed {
		return recoveryerrors.ErrConflict
	}
	var allowed bool
	switch to {
	case valueobjects.Recovered:
		allowed = owner
	case valueobjects.Stale:
		allowed = moderator || actorID == r.ReportedBy
	case valueobjects.Rejected:
		allowed = moderator
	default:
		return recoveryerrors.ErrInvalid
	}
	if !allowed {
		return recoveryerrors.ErrForbidden
	}
	return nil
}

// normalizeTransitionReason quita los espacios de los extremos y exige entre 1 y 200 caracteres Unicode.
func normalizeTransitionReason(reason string) (string, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" || utf8.RuneCountInString(reason) > 200 {
		return "", recoveryerrors.ErrInvalid
	}
	return reason, nil
}

// recordTransition guarda el estado anterior en el historial antes de actualizar el estado y la versión.
func (r *FoundDocumentReport) recordTransition(to valueobjects.ReportStatus, actorID int64, reason string, now time.Time) {
	r.History = append(r.History, valueobjects.Transition{From: r.Status, To: to, ActorID: actorID, Reason: reason, At: now.UTC()})
	r.Status = to
	r.Version++
}

// PotentialContradiction señala reportes vigentes en ubicaciones distintas para el mismo documento.
// La señal permite revisar los reportes y deja abierta la confirmación de custodia.
func PotentialContradiction(reports []FoundDocumentReport, now time.Time, ttl time.Duration) bool {
	locations := map[int64]bool{}
	for _, r := range reports {
		if r.Available(now, ttl) {
			locations[r.LocationID] = true
		}
	}
	return len(locations) > 1
}
