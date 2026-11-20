// Package recoveryerrors define los errores compartidos por el dominio de recuperación.
package recoveryerrors

import "errors"

var (
	ErrInvalid            = errors.New("invalid recovery input")
	ErrUnauthorized       = errors.New("authentication required")
	ErrForbidden          = errors.New("operation not authorized")
	ErrConflict           = errors.New("conflicting operation")
	ErrNotFound           = errors.New("recovery record not found")
	ErrOwnership          = errors.New("verified ownership required")
	ErrCapacity           = errors.New("watch capacity exhausted")
	ErrCustodyUnavailable = errors.New("establishment custody confirmation unavailable")
)
