package models

import (
	"errors"
	"strings"
	"unicode/utf8"
)

// DocumentFactory construye documentos encontrados después de validar sus datos.
type DocumentFactory struct{}

// ErrInvalidDocument indica que uno o más campos del documento incumplen la validación.
var ErrInvalidDocument = errors.New("documento inválido: número, tipo, nombre y dirección son obligatorios (máximo 200 caracteres cada uno)")

// Create normaliza los campos de texto y valida la información antes de crear el documento.
func (DocumentFactory) Create(number, documentType, ownerName, address string, userID int64) (Document, error) {
	values := []*string{&number, &documentType, &ownerName, &address}
	for _, value := range values {
		*value = strings.TrimSpace(*value)
		if *value == "" || utf8.RuneCountInString(*value) > 200 {
			return Document{}, ErrInvalidDocument
		}
	}
	return Document{Number: number, Type: documentType, OwnerName: ownerName, Address: address, ReportedBy: userID}, nil
}
