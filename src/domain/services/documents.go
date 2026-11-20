package services

import (
	"context"
	"fmt"
	"strings"

	"perdimidocumento/src/domain/factories"
	"perdimidocumento/src/domain/models/document"
	"perdimidocumento/src/domain/ports"
)

// DocumentProxy valida los hallazgos antes de guardarlos y audita los intentos rechazados.
type DocumentProxy struct {
	Repository ports.Repository
	Factory    factories.DocumentFactory
}

func (p DocumentProxy) Register(ctx context.Context, userID int64, number, documentType, ownerName, address string) (document.FoundDocument, error) {
	doc, err := p.Factory.Create(number, documentType, ownerName, address, userID)
	if err != nil {
		// Limit untrusted input stored in the audit log.
		logNumber := []rune(strings.TrimSpace(number))
		if len(logNumber) > 200 {
			logNumber = logNumber[:200]
		}
		if logErr := p.Repository.LogRejectedDocument(ctx, userID, string(logNumber)); logErr != nil {
			return document.FoundDocument{}, fmt.Errorf("no se pudo auditar el rechazo: %w", logErr)
		}
		return document.FoundDocument{}, err
	}
	return p.Repository.RegisterDocument(ctx, doc)
}
