package location

import (
	"strings"
	"unicode/utf8"

	"perdimidocumento/src/domain/models/valueobjects"
	"perdimidocumento/src/domain/recoveryerrors"
)

// Location describe el lugar donde se reportó un documento.
type Location struct {
	ID                                                     int64
	Name, Address, Municipality, Department, Country, Zone string
	Visibility                                             valueobjects.Visibility
}

func (l Location) Validate() error {
	for _, s := range []string{l.Name, l.Address, l.Municipality, l.Department} {
		if strings.TrimSpace(s) == "" || utf8.RuneCountInString(s) > 200 {
			return recoveryerrors.ErrInvalid
		}
	}
	if l.Country != "CO" || len(l.Zone) > 200 || (l.Visibility != "" && l.Visibility != valueobjects.Restricted && l.Visibility != valueobjects.Public) {
		return recoveryerrors.ErrInvalid
	}
	return nil
}

// Summary incluye nombre y dirección exacta únicamente para ubicaciones marcadas como públicas.
func (l Location) Summary() valueobjects.LocationSummary {
	s := valueobjects.LocationSummary{Municipality: l.Municipality, Department: l.Department, Country: l.Country, Zone: l.Zone}
	if l.Visibility == valueobjects.Public {
		s.Name = l.Name
		s.Address = l.Address
	}
	return s
}
