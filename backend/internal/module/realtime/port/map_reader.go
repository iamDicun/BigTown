package port

import (
	"context"

	characterentity "backend/internal/module/character/entity"
)

// MapReader là cross-module dependency nhỏ giống CharacterReader của module chat: realtime
// bootstrap cần metadata map mặc định hiện hành (tilemap/tileset/spawn point) để trả cho frontend
// mà không hardcode, nhưng không phụ thuộc vào character/port hay character/repository.
// Bind bằng *character/usecase.CharacterUsecase.GetDefaultMap ở realtime/provider.go.
type MapReader interface {
	GetDefaultMap(ctx context.Context) (*characterentity.MapInfo, error)
}
