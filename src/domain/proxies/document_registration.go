package proxies

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
	Repository ports.DocumentRegistrationRepository
	Factory    factories.DocumentFactory
}

const maxAuditNumberLength = 200

func (p DocumentProxy) Register(ctx context.Context, userID int64, number, documentType, ownerName, address string) (document.FoundDocument, error) {
	doc, err := p.Factory.Create(number, documentType, ownerName, address, userID)
	if err == nil {
		return p.Repository.RegisterDocument(ctx, doc)
	}

	if auditErr := p.Repository.LogRejectedDocument(ctx, userID, rejectedDocumentNumber(number)); auditErr != nil {
		return document.FoundDocument{}, fmt.Errorf("no se pudo auditar el rechazo: %w", auditErr)
	}
	return document.FoundDocument{}, err
}

func rejectedDocumentNumber(number string) string {
	runes := []rune(strings.TrimSpace(number))
	if len(runes) > maxAuditNumberLength {
		runes = runes[:maxAuditNumberLength]
	}
	return string(runes)
}
