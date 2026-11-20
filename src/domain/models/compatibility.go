// Package models conserva los nombres usados por los consumidores del servidor actual.
package models

import (
	"perdimidocumento/src/domain/factories"
	"perdimidocumento/src/domain/models/document"
	"perdimidocumento/src/domain/models/notification"
	"perdimidocumento/src/domain/models/user"
)

type (
	Document        = document.FoundDocument
	Notification    = notification.DocumentNotification
	User            = user.User
	DocumentFactory = factories.DocumentFactory
)

var ErrInvalidDocument = factories.ErrInvalidDocument
