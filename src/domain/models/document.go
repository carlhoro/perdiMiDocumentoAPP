package models

import "time"

type Document struct {
	ID         int64
	Number     string
	Type       string
	OwnerName  string
	Address    string
	ReportedBy int64
	CreatedAt  time.Time
}
