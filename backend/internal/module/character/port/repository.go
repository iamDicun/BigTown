package port

import (
	"context"
	"database/sql"

	"backend/internal/module/character/entity"
)

type CharacterRepository interface {
	FindByUserID(ctx context.Context, userID string) (*entity.Character, error)
	CreateWithTx(ctx context.Context, tx *sql.Tx, userID string, name string, baseAssetKey string) (*entity.Character, error)

	// FindMapByCode đọc metadata map theo `code` (bảng `maps`). Dùng cho cả provisioning
	// (resolve map_id lúc tạo character) lẫn realtime bootstrap (trả asset key/spawn point thật).
	FindMapByCode(ctx context.Context, code string) (*entity.MapInfo, error)

	// SyncMapID đồng bộ map_id của 1 character theo map mặc định hiện hành (GAME_DEFAULT_MAP_CODE).
	// Nếu currentMapID đã khớp thì không ghi DB, trả lại nguyên currentMapID. Nếu map mặc định
	// chưa được seed, trả lại nguyên currentMapID (không chặn login vì thiếu seed data).
	SyncMapID(ctx context.Context, characterID string, currentMapID *string) (*string, error)
}
