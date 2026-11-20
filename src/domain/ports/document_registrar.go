package ports

import (
	"context"

	"perdimidocumento/src/domain/models"
)

// DocumentRegistrar recibe los datos de un hallazgo y lo registra en nombre del usuario.
type DocumentRegistrar interface {
	Register(context.Context, int64, string, string, string, string) (models.Document, error)
}
