package models

import "time"

// Notification representa un aviso generado por una coincidencia documental.
type Notification struct {
	ID        int64
	Document  Document
	CreatedAt time.Time
}
