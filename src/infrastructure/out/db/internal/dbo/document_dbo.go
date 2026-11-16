package dbo

type DocumentDBO struct {
	ID         int64
	Number     string
	Type       string
	OwnerName  string
	Address    string
	ReportedBy int64
	CreatedAt  int64
}
