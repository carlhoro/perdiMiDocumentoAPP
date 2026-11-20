package document

import "perdimidocumento/src/domain/models/valueobjects"

// DocumentIdentity representa una identidad documental que puede tener varios hallazgos y referencias protegidas.
type DocumentIdentity struct {
	ID            int64
	Country, Type string
	References    []valueobjects.ProtectedReference
}
