package tests

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"perdimidocumento/src/domain/models/recovery"
)

// TestIdentityNormalization cubre el formato admitido y el ocultamiento del número al imprimir la identidad.
func TestIdentityNormalization(t *testing.T) {
	i, e := (recovery.IdentityInput{Country: " co ", Type: "cc", Number: " 00123 "}).Normalize()
	if e != nil || i.Number != "00123" || i.Country != "CO" {
		t.Fatal(i, e)
	}
	for _, i := range []recovery.IdentityInput{{Country: "US", Type: "CC", Number: "1"}, {Country: "CO", Type: "TI", Number: "1"}, {Country: "CO", Type: "CC", Number: "1-2"}, {Country: "CO", Type: "CC", Number: "１２"}, {Country: "CO", Type: "CC", Number: ""}, {Country: "CO", Type: "CC", Number: strings.Repeat("1", 201)}} {
		if _, e := i.Normalize(); !errors.Is(e, recovery.ErrInvalid) {
			t.Fatal("accepted invalid identity")
		}
	}
	for _, format := range []string{"%v", "%+v", "%#v"} {
		if strings.Contains(fmt.Sprintf(format, i), "00123") {
			t.Fatal("identity leaked")
		}
	}
}

// TestReportLifecycleAndExpiry comprueba el instante exacto de vencimiento y los permisos de transición.
func TestReportLifecycleAndExpiry(t *testing.T) {
	now := time.Date(2026, 11, 19, 0, 0, 0, 0, time.UTC)
	r, e := recovery.NewReport(1, 1, 2, now, now)
	if e != nil {
		t.Fatal(e)
	}
	ttl := 30 * 24 * time.Hour
	if !r.Available(now.Add(ttl-time.Nanosecond), ttl) || r.Available(now.Add(ttl), ttl) {
		t.Fatal("expiry boundary")
	}
	if e = r.Change(recovery.Confirmed, 2, false, false, "seen", now); !errors.Is(e, recovery.ErrCustodyUnavailable) {
		t.Fatal(e)
	}
	if e = r.Change(recovery.Recovered, 2, false, false, "mine", now); !errors.Is(e, recovery.ErrForbidden) {
		t.Fatal(e)
	}
	if e = r.Change(recovery.Rejected, 2, false, false, "bad", now); !errors.Is(e, recovery.ErrForbidden) {
		t.Fatal(e)
	}
	if e = r.Change(recovery.Stale, 2, false, false, "no longer seen", now); e != nil {
		t.Fatal(e)
	}
	if len(r.History) != 1 || r.Version != 2 {
		t.Fatal(r)
	}
	if e = r.Change(recovery.Reported, 2, true, true, "reopen", now); !errors.Is(e, recovery.ErrConflict) {
		t.Fatal(e)
	}
}

// TestRestrictedLocationSummary comprueba qué datos se muestran con visibilidad predeterminada y pública.
func TestRestrictedLocationSummary(t *testing.T) {
	l := recovery.Location{Name: "Private name", Address: "Private address", Country: "CO", Municipality: "Bogota", Department: "Bogota", Zone: "North"}
	if e := l.Validate(); e != nil {
		t.Fatal(e)
	}
	if s := l.Summary(); s.Name != "" || s.Address != "" || s.Municipality == "" {
		t.Fatal(s)
	}
	l.Visibility = recovery.Public
	if l.Summary().Address != l.Address {
		t.Fatal("public establishment hidden")
	}
}

// TestOwnershipMustBindExactUserAndDocument exige la pareja exacta y rechaza evidencia vencida o revocada.
func TestOwnershipMustBindExactUserAndDocument(t *testing.T) {
	now := time.Date(2026, 11, 19, 0, 0, 0, 0, time.UTC)
	proof := recovery.VerifiedOwnership{UserID: 1, DocumentID: 2, Issuer: "authority", EvidenceReference: "proof", VerifiedAt: now.Add(-time.Hour), ExpiresAt: now.Add(time.Hour)}
	if !proof.Valid(1, 2, now) || proof.Valid(3, 2, now) || proof.Valid(1, 3, now) || proof.Valid(1, 2, proof.ExpiresAt) {
		t.Fatal("invalid ownership boundary")
	}
	proof.Revoked = true
	if proof.Valid(1, 2, now) {
		t.Fatal("revoked ownership accepted")
	}
}
