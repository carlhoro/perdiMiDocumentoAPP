package services

import (
	"context"

	"perdimidocumento/src/domain/ports"
)

// SubscriptionService registra en el repositorio el interés del usuario por un número documental.
type SubscriptionService struct {
	Repository ports.SubscriptionRepository
}

func (s SubscriptionService) Subscribe(ctx context.Context, userID int64, number string) error {
	return s.Repository.Subscribe(ctx, userID, number)
}
