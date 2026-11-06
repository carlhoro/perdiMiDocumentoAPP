package models

import "time"

type Notification struct {
	ID        int64
	Document  Document
	CreatedAt time.Time
}
