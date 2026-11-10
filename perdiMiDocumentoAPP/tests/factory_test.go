package tests

import (
	"errors"
	"strings"
	"testing"

	"perdimidocumento/src/domain/models"
)

func TestDocumentFactory(t *testing.T) {
	factory := models.DocumentFactory{}
	doc, err := factory.Create(" 001 ", " Cédula ", " Ana ", " Oficina ", 42)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Number != "001" || doc.Type != "Cédula" || doc.OwnerName != "Ana" || doc.Address != "Oficina" || doc.ReportedBy != 42 {
		t.Fatalf("documento inesperado: %+v", doc)
	}
	for _, values := range [][4]string{
		{"", "id", "Ana", "Oficina"},
		{"1", " ", "Ana", "Oficina"},
		{"1", "id", "", "Oficina"},
		{"1", "id", "Ana", ""},
		{strings.Repeat("x", 201), "id", "Ana", "Oficina"},
	} {
		_, err := factory.Create(values[0], values[1], values[2], values[3], 42)
		if !errors.Is(err, models.ErrInvalidDocument) {
			t.Fatalf("validación de %v: %v", values, err)
		}
	}
}
