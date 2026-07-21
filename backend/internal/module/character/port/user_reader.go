package port

import (
	"context"

	userentity "backend/internal/module/user/entity"
)

// UserReader là cross-module dependency nhỏ giống CharacterResolver/MapReader ở realtime module:
// character cần full_name thật của user để đặt tên character khi tạo qua đường an toàn dự phòng
// (GetOrCreateForUser gặp user chưa có character — thường là user đã tồn tại từ trước khi character
// tự động được tạo lúc Register/TeamsLogin), thay vì dùng tên mặc định cứng (vd "Player"). Bind bằng
// *user/repository.UserRepository ở app wiring (xem internal/app/app.go).
type UserReader interface {
	FindByID(ctx context.Context, id string) (*userentity.User, error)
}
