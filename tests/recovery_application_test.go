package tests

import (
	"context"
	"errors"
	"fmt"
	app "perdimidocumento/src/application/recovery"
	m "perdimidocumento/src/domain/models/recovery"
	p "perdimidocumento/src/domain/ports/recovery"
	"sync"
	"testing"
	"time"
)

// This serialized transactional double is deliberately test-only, not a storage adapter.
type recoveryMemory struct {
	mu        sync.Mutex
	docs      []m.Document
	locations []m.Location
	reports   []m.FoundDocumentReport
	watches   []m.WatchedDocument
	notes     []m.Notification
	ops       []p.Operation
	audits    []p.AuditEntry
	fail      bool
}

func (s *recoveryMemory) Within(ctx context.Context, f func(p.Store) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e := ctx.Err(); e != nil {
		return e
	}
	tx := &recoveryMemory{docs: append([]m.Document{}, s.docs...), locations: append([]m.Location{}, s.locations...), reports: append([]m.FoundDocumentReport{}, s.reports...), watches: append([]m.WatchedDocument{}, s.watches...), notes: append([]m.Notification{}, s.notes...), ops: append([]p.Operation{}, s.ops...), audits: append([]p.AuditEntry{}, s.audits...), fail: s.fail}
	if e := f(tx); e != nil {
		return e
	}
	s.docs = tx.docs
	s.locations = tx.locations
	s.reports = tx.reports
	s.watches = tx.watches
	s.notes = tx.notes
	s.ops = tx.ops
	s.audits = tx.audits
	return nil
}
func (s *recoveryMemory) Document(_ context.Context, id int64) (m.Document, error) {
	for _, d := range s.docs {
		if d.ID == id {
			return d, nil
		}
	}
	return m.Document{}, m.ErrNotFound
}
func (s *recoveryMemory) ResolveDocument(_ context.Context, c, k string, refs []m.ProtectedReference, create bool) (m.Document, error) {
	for _, d := range s.docs {
		for _, a := range d.References {
			for _, b := range refs {
				if c == d.Country && k == d.Type && a == b {
					return d, nil
				}
			}
		}
	}
	if !create {
		return m.Document{}, m.ErrNotFound
	}
	d := m.Document{ID: int64(len(s.docs) + 1), Country: c, Type: k, References: refs}
	s.docs = append(s.docs, d)
	return d, nil
}
func (s *recoveryMemory) SaveLocation(_ context.Context, l m.Location) (m.Location, error) {
	l.ID = int64(len(s.locations) + 1)
	s.locations = append(s.locations, l)
	return l, nil
}
func (s *recoveryMemory) Location(_ context.Context, id int64) (m.Location, error) {
	for _, l := range s.locations {
		if l.ID == id {
			return l, nil
		}
	}
	return m.Location{}, m.ErrNotFound
}
func (s *recoveryMemory) SaveReport(_ context.Context, r m.FoundDocumentReport) (m.FoundDocumentReport, error) {
	r.History = append([]m.Transition{}, r.History...)
	if r.ID == 0 {
		r.ID = int64(len(s.reports) + 1)
		s.reports = append(s.reports, r)
		return r, nil
	}
	for i, old := range s.reports {
		if old.ID == r.ID {
			if r.Version != old.Version+1 {
				return r, m.ErrConflict
			}
			s.reports[i] = r
			return r, nil
		}
	}
	return r, m.ErrNotFound
}
func (s *recoveryMemory) Report(_ context.Context, id int64) (m.FoundDocumentReport, error) {
	for _, r := range s.reports {
		if r.ID == id {
			r.History = append([]m.Transition{}, r.History...)
			return r, nil
		}
	}
	return m.FoundDocumentReport{}, m.ErrNotFound
}
func (s *recoveryMemory) Reports(_ context.Context, id int64) (out []m.FoundDocumentReport, e error) {
	for _, r := range s.reports {
		if r.DocumentID == id {
			out = append(out, r)
		}
	}
	return
}
func (s *recoveryMemory) SaveWatch(_ context.Context, w m.WatchedDocument) (m.WatchedDocument, error) {
	for i, old := range s.watches {
		if old.UserID == w.UserID && old.DocumentID == w.DocumentID {
			w.ID = old.ID
			s.watches[i] = w
			return w, nil
		}
	}
	w.ID = int64(len(s.watches) + 1)
	s.watches = append(s.watches, w)
	return w, nil
}
func (s *recoveryMemory) Watches(_ context.Context, id int64) (out []m.WatchedDocument, e error) {
	for _, w := range s.watches {
		if w.DocumentID == id {
			out = append(out, w)
		}
	}
	return
}
func (s *recoveryMemory) UserWatches(_ context.Context, id int64) (out []m.WatchedDocument, e error) {
	for _, w := range s.watches {
		if w.UserID == id {
			out = append(out, w)
		}
	}
	return
}
func (s *recoveryMemory) InsertNotification(_ context.Context, n m.Notification) error {
	if s.fail {
		return errors.New("test storage failure")
	}
	for _, old := range s.notes {
		if old.UserID == n.UserID && old.ReportID == n.ReportID && old.Kind == n.Kind {
			return nil
		}
	}
	n.ID = int64(len(s.notes) + 1)
	s.notes = append(s.notes, n)
	return nil
}
func (s *recoveryMemory) Notifications(_ context.Context, id int64) (out []m.Notification, e error) {
	for _, n := range s.notes {
		if n.UserID == id {
			out = append(out, n)
		}
	}
	return
}
func (s *recoveryMemory) Audit(_ context.Context, a p.AuditEntry) error {
	s.audits = append(s.audits, a)
	return nil
}
func (s *recoveryMemory) Operation(_ context.Context, u int64, key string) (p.Operation, error) {
	for _, o := range s.ops {
		if o.ActorID == u && o.Key == key {
			return o, nil
		}
	}
	return p.Operation{}, m.ErrNotFound
}
func (s *recoveryMemory) SaveOperation(_ context.Context, o p.Operation) error {
	s.ops = append(s.ops, o)
	return nil
}

type recoveryClock struct{ now time.Time }

func (c *recoveryClock) Now() time.Time { return c.now }

type recoveryAuth struct{}

func (recoveryAuth) Authenticated(_ context.Context, u int64) (bool, error) { return u > 0, nil }
func (recoveryAuth) Moderator(_ context.Context, u int64) (bool, error)     { return u == 99, nil }

type recoveryProtector struct{}

func (recoveryProtector) Lookup(_ context.Context, i m.IdentityInput) ([]m.ProtectedReference, error) {
	return []m.ProtectedReference{{Value: "TEST-ONLY:" + i.Number, KeyVersion: "fake", NormalizationVersion: "1"}}, nil
}

type recoveryOwnership struct {
	clock   *recoveryClock
	revoked bool
}

func (v *recoveryOwnership) Verify(_ context.Context, u int64, d m.Document) (m.VerifiedOwnership, error) {
	return m.VerifiedOwnership{UserID: 1, DocumentID: d.ID, Issuer: "test", EvidenceReference: "evidence", VerifiedAt: v.clock.now.Add(-time.Hour), ExpiresAt: v.clock.now.Add(time.Hour), Revoked: v.revoked}, nil
}
func recoverySetup(t *testing.T) (app.Service, *recoveryMemory, *recoveryClock, *recoveryOwnership) {
	t.Helper()
	clock := &recoveryClock{time.Date(2026, 11, 19, 0, 0, 0, 0, time.UTC)}
	store := &recoveryMemory{}
	ownership := &recoveryOwnership{clock: clock}
	s := app.Service{Transactions: store, Identity: recoveryProtector{}, Ownership: ownership, Actors: recoveryAuth{}, Clock: clock}
	_, e := s.SaveLocation(context.Background(), 2, m.Location{Name: "Shop", Address: "Street", Municipality: "Bogota", Department: "Bogota", Country: "CO"})
	if e != nil {
		t.Fatal(e)
	}
	return s, store, clock, ownership
}
func recoveryInput() m.IdentityInput {
	return m.IdentityInput{Country: "CO", Type: "CC", Number: "00123"}
}

type recoveryCapacity int

func (c recoveryCapacity) Capacity(context.Context, int64) (int, error) { return int(c), nil }
func TestRecoveryConcurrentActivationUsesOneSlot(t *testing.T) {
	s, store, _, _ := recoverySetup(t)
	ctx := context.Background()
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, number := range []string{"00123", "00999"} {
		number := number
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, e := s.Activate(ctx, 1, m.IdentityInput{Country: "CO", Type: "CC", Number: number})
			errs <- e
		}()
	}
	wg.Wait()
	close(errs)
	successes := 0
	for e := range errs {
		if e == nil {
			successes++
		} else if !errors.Is(e, m.ErrConflict) && !errors.Is(e, m.ErrCapacity) {
			t.Fatal(e)
		}
	}
	if successes != 1 || len(store.watches) != 1 || len(store.docs) != 1 {
		t.Fatal("concurrent activation exceeded capacity")
	}
}
func TestRecoveryCapacityAndUnavailableVerifier(t *testing.T) {
	s, store, _, _ := recoverySetup(t)
	ctx := context.Background()
	s.Ownership = nil
	if _, e := s.Activate(ctx, 1, recoveryInput()); !errors.Is(e, m.ErrOwnership) {
		t.Fatal(e)
	}
	if len(store.watches) != 0 {
		t.Fatal("watch without verifier")
	}
	s, store, _, _ = recoverySetup(t)
	s.Capacity = recoveryCapacity(0)
	if _, e := s.Activate(ctx, 1, recoveryInput()); !errors.Is(e, m.ErrCapacity) {
		t.Fatal(e)
	}
	if len(store.watches) != 0 || len(store.docs) != 0 {
		t.Fatal("capacity rollback")
	}
	s.Capacity = recoveryCapacity(1)
	if _, e := s.Activate(ctx, 1, recoveryInput()); e != nil {
		t.Fatal(e)
	}
	s.Capacity = recoveryCapacity(0)
	if _, e := s.Activate(ctx, 1, recoveryInput()); e != nil {
		t.Fatal("downgrade canceled existing watch", e)
	}
}
func TestRecoveryHistoricalReportsAndContradictions(t *testing.T) {
	s, store, clock, _ := recoverySetup(t)
	ctx := context.Background()
	if _, e := s.Register(ctx, 2, p.RegisterReport{Identity: recoveryInput(), LocationID: 1, ObservedAt: clock.now.Add(-31 * 24 * time.Hour), OperationKey: "old"}); e != nil {
		t.Fatal(e)
	}
	if store.reports[0].Status != m.Stale {
		t.Fatal("old sighting became available")
	}
	_, e := s.Register(ctx, 3, p.RegisterReport{Identity: recoveryInput(), LocationID: 1, ObservedAt: clock.now, OperationKey: "new"})
	if e != nil {
		t.Fatal(e)
	}
	l, e := s.SaveLocation(ctx, 3, m.Location{Name: "Other shop", Address: "Other street", Municipality: "Bogota", Department: "Bogota", Country: "CO", Visibility: m.Public})
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.Register(ctx, 3, p.RegisterReport{Identity: recoveryInput(), LocationID: l.ID, ObservedAt: clock.now, OperationKey: "other"})
	if e != nil {
		t.Fatal(e)
	}
	if !m.PotentialContradiction(store.reports, clock.now, 30*24*time.Hour) {
		t.Fatal("contradiction lost")
	}
	out, e := s.Search(ctx, 2, recoveryInput())
	if e != nil || len(out) != 2 {
		t.Fatal(out, e)
	}
}
func TestRecoveryObserverBothOrdersAndReactivation(t *testing.T) {
	for _, first := range []bool{true, false} {
		t.Run(fmt.Sprint(first), func(t *testing.T) {
			s, store, clock, _ := recoverySetup(t)
			ctx := context.Background()
			activate := func() {
				t.Helper()
				if _, e := s.Activate(ctx, 1, recoveryInput()); e != nil {
					t.Fatal(e)
				}
			}
			if first {
				activate()
			}
			c := p.RegisterReport{Identity: recoveryInput(), LocationID: 1, ObservedAt: clock.now, OperationKey: "one"}
			r, e := s.Register(ctx, 2, c)
			if e != nil {
				t.Fatal(e)
			}
			activate()
			if _, e = s.Register(ctx, 2, c); e != nil {
				t.Fatal(e)
			}
			c.OperationKey = "two"
			if r2, e := s.Register(ctx, 2, c); e != nil || r.ID != r2.ID {
				t.Fatal(r2, e)
			}
			if e = s.Stop(ctx, 1, r.DocumentID); e != nil {
				t.Fatal(e)
			}
			activate()
			if len(store.notes) != 1 || len(store.reports) != 1 || len(store.watches) != 1 {
				t.Fatal("duplicates")
			}
			views, e := s.Notifications(ctx, 1)
			if e != nil || len(views) != 1 || views[0].Match.Location.Address != "" {
				t.Fatal(views, e)
			}
			other, e := s.Notifications(ctx, 2)
			if e != nil || len(other) != 0 {
				t.Fatal(other, e)
			}
		})
	}
}
func TestRecoveryAuthorizationExpiryAndRevocation(t *testing.T) {
	s, store, clock, proof := recoverySetup(t)
	ctx := context.Background()
	if _, e := s.Search(ctx, 0, recoveryInput()); !errors.Is(e, m.ErrUnauthorized) {
		t.Fatal(e)
	}
	if len(store.docs) != 0 {
		t.Fatal("anonymous query reached store")
	}
	if _, e := s.Activate(ctx, 2, recoveryInput()); !errors.Is(e, m.ErrOwnership) {
		t.Fatal(e)
	}
	if len(store.docs) != 0 {
		t.Fatal("failed ownership did not roll back")
	}
	if _, e := s.Activate(ctx, 1, recoveryInput()); e != nil {
		t.Fatal(e)
	}
	_, e := s.Register(ctx, 2, p.RegisterReport{Identity: recoveryInput(), LocationID: 1, ObservedAt: clock.now, OperationKey: "one"})
	if e != nil {
		t.Fatal(e)
	}
	proof.revoked = true
	if views, e := s.Notifications(ctx, 1); e != nil || len(views) != 0 {
		t.Fatal(views, e)
	}
	if store.watches[0].Status != m.Suspended {
		t.Fatal("watch not suspended")
	}
	clock.now = clock.now.Add(30 * 24 * time.Hour)
	if views, e := s.Search(ctx, 2, recoveryInput()); e != nil || len(views) != 0 {
		t.Fatal(views, e)
	}
}
func TestRecoveryAtomicFailureAndConcurrentMatch(t *testing.T) {
	s, store, clock, _ := recoverySetup(t)
	ctx := context.Background()
	if _, e := s.Activate(ctx, 1, recoveryInput()); e != nil {
		t.Fatal(e)
	}
	store.fail = true
	c := p.RegisterReport{Identity: recoveryInput(), LocationID: 1, ObservedAt: clock.now, OperationKey: "one"}
	if _, e := s.Register(ctx, 2, c); e == nil {
		t.Fatal("expected failure")
	}
	if len(store.reports) != 0 || len(store.notes) != 0 || len(store.ops) != 0 {
		t.Fatal("partial transaction")
	}
	store.fail = false
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	wg.Add(2)
	go func() { defer wg.Done(); _, e := s.Register(ctx, 2, c); errs <- e }()
	go func() { defer wg.Done(); _, e := s.Activate(ctx, 1, recoveryInput()); errs <- e }()
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	if len(store.notes) != 1 {
		t.Fatal("lost or duplicated concurrent match")
	}
}
func TestRecoveryTransitionsAndOperationConflict(t *testing.T) {
	s, _, clock, _ := recoverySetup(t)
	ctx := context.Background()
	c := p.RegisterReport{Identity: recoveryInput(), LocationID: 1, ObservedAt: clock.now, OperationKey: "one"}
	r, e := s.Register(ctx, 2, c)
	if e != nil {
		t.Fatal(e)
	}
	c.ObservedAt = clock.now.Add(-time.Minute)
	if _, e = s.Register(ctx, 2, c); !errors.Is(e, m.ErrConflict) {
		t.Fatal(e)
	}
	if e = s.ChangeReport(ctx, 2, r.ID, m.Confirmed, "at shop"); !errors.Is(e, m.ErrCustodyUnavailable) {
		t.Fatal(e)
	}
	if e = s.ChangeReport(ctx, 2, r.ID, m.Recovered, "mine"); !errors.Is(e, m.ErrOwnership) {
		t.Fatal(e)
	}
	if e = s.ChangeReport(ctx, 1, r.ID, m.Recovered, "collected"); e != nil {
		t.Fatal(e)
	}
}
