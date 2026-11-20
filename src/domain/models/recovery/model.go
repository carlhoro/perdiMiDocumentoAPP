// Package recovery models document recovery independently of transport and storage.
package recovery

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
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

// IdentityInput is transient. Never persist or log its number.
type IdentityInput struct{ Country, Type, Number string }

func (IdentityInput) String() string   { return "[redacted identity]" }
func (IdentityInput) GoString() string { return "[redacted identity]" }
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

type ProtectedReference struct{ Value, KeyVersion, NormalizationVersion string }

func (r ProtectedReference) Valid() bool {
	return r.Value != "" && r.KeyVersion != "" && r.NormalizationVersion != ""
}

type Document struct {
	ID            int64
	Country, Type string
	References    []ProtectedReference
}

type Visibility string

const (
	Restricted Visibility = "restricted"
	Public     Visibility = "public"
)

type Location struct {
	ID                                                     int64
	Name, Address, Municipality, Department, Country, Zone string
	Visibility                                             Visibility
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

// LocationSummary deliberately omits identity, actor and internal IDs.
type LocationSummary struct{ Name, Address, Municipality, Department, Country, Zone string }

func (l Location) Summary() LocationSummary {
	s := LocationSummary{Municipality: l.Municipality, Department: l.Department, Country: l.Country, Zone: l.Zone}
	if l.Visibility == Public {
		s.Name = l.Name
		s.Address = l.Address
	}
	return s
}

type ReportStatus string

const (
	Reported  ReportStatus = "reported"
	Confirmed ReportStatus = "confirmed"
	Recovered ReportStatus = "recovered"
	Stale     ReportStatus = "stale"
	Rejected  ReportStatus = "rejected"
)

type Transition struct {
	From, To ReportStatus
	ActorID  int64
	Reason   string
	At       time.Time
}
type FoundDocumentReport struct {
	ID, DocumentID, LocationID, ReportedBy int64
	ObservedAt, CreatedAt                  time.Time
	Status                                 ReportStatus
	Version                                int64
	History                                []Transition
}

func NewReport(documentID, locationID, actorID int64, observed, now time.Time) (FoundDocumentReport, error) {
	if documentID <= 0 || locationID <= 0 || actorID <= 0 || observed.IsZero() || observed.After(now) {
		return FoundDocumentReport{}, ErrInvalid
	}
	return FoundDocumentReport{DocumentID: documentID, LocationID: locationID, ReportedBy: actorID, ObservedAt: observed.UTC(), CreatedAt: now.UTC(), Status: Reported, Version: 1}, nil
}
func (r FoundDocumentReport) Available(now time.Time, ttl time.Duration) bool {
	return (r.Status == Reported || r.Status == Confirmed) && now.Before(r.ObservedAt.Add(ttl))
}
func (r *FoundDocumentReport) Change(to ReportStatus, actorID int64, moderator, owner bool, reason string, now time.Time) error {
	if actorID <= 0 {
		return ErrUnauthorized
	}
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

type VerifiedOwnership struct {
	UserID, DocumentID        int64
	Issuer, EvidenceReference string
	VerifiedAt, ExpiresAt     time.Time
	Revoked                   bool
}

func (v VerifiedOwnership) Valid(userID, documentID int64, now time.Time) bool {
	return !v.Revoked && v.UserID == userID && v.DocumentID == documentID && v.Issuer != "" && v.EvidenceReference != "" && !v.VerifiedAt.IsZero() && !now.Before(v.VerifiedAt) && now.Before(v.ExpiresAt)
}

type WatchStatus string

const (
	Active    WatchStatus = "active"
	Suspended WatchStatus = "suspended"
	Stopped   WatchStatus = "stopped"
)

type WatchedDocument struct {
	ID, UserID, DocumentID int64
	Status                 WatchStatus
	Ownership              VerifiedOwnership
}
type Notification struct {
	ID, UserID, WatchID, ReportID int64
	Kind                          string
	CreatedAt                     time.Time
}
type Availability struct {
	Location  LocationSummary
	Status    ReportStatus
	ExpiresAt time.Time
}
type NotificationView struct {
	ID        int64
	Match     Availability
	CreatedAt time.Time
}

// PotentialContradiction detects concurrent location claims without deciding custody.
func PotentialContradiction(reports []FoundDocumentReport, now time.Time, ttl time.Duration) bool {
	locations := map[int64]bool{}
	for _, r := range reports {
		if r.Available(now, ttl) {
			locations[r.LocationID] = true
		}
	}
	return len(locations) > 1
}
