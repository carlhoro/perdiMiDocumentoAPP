// Package recovery coordina los casos de uso y accede al almacenamiento y otros servicios mediante puertos.
package recovery

import (
	"context"
	"errors"
	m "perdimidocumento/src/domain/models/recovery"
	p "perdimidocumento/src/domain/ports/recovery"
	"sort"
	"strings"
	"time"
)

// Service coordina las reglas de recuperación con las dependencias provistas por sus adaptadores.
type Service struct {
	Transactions               p.UnitOfWork
	Identity                   p.IdentityProtector
	Ownership                  p.OwnershipVerifier
	Actors                     p.ActorAuthorizer
	Capacity                   p.WatchCapacityProvider
	Clock                      p.Clock
	ReportTTL, DuplicateWindow time.Duration
}

// El compilador comprueba que Service implemente los puertos de entrada.
var _ p.ReportRegistrar = Service{}
var _ p.DocumentSearcher = Service{}
var _ p.WatchManager = Service{}
var _ p.NotificationReader = Service{}

func (s Service) ttl() time.Duration {
	if s.ReportTTL > 0 {
		return s.ReportTTL
	}
	return 30 * 24 * time.Hour
}
func (s Service) window() time.Duration {
	if s.DuplicateWindow > 0 {
		return s.DuplicateWindow
	}
	return 24 * time.Hour
}
func (s Service) authenticate(ctx context.Context, user int64) error {
	if user <= 0 || s.Actors == nil {
		return m.ErrUnauthorized
	}
	ok, err := s.Actors.Authenticated(ctx, user)
	if err != nil {
		return err
	}
	if !ok {
		return m.ErrUnauthorized
	}
	return nil
}

// refs normaliza la identidad antes de pedir al protector las referencias que puede consultar el repositorio.
func (s Service) refs(ctx context.Context, input m.IdentityInput) (m.IdentityInput, []m.ProtectedReference, error) {
	i, err := input.Normalize()
	if err != nil {
		return i, nil, err
	}
	if s.Identity == nil {
		return i, nil, m.ErrForbidden
	}
	refs, err := s.Identity.Lookup(ctx, i)
	if err != nil {
		return i, nil, err
	}
	if len(refs) == 0 {
		return i, nil, m.ErrInvalid
	}
	for _, ref := range refs {
		if !ref.Valid() {
			return i, nil, m.ErrInvalid
		}
	}
	return i, refs, nil
}

// proof comprueba que la evidencia devuelta por el verificador corresponda al usuario, al documento y al momento de la operación.
func (s Service) proof(ctx context.Context, user int64, doc m.Document, now time.Time) (m.VerifiedOwnership, error) {
	if s.Ownership == nil {
		return m.VerifiedOwnership{}, m.ErrOwnership
	}
	v, err := s.Ownership.Verify(ctx, user, doc)
	if err != nil {
		return v, err
	}
	if !v.Valid(user, doc.ID, now) {
		return v, m.ErrOwnership
	}
	return v, nil
}

// SaveLocation crea una ubicación para un usuario autenticado y aplica visibilidad restringida por defecto.
func (s Service) SaveLocation(ctx context.Context, user int64, l m.Location) (out m.Location, err error) {
	if err = s.authenticate(ctx, user); err != nil {
		return
	}
	if l.ID != 0 {
		return out, m.ErrInvalid
	}
	if err = l.Validate(); err != nil {
		return
	}
	if l.Visibility == "" {
		l.Visibility = m.Restricted
	}
	err = s.Transactions.Within(ctx, func(tx p.Store) error { var e error; out, e = tx.SaveLocation(ctx, l); return e })
	return
}

// Register registra un hallazgo, sus avisos y la auditoría de éxito en una misma transacción.
func (s Service) Register(ctx context.Context, user int64, c p.RegisterReport) (out m.FoundDocumentReport, err error) {
	if err = s.authenticate(ctx, user); err != nil {
		return
	}
	now := s.Clock.Now().UTC()
	if strings.TrimSpace(c.OperationKey) == "" || len(c.OperationKey) > 200 || c.ObservedAt.IsZero() || c.ObservedAt.After(now) || c.LocationID <= 0 {
		return out, s.auditRejection(ctx, user, m.ErrInvalid)
	}
	i, refs, err := s.refs(ctx, c.Identity)
	if err != nil {
		return out, s.auditRejection(ctx, user, err)
	}
	err = s.Transactions.Within(ctx, func(tx p.Store) error {
		if _, e := tx.Location(ctx, c.LocationID); e != nil {
			return e
		}
		doc, e := tx.ResolveDocument(ctx, i.Country, i.Type, refs, true)
		if e != nil {
			return e
		}
		// Un reintento con la misma clave devuelve el reporte original si los datos coinciden.
		op, e := tx.Operation(ctx, user, c.OperationKey)
		if e == nil {
			if op.DocumentID != doc.ID || op.LocationID != c.LocationID || !op.ObservedAt.Equal(c.ObservedAt) {
				return m.ErrConflict
			}
			out, e = tx.Report(ctx, op.ReportID)
			return e
		}
		if !errors.Is(e, m.ErrNotFound) {
			return e
		}
		reports, e := tx.Reports(ctx, doc.ID)
		if e != nil {
			return e
		}
		// Con una clave nueva, la deduplicación usa el reportante, la ubicación, el estado y la fecha de registro.
		for _, r := range reports {
			if r.ReportedBy == user && r.LocationID == c.LocationID && r.Status == m.Reported && !r.CreatedAt.Before(now.Add(-s.window())) {
				out = r
				break
			}
		}
		if out.ID == 0 {
			out, e = m.NewReport(doc.ID, c.LocationID, user, c.ObservedAt, now)
			if e != nil {
				return e
			}
			// Los avistamientos fuera de plazo se conservan como reportes históricos.
			if !out.Available(now, s.ttl()) {
				out.Status = m.Stale
			}
			out, e = tx.SaveReport(ctx, out)
			if e != nil {
				return e
			}
			if e = s.reportObserved(ctx, tx, doc, out, now); e != nil {
				return e
			}
		}
		if e = tx.SaveOperation(ctx, p.Operation{ActorID: user, DocumentID: doc.ID, LocationID: c.LocationID, ReportID: out.ID, Key: c.OperationKey, ObservedAt: c.ObservedAt.UTC()}); e != nil {
			return e
		}
		return tx.Audit(ctx, p.AuditEntry{ActorID: user, ReportID: out.ID, Operation: "register_report", Outcome: "accepted", At: now})
	})
	return
}

// auditRejection guarda el resultado del rechazo en su propia transacción y conserva el error original.
func (s Service) auditRejection(ctx context.Context, user int64, cause error) error {
	err := s.Transactions.Within(ctx, func(tx p.Store) error {
		return tx.Audit(ctx, p.AuditEntry{ActorID: user, Operation: "register_report", Outcome: "invalid", At: s.Clock.Now().UTC()})
	})
	if err != nil {
		return errors.Join(cause, err)
	}
	return cause
}

// Search muestra la disponibilidad del hallazgo vigente más reciente de cada ubicación a un usuario autenticado.
func (s Service) Search(ctx context.Context, user int64, input m.IdentityInput) (out []m.Availability, err error) {
	out = []m.Availability{}
	if err = s.authenticate(ctx, user); err != nil {
		return
	}
	i, refs, err := s.refs(ctx, input)
	if err != nil {
		return out, err
	}
	now := s.Clock.Now().UTC()
	err = s.Transactions.Within(ctx, func(tx p.Store) error {
		// Las búsquedas solo consultan documentos existentes.
		doc, e := tx.ResolveDocument(ctx, i.Country, i.Type, refs, false)
		if errors.Is(e, m.ErrNotFound) {
			return nil
		}
		if e != nil {
			return e
		}
		reports, e := tx.Reports(ctx, doc.ID)
		if e != nil {
			return e
		}
		sort.SliceStable(reports, func(i, j int) bool { return reports[i].ObservedAt.After(reports[j].ObservedAt) })
		seen := map[int64]bool{}
		for _, r := range reports {
			if !r.Available(now, s.ttl()) || seen[r.LocationID] {
				continue
			}
			l, e := tx.Location(ctx, r.LocationID)
			if e != nil {
				return e
			}
			seen[r.LocationID] = true
			out = append(out, m.Availability{Location: l.Summary(), Status: r.Status, ExpiresAt: r.ObservedAt.Add(s.ttl())})
		}
		return nil
	})
	return
}

// Activate acredita la titularidad, comprueba el cupo y busca coincidencias con reportes ya registrados.
func (s Service) Activate(ctx context.Context, user int64, input m.IdentityInput) (out m.WatchedDocument, err error) {
	if err = s.authenticate(ctx, user); err != nil {
		return
	}
	i, refs, err := s.refs(ctx, input)
	if err != nil {
		return out, err
	}
	now := s.Clock.Now().UTC()
	err = s.Transactions.Within(ctx, func(tx p.Store) error {
		doc, e := tx.ResolveDocument(ctx, i.Country, i.Type, refs, true)
		if e != nil {
			return e
		}
		proof, e := s.proof(ctx, user, doc, now)
		if e != nil {
			return e
		}
		watches, e := tx.UserWatches(ctx, user)
		if e != nil {
			return e
		}
		count := 0
		for _, w := range watches {
			if w.DocumentID == doc.ID {
				out = w
			}
			if w.Status == m.Active {
				count++
			}
			// Un usuario puede tener activa una sola cédula distinta, aunque su cupo sea mayor.
			if w.Status == m.Active && w.DocumentID != doc.ID {
				return m.ErrConflict
			}
		}
		// Volver a activar un seguimiento ya activo conserva el cupo que estaba usando.
		if out.Status != m.Active {
			capacity := 1
			if s.Capacity != nil {
				capacity, e = s.Capacity.Capacity(ctx, user)
				if e != nil {
					return e
				}
			}
			if count >= capacity {
				return m.ErrCapacity
			}
		}
		out.UserID = user
		out.DocumentID = doc.ID
		out.Status = m.Active
		out.Ownership = proof
		out, e = tx.SaveWatch(ctx, out)
		if e != nil {
			return e
		}
		reports, e := tx.Reports(ctx, doc.ID)
		if e != nil {
			return e
		}
		for _, r := range reports {
			if e = s.notify(ctx, tx, out, r, now); e != nil {
				return e
			}
		}
		return tx.Audit(ctx, p.AuditEntry{ActorID: user, Operation: "activate_watch", Outcome: "accepted", At: now})
	})
	return
}

// Stop desactiva el seguimiento del usuario para el documento indicado y conserva el registro del seguimiento.
func (s Service) Stop(ctx context.Context, user, documentID int64) error {
	if err := s.authenticate(ctx, user); err != nil {
		return err
	}
	return s.Transactions.Within(ctx, func(tx p.Store) error {
		watches, e := tx.UserWatches(ctx, user)
		if e != nil {
			return e
		}
		for _, w := range watches {
			if w.DocumentID == documentID {
				w.Status = m.Stopped
				if _, e = tx.SaveWatch(ctx, w); e != nil {
					return e
				}
				return tx.Audit(ctx, p.AuditEntry{ActorID: user, Operation: "stop_watch", Outcome: "accepted", At: s.Clock.Now().UTC()})
			}
		}
		return m.ErrNotFound
	})
}

// reportObserved genera avisos para seguimientos previos a un nuevo hallazgo.
// Vuelve a verificar la titularidad dentro de la transacción y suspende el seguimiento si perdió vigencia.
func (s Service) reportObserved(ctx context.Context, tx p.Store, doc m.Document, r m.FoundDocumentReport, now time.Time) error {
	if !r.Available(now, s.ttl()) {
		return nil
	}
	watches, err := tx.Watches(ctx, doc.ID)
	if err != nil {
		return err
	}
	for _, w := range watches {
		if w.Status != m.Active {
			continue
		}
		v, e := s.proof(ctx, w.UserID, doc, now)
		if errors.Is(e, m.ErrOwnership) {
			w.Status = m.Suspended
			if _, e = tx.SaveWatch(ctx, w); e != nil {
				return e
			}
			continue
		}
		if e != nil {
			return e
		}
		w.Ownership = v
		if e = s.notify(ctx, tx, w, r, now); e != nil {
			return e
		}
	}
	return nil
}
func (s Service) notify(ctx context.Context, tx p.Store, w m.WatchedDocument, r m.FoundDocumentReport, now time.Time) error {
	if w.Status != m.Active || w.DocumentID != r.DocumentID || !w.Ownership.Valid(w.UserID, r.DocumentID, now) || !r.Available(now, s.ttl()) {
		return nil
	}
	return tx.InsertNotification(ctx, m.Notification{UserID: w.UserID, WatchID: w.ID, ReportID: r.ID, Kind: "match", CreatedAt: now})
}

// Notifications arma la vista de los avisos con la titularidad, la visibilidad y el estado vigentes al consultar.
func (s Service) Notifications(ctx context.Context, user int64) (out []m.NotificationView, err error) {
	out = []m.NotificationView{}
	if err = s.authenticate(ctx, user); err != nil {
		return
	}
	now := s.Clock.Now().UTC()
	err = s.Transactions.Within(ctx, func(tx p.Store) error {
		notes, e := tx.Notifications(ctx, user)
		if e != nil {
			return e
		}
		watches, e := tx.UserWatches(ctx, user)
		if e != nil {
			return e
		}
		for _, n := range notes {
			if n.UserID != user {
				return m.ErrForbidden
			}
			var watch m.WatchedDocument
			for _, w := range watches {
				if w.ID == n.WatchID {
					watch = w
					break
				}
			}
			if watch.Status != m.Active {
				continue
			}
			r, e := tx.Report(ctx, n.ReportID)
			if e != nil {
				return e
			}
			if r.DocumentID != watch.DocumentID {
				return m.ErrConflict
			}
			doc, e := tx.Document(ctx, watch.DocumentID)
			if e != nil {
				return e
			}
			if _, e = s.proof(ctx, user, doc, now); e != nil {
				if errors.Is(e, m.ErrOwnership) {
					watch.Status = m.Suspended
					if _, e = tx.SaveWatch(ctx, watch); e != nil {
						return e
					}
					continue
				}
				return e
			}
			l, e := tx.Location(ctx, r.LocationID)
			if e != nil {
				return e
			}
			// El vencimiento se refleja en la respuesta. El estado guardado del reporte se conserva.
			status := r.Status
			if (status == m.Reported || status == m.Confirmed) && !r.Available(now, s.ttl()) {
				status = m.Stale
			}
			out = append(out, m.NotificationView{ID: n.ID, Match: m.Availability{Location: l.Summary(), Status: status, ExpiresAt: r.ObservedAt.Add(s.ttl())}, CreatedAt: n.CreatedAt})
		}
		return nil
	})
	return
}

// ChangeReport reúne los permisos que exige el estado solicitado y delega la transición al reporte.
func (s Service) ChangeReport(ctx context.Context, user, id int64, to m.ReportStatus, reason string) error {
	if err := s.authenticate(ctx, user); err != nil {
		return err
	}
	now := s.Clock.Now().UTC()
	return s.Transactions.Within(ctx, func(tx p.Store) error {
		r, e := tx.Report(ctx, id)
		if e != nil {
			return e
		}
		moderator := false
		owner := false
		if to == m.Rejected || (to == m.Stale && user != r.ReportedBy) {
			moderator, e = s.Actors.Moderator(ctx, user)
			if e != nil {
				return e
			}
		}
		if to == m.Recovered {
			doc, err := tx.Document(ctx, r.DocumentID)
			if err != nil {
				return err
			}
			_, e = s.proof(ctx, user, doc, now)
			if e != nil {
				return e
			}
			owner = true
		}
		if e = r.Change(to, user, moderator, owner, reason, now); e != nil {
			return e
		}
		if _, e = tx.SaveReport(ctx, r); e != nil {
			return e
		}
		return tx.Audit(ctx, p.AuditEntry{ActorID: user, ReportID: id, Operation: "change_report", Outcome: string(to), At: now})
	})
}
