package recovery

import (
	"context"
	m "perdimidocumento/src/domain/models/recovery"
	"time"
)

type Clock interface{ Now() time.Time }

// IdentityProtector must derive keyed, versioned HMAC references, never plain hashes.
// Lookup returns all readable versions; the first is the current write version.
type IdentityProtector interface {
	Lookup(context.Context, m.IdentityInput) ([]m.ProtectedReference, error)
}

// OwnershipVerifier must attest the exact local user/document relationship.
// Authentication, matching names and verified email alone are insufficient.
type OwnershipVerifier interface {
	Verify(context.Context, int64, m.Document) (m.VerifiedOwnership, error)
}
type ActorAuthorizer interface {
	Authenticated(context.Context, int64) (bool, error)
	Moderator(context.Context, int64) (bool, error)
}
type WatchCapacityProvider interface {
	Capacity(context.Context, int64) (int, error)
}
type DocumentRepository interface {
	Document(context.Context, int64) (m.Document, error)
	ResolveDocument(context.Context, string, string, []m.ProtectedReference, bool) (m.Document, error)
}
type LocationRepository interface {
	SaveLocation(context.Context, m.Location) (m.Location, error)
	Location(context.Context, int64) (m.Location, error)
}
type ReportRepository interface {
	SaveReport(context.Context, m.FoundDocumentReport) (m.FoundDocumentReport, error)
	Report(context.Context, int64) (m.FoundDocumentReport, error)
	Reports(context.Context, int64) ([]m.FoundDocumentReport, error)
}
type WatchRepository interface {
	SaveWatch(context.Context, m.WatchedDocument) (m.WatchedDocument, error)
	Watches(context.Context, int64) ([]m.WatchedDocument, error)
	UserWatches(context.Context, int64) ([]m.WatchedDocument, error)
}
type NotificationRepository interface {
	InsertNotification(context.Context, m.Notification) error
	Notifications(context.Context, int64) ([]m.Notification, error)
}
type AuditEntry struct {
	ActorID, ReportID  int64
	Operation, Outcome string
	At                 time.Time
}
type AuditRepository interface {
	Audit(context.Context, AuditEntry) error
}

// Operation holds only protected/internal data, never the original input number.
type Operation struct {
	ActorID, DocumentID, LocationID, ReportID int64
	Key                                       string
	ObservedAt                                time.Time
}
type OperationRepository interface {
	Operation(context.Context, int64, string) (Operation, error)
	SaveOperation(context.Context, Operation) error
}

// Store is a transaction-scoped bundle of the focused repository contracts.
type Store interface {
	DocumentRepository
	LocationRepository
	ReportRepository
	WatchRepository
	NotificationRepository
	AuditRepository
	OperationRepository
}

// UnitOfWork serializes conflicting report/watch operations, including quota reads.
// A callback error must roll back every write. Adapters enforce unique user/document
// watches, protected references, actor/operation keys and user/report/kind notices.
// SaveReport must compare Version on updates to reject stale writers.
type UnitOfWork interface {
	Within(context.Context, func(Store) error) error
}
type RegisterReport struct {
	Identity     m.IdentityInput
	LocationID   int64
	ObservedAt   time.Time
	OperationKey string
}
type ReportRegistrar interface {
	Register(context.Context, int64, RegisterReport) (m.FoundDocumentReport, error)
}
type DocumentSearcher interface {
	Search(context.Context, int64, m.IdentityInput) ([]m.Availability, error)
}
type WatchManager interface {
	Activate(context.Context, int64, m.IdentityInput) (m.WatchedDocument, error)
	Stop(context.Context, int64, int64) error
}
type NotificationReader interface {
	Notifications(context.Context, int64) ([]m.NotificationView, error)
}
