package port

import (
	"context"
)

type CharacterInfo struct {
	ID    string
	Coins int
}

type CharacterReader interface {
	GetByUserID(ctx context.Context, userID string) (*CharacterInfo, error)
	GetCoins(ctx context.Context, characterID string) (int, error)
}
