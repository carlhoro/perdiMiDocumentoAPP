package dbo

type NotificationDBO struct {
	ID        int64
	CreatedAt int64
	Document  DocumentDBO
}
