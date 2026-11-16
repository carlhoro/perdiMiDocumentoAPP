package models

import (
	"errors"
	"strings"
	"unicode/utf8"
)

var ErrInvalidDocument = errors.New("documento inválido: número, tipo, nombre y dirección son obligatorios (máximo 200 caracteres cada uno)")

// DocumentFactory is the factory method for found documents.
type DocumentFactory struct{}

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
