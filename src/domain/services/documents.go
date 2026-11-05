package services

import (
	"context"
	"fmt"
	"strings"

	"perdimidocumento/src/domain/models"
	"perdimidocumento/src/domain/ports"
)

// DocumentProxy validates document creation and audits rejected attempts.
type DocumentProxy struct {
	Repository ports.Repository
	Factory    models.DocumentFactory
}

func (p DocumentProxy) Register(ctx context.Context, userID int64, number, documentType, ownerName, address string) (models.Document, error) {
	doc, err := p.Factory.Create(number, documentType, ownerName, address, userID)
	if err != nil {
		// Limit untrusted input stored in the audit log.
		logNumber := []rune(strings.TrimSpace(number))
		if len(logNumber) > 200 {
			logNumber = logNumber[:200]
		}
		if logErr := p.Repository.LogRejectedDocument(ctx, userID, string(logNumber)); logErr != nil {
			return models.Document{}, fmt.Errorf("no se pudo auditar el rechazo: %w", logErr)
		}
		return models.Document{}, err
	}
	return p.Repository.RegisterDocument(ctx, doc)
}
