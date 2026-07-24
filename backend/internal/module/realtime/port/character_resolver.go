package port

import (
	"context"

	characterentity "backend/internal/module/character/entity"
)

// CharacterResolver là cross-module dependency nhỏ giống CharacterReader của module chat: realtime
// cần character_id thật của user đang giữ kết nối để join room/validate movement, nhưng không phụ
// thuộc vào character/port hay character/repository. Bind bằng *character/usecase.CharacterUsecase.
type CharacterResolver interface {
	GetByUserID(ctx context.Context, userID string) (*characterentity.Character, error)
}
