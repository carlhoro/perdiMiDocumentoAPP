package mapper

import (
	"time"

	"perdimidocumento/src/domain/models"
	"perdimidocumento/src/infrastructure/out/db/internal/dbo"
)

func DocumentToDBO(doc models.Document) dbo.DocumentDBO {
	return dbo.DocumentDBO{
		ID: doc.ID, Number: doc.Number, Type: doc.Type, OwnerName: doc.OwnerName,
		Address: doc.Address, ReportedBy: doc.ReportedBy, CreatedAt: doc.CreatedAt.Unix(),
	}
}

func DocumentToDomain(doc dbo.DocumentDBO) models.Document {
	return models.Document{
		ID: doc.ID, Number: doc.Number, Type: doc.Type, OwnerName: doc.OwnerName,
		Address: doc.Address, ReportedBy: doc.ReportedBy, CreatedAt: time.Unix(doc.CreatedAt, 0).UTC(),
	}
}
