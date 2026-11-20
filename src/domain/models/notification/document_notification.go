package notification

import (
	"time"

	"perdimidocumento/src/domain/models/document"
)

// DocumentNotification representa un aviso generado por una coincidencia documental.
type DocumentNotification struct {
	ID        int64
	Document  document.FoundDocument
	CreatedAt time.Time
}
