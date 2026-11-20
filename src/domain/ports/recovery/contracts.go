// Package recovery define los contratos entre los casos de uso de recuperación y sus adaptadores.
package recovery

import (
	"context"
	"time"

	m "perdimidocumento/src/domain/models/recovery"
)

// Clock permite usar una hora controlada al probar vencimientos y reintentos.
type Clock interface{ Now() time.Time }

// IdentityProtector deriva referencias HMAC versionadas a partir de la identidad normalizada.
// Lookup devuelve primero la versión de escritura vigente y luego las versiones anteriores admitidas.
type IdentityProtector interface {
	Lookup(context.Context, m.IdentityInput) ([]m.ProtectedReference, error)
}

// OwnershipVerifier acredita la titularidad entre una cuenta local y un documento concreto.
// La evidencia debe vincular ambos identificadores. Una sesión activa, el nombre o el correo verificado son insuficientes.
type OwnershipVerifier interface {
	Verify(context.Context, int64, m.Document) (m.VerifiedOwnership, error)
}

// ActorAuthorizer comprueba la vigencia del usuario local y su permiso de moderación.
type ActorAuthorizer interface {
	Authenticated(context.Context, int64) (bool, error)
	Moderator(context.Context, int64) (bool, error)
}

// WatchCapacityProvider consulta el cupo del usuario al activar un seguimiento.
type WatchCapacityProvider interface {
	Capacity(context.Context, int64) (int, error)
}

// DocumentRepository resuelve identidades protegidas y consulta documentos por ID interno.
type DocumentRepository interface {
	Document(context.Context, int64) (m.Document, error)
	// ResolveDocument reúne las referencias de un mismo país y tipo en un documento canónico.
	// El último argumento indica si puede crearlo cuando ninguna referencia coincide.
	ResolveDocument(context.Context, string, string, []m.ProtectedReference, bool) (m.Document, error)
}

// LocationRepository persiste las ubicaciones vinculadas a los hallazgos.
type LocationRepository interface {
	SaveLocation(context.Context, m.Location) (m.Location, error)
	Location(context.Context, int64) (m.Location, error)
}

// ReportRepository persiste reportes y consulta el historial de hallazgos por documento.
type ReportRepository interface {
	// SaveReport debe comprobar Version al actualizar para rechazar cambios hechos sobre una versión anterior.
	SaveReport(context.Context, m.FoundDocumentReport) (m.FoundDocumentReport, error)
	Report(context.Context, int64) (m.FoundDocumentReport, error)
	// Reports devuelve los hallazgos registrados para un documento, incluidos los históricos.
	Reports(context.Context, int64) ([]m.FoundDocumentReport, error)
}

// WatchRepository persiste seguimientos consultables por documento o usuario.
type WatchRepository interface {
	SaveWatch(context.Context, m.WatchedDocument) (m.WatchedDocument, error)
	// Watches consulta los seguimientos por documento. UserWatches los consulta por usuario.
	Watches(context.Context, int64) ([]m.WatchedDocument, error)
	UserWatches(context.Context, int64) ([]m.WatchedDocument, error)
}

// NotificationRepository guarda las coincidencias y consulta los avisos por usuario.
type NotificationRepository interface {
	// InsertNotification trata como éxito la repetición del mismo usuario, reporte y clase de aviso.
	InsertNotification(context.Context, m.Notification) error
	Notifications(context.Context, int64) ([]m.Notification, error)
}

// AuditEntry registra quién ejecutó una operación, sobre qué reporte y con qué resultado.
type AuditEntry struct {
	ActorID, ReportID  int64
	Operation, Outcome string
	At                 time.Time
}

// AuditRepository persiste los resultados de operaciones auditadas.
type AuditRepository interface {
	Audit(context.Context, AuditEntry) error
}

// Operation vincula una clave de reintento con el resultado original mediante IDs internos.
// La clave debe ser opaca y mantenerse libre del número de documento original.
type Operation struct {
	ActorID, DocumentID, LocationID, ReportID int64
	Key                                       string
	ObservedAt                                time.Time
}

// OperationRepository conserva el resultado original de cada registro para resolver reintentos.
type OperationRepository interface {
	Operation(context.Context, int64, string) (Operation, error)
	SaveOperation(context.Context, Operation) error
}

// Store reúne los repositorios disponibles dentro de una misma transacción.
type Store interface {
	DocumentRepository
	LocationRepository
	ReportRepository
	WatchRepository
	NotificationRepository
	AuditRepository
	OperationRepository
}

// UnitOfWork serializa las operaciones de reportes y seguimientos que compiten entre sí,
// incluidas las lecturas de cupo. Si la función devuelve un error, revierte todas sus escrituras.
// El adaptador garantiza la unicidad de las referencias protegidas, los seguimientos por
// usuario y documento, las claves de operación por actor y los avisos por usuario, reporte y clase.
type UnitOfWork interface {
	Within(context.Context, func(Store) error) error
}

// RegisterReport contiene los datos de un avistamiento y la clave para reintentar su registro.
type RegisterReport struct {
	Identity     m.IdentityInput
	LocationID   int64
	ObservedAt   time.Time
	OperationKey string
}

// ReportRegistrar registra un hallazgo en nombre del usuario autenticado.
type ReportRegistrar interface {
	Register(context.Context, int64, RegisterReport) (m.FoundDocumentReport, error)
}

// DocumentSearcher consulta la disponibilidad de un documento para un usuario autenticado.
type DocumentSearcher interface {
	Search(context.Context, int64, m.IdentityInput) ([]m.Availability, error)
}

// WatchManager activa seguimientos por identidad documental y los detiene por ID de documento.
type WatchManager interface {
	Activate(context.Context, int64, m.IdentityInput) (m.WatchedDocument, error)
	Stop(context.Context, int64, int64) error
}

// NotificationReader devuelve los avisos que el usuario puede consultar en ese momento.
type NotificationReader interface {
	Notifications(context.Context, int64) ([]m.NotificationView, error)
}
