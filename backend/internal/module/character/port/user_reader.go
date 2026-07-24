package port

import (
	"context"

	userentity "backend/internal/module/user/entity"
)

// UserReader là cross-module dependency nhỏ giống CharacterResolver/MapReader ở realtime module.
// Giữ interface này để character module có thể đọc hồ sơ user nếu cần validate/enrich character
// creation mà không phụ thuộc trực tiếp vào user/repository.
type UserReader interface {
	FindByID(ctx context.Context, id string) (*userentity.User, error)
}
