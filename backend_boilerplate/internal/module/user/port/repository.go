package port

import (
	"context"
	"database/sql"

	"backend/internal/module/user/entity"
)

type UserRepository interface {
	FindAll(ctx context.Context) ([]entity.User, error)

	// Các method dưới đây tồn tại ở đây (không phải chỉ ở auth/port/user_reader.go) vì UserRepository
	// là implementation thật duy nhất — auth module tự định nghĩa interface UserReader nhỏ hơn theo
	// nhu cầu của chính nó rồi bind implementation này vào, xem auth/provider.go.
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	CreateWithTx(ctx context.Context, tx *sql.Tx, fullName string, email string) (*entity.User, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	FindByID(ctx context.Context, id string) (*entity.User, error)
}
