package valueobjects

import (
	"strings"

	"perdimidocumento/src/domain/recoveryerrors"
)

// IdentityInput lleva el número original solo durante el procesamiento de la solicitud.
// La persistencia y los registros deben usar referencias protegidas o identificadores internos.
type IdentityInput struct{ Country, Type, Number string }

// ProtectedReference identifica un documento mediante un HMAC y las versiones de clave y normalización usadas.
type ProtectedReference struct{ Value, KeyVersion, NormalizationVersion string }

// String y GoString ocultan la identidad en los formatos de texto habituales de fmt.
func (IdentityInput) String() string { return "[redacted identity]" }

func (IdentityInput) GoString() string { return "[redacted identity]" }

// Normalize admite cédulas de ciudadanía colombianas. Conserva el número como texto para preservar los ceros iniciales.
func (i IdentityInput) Normalize() (IdentityInput, error) {
	i.Country = strings.ToUpper(strings.TrimSpace(i.Country))
	i.Type = strings.ToUpper(strings.TrimSpace(i.Type))
	i.Number = strings.TrimSpace(i.Number)
	if i.Country != "CO" || i.Type != "CC" || len(i.Number) == 0 || len(i.Number) > 200 {
		return IdentityInput{}, recoveryerrors.ErrInvalid
	}
	for _, r := range i.Number {
		if r < '0' || r > '9' {
			return IdentityInput{}, recoveryerrors.ErrInvalid
		}
	}
	return i, nil
}

// Valid comprueba que la referencia y sus versiones estén presentes.
func (r ProtectedReference) Valid() bool {
	return r.Value != "" && r.KeyVersion != "" && r.NormalizationVersion != ""
}
