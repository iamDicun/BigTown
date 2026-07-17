package port

import (
	"context"
	"database/sql"

	characterentity "backend/internal/module/character/entity"
)

// CharacterProvisioner là cross-module dependency nhỏ giống UserReader (xem user_reader.go):
// auth cần tạo 1 characters row mặc định ngay khi user mới được tạo, nhưng không phụ thuộc vào
// toàn bộ backend/internal/module/character/port.
type CharacterProvisioner interface {
	CreateDefaultWithTx(ctx context.Context, tx *sql.Tx, userID string, name string) (*characterentity.Character, error)
}
