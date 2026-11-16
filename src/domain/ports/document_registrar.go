package ports

import (
	"context"

	"perdimidocumento/src/domain/models"
)

type DocumentRegistrar interface {
	Register(context.Context, int64, string, string, string, string) (models.Document, error)
}
