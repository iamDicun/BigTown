package port

import (
	"context"
	"database/sql"

	userentity "backend/internal/module/user/entity"
)

// UserReader là ví dụ cụ thể của quy tắc "cross-module dependency": auth cần đọc/ghi user nhưng
// KHÔNG import backend/internal/module/user/port. Auth tự định nghĩa interface nhỏ theo đúng nhu cầu
// của chính nó, rồi auth/provider.go bind implementation thật (user/repository.UserRepository) vào
// interface này. Xem ARCHITECTURE_GUIDE.md mục 5.
type UserReader interface {
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	CreateWithTx(ctx context.Context, tx *sql.Tx, fullName string, email string) (*userentity.User, error)
	FindByEmail(ctx context.Context, email string) (*userentity.User, error)
	FindByID(ctx context.Context, id string) (*userentity.User, error)
}
