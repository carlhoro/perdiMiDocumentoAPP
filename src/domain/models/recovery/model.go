// Package recovery define las entidades y reglas del dominio de recuperación de documentos.
package recovery

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

// IdentityInput lleva el número original solo durante el procesamiento de la solicitud.
// La persistencia y los registros deben usar referencias protegidas o identificadores internos.
type IdentityInput struct{ Country, Type, Number string }

// ProtectedReference identifica un documento mediante un HMAC y las versiones de clave y normalización usadas.
type ProtectedReference struct{ Value, KeyVersion, NormalizationVersion string }

// Document representa una identidad documental que puede tener varios hallazgos y referencias protegidas.
type Document struct {
	ID            int64
	Country, Type string
	References    []ProtectedReference
}

// Visibility determina qué detalle de una ubicación puede mostrarse.
type Visibility string

// Location describe el lugar donde se reportó un documento.
type Location struct {
	ID                                                     int64
	Name, Address, Municipality, Department, Country, Zone string
	Visibility                                             Visibility
}

// LocationSummary contiene los datos de ubicación que pueden aparecer en búsquedas y avisos.
type LocationSummary struct{ Name, Address, Municipality, Department, Country, Zone string }

// ReportStatus identifica el estado de un reporte de hallazgo.
type ReportStatus string

// Transition registra el motivo y el actor de un cambio de estado.
type Transition struct {
	From, To ReportStatus
	ActorID  int64
	Reason   string
	At       time.Time
}

// FoundDocumentReport registra un avistamiento de un documento en una ubicación.
// ObservedAt marca cuándo se vio el documento. CreatedAt marca cuándo se registró el reporte.
type FoundDocumentReport struct {
	ID, DocumentID, LocationID, ReportedBy int64
	ObservedAt, CreatedAt                  time.Time
	Status                                 ReportStatus
	Version                                int64
	History                                []Transition
}

// VerifiedOwnership representa la evidencia de titularidad emitida para una pareja de usuario y documento.
type VerifiedOwnership struct {
	UserID, DocumentID        int64
	Issuer, EvidenceReference string
	VerifiedAt, ExpiresAt     time.Time
	Revoked                   bool
}

// WatchStatus indica si un seguimiento está activo, suspendido o detenido.
type WatchStatus string

// WatchedDocument guarda el seguimiento del usuario y la evidencia de titularidad asociada.
type WatchedDocument struct {
	ID, UserID, DocumentID int64
	Status                 WatchStatus
	Ownership              VerifiedOwnership
}

// Notification guarda la coincidencia por sus IDs para armar la vista con los datos vigentes al consultarla.
type Notification struct {
	ID, UserID, WatchID, ReportID int64
	Kind                          string
	CreatedAt                     time.Time
}

// Availability resume el estado, el vencimiento y la ubicación visible de un hallazgo.
type Availability struct {
	Location  LocationSummary
	Status    ReportStatus
	ExpiresAt time.Time
}

// NotificationView combina la fecha del aviso con la disponibilidad calculada al consultar.
type NotificationView struct {
	ID        int64
	Match     Availability
	CreatedAt time.Time
}

const (
	Restricted Visibility = "restricted"
	Public     Visibility = "public"
)

const (
	Reported  ReportStatus = "reported"
	Confirmed ReportStatus = "confirmed"
	Recovered ReportStatus = "recovered"
	Stale     ReportStatus = "stale"
	Rejected  ReportStatus = "rejected"
)

const (
	Active    WatchStatus = "active"
	Suspended WatchStatus = "suspended"
	Stopped   WatchStatus = "stopped"
)

var (
	ErrInvalid            = errors.New("invalid recovery input")
	ErrUnauthorized       = errors.New("authentication required")
	ErrForbidden          = errors.New("operation not authorized")
	ErrConflict           = errors.New("conflicting operation")
	ErrNotFound           = errors.New("recovery record not found")
	ErrOwnership          = errors.New("verified ownership required")
	ErrCapacity           = errors.New("watch capacity exhausted")
	ErrCustodyUnavailable = errors.New("establishment custody confirmation unavailable")
)

// String y GoString ocultan la identidad en los formatos de texto habituales de fmt.
func (IdentityInput) String() string { return "[redacted identity]" }

func (IdentityInput) GoString() string { return "[redacted identity]" }

// Normalize admite cédulas de ciudadanía colombianas. Conserva el número como texto para preservar los ceros iniciales.
func (i IdentityInput) Normalize() (IdentityInput, error) {
	i.Country = strings.ToUpper(strings.TrimSpace(i.Country))
	i.Type = strings.ToUpper(strings.TrimSpace(i.Type))
	i.Number = strings.TrimSpace(i.Number)
	if i.Country != "CO" || i.Type != "CC" || len(i.Number) == 0 || len(i.Number) > 200 {
		return IdentityInput{}, ErrInvalid
	}
	for _, r := range i.Number {
		if r < '0' || r > '9' {
			return IdentityInput{}, ErrInvalid
		}
	}
	return i, nil
}

// Valid comprueba que la referencia y sus versiones estén presentes.
func (r ProtectedReference) Valid() bool {
	return r.Value != "" && r.KeyVersion != "" && r.NormalizationVersion != ""
}

func (l Location) Validate() error {
	for _, s := range []string{l.Name, l.Address, l.Municipality, l.Department} {
		if strings.TrimSpace(s) == "" || utf8.RuneCountInString(s) > 200 {
			return ErrInvalid
		}
	}
	if l.Country != "CO" || len(l.Zone) > 200 || (l.Visibility != "" && l.Visibility != Restricted && l.Visibility != Public) {
		return ErrInvalid
	}
	return nil
}

// Summary incluye nombre y dirección exacta únicamente para ubicaciones marcadas como públicas.
func (l Location) Summary() LocationSummary {
	s := LocationSummary{Municipality: l.Municipality, Department: l.Department, Country: l.Country, Zone: l.Zone}
	if l.Visibility == Public {
		s.Name = l.Name
		s.Address = l.Address
	}
	return s
}

// NewReport crea un reporte inicial y valida que el avistamiento tenga una fecha válida.
func NewReport(documentID, locationID, actorID int64, observed, now time.Time) (FoundDocumentReport, error) {
	if documentID <= 0 || locationID <= 0 || actorID <= 0 || observed.IsZero() || observed.After(now) {
		return FoundDocumentReport{}, ErrInvalid
	}
	return FoundDocumentReport{DocumentID: documentID, LocationID: locationID, ReportedBy: actorID, ObservedAt: observed.UTC(), CreatedAt: now.UTC(), Status: Reported, Version: 1}, nil
}

// Available calcula la vigencia desde el avistamiento. El reporte vence justo al cumplirse el plazo.
func (r FoundDocumentReport) Available(now time.Time, ttl time.Duration) bool {
	return (r.Status == Reported || r.Status == Confirmed) && now.Before(r.ObservedAt.Add(ttl))
}

// Change aplica las reglas de transición del reporte según el actor y el estado solicitado.
func (r *FoundDocumentReport) Change(to ReportStatus, actorID int64, moderator, owner bool, reason string, now time.Time) error {
	if actorID <= 0 {
		return ErrUnauthorized
	}
	// La confirmación de custodia requiere acreditar establecimientos y sigue pendiente de implementación.
	if to == Confirmed {
		return ErrCustodyUnavailable
	}
	if r.Status != Reported && r.Status != Confirmed {
		return ErrConflict
	}
	allowed := false
	switch to {
	case Recovered:
		allowed = owner
	case Stale:
		allowed = moderator || actorID == r.ReportedBy
	case Rejected:
		allowed = moderator
	default:
		return ErrInvalid
	}
	if !allowed {
		return ErrForbidden
	}
	reason = strings.TrimSpace(reason)
	if reason == "" || utf8.RuneCountInString(reason) > 200 {
		return ErrInvalid
	}
	r.History = append(r.History, Transition{From: r.Status, To: to, ActorID: actorID, Reason: reason, At: now.UTC()})
	r.Status = to
	r.Version++
	return nil
}

// Valid exige la pareja exacta y evidencia vigente. La fecha de emisión se incluye en el intervalo y la de vencimiento lo cierra.
func (v VerifiedOwnership) Valid(userID, documentID int64, now time.Time) bool {
	return !v.Revoked && v.UserID == userID && v.DocumentID == documentID && v.Issuer != "" && v.EvidenceReference != "" && !v.VerifiedAt.IsZero() && !now.Before(v.VerifiedAt) && now.Before(v.ExpiresAt)
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
