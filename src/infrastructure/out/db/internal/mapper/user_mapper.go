package mapper

import (
	"perdimidocumento/src/domain/models"
	"perdimidocumento/src/infrastructure/out/db/internal/dbo"
)

func UserToDomain(user dbo.UserDBO) models.User {
	return models.User{ID: user.ID, Username: user.Username, PasswordHash: user.PasswordHash}
}
