package port

import (
	"context"

	characterentity "backend/internal/module/character/entity"
)

// CharacterReader là cross-module dependency nhỏ giống UserReader của module auth: chat cần
// character_id + display name của user gửi tin, nhưng không phụ thuộc vào character/port.
// Bind bằng *character/usecase.CharacterUsecase (get-or-create) ở chat/module.go.
type CharacterReader interface {
	GetOrCreateForUser(ctx context.Context, userID string, defaultName string) (*characterentity.Character, error)
}
