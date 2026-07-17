package usecase

import (
	"context"
	"strings"

	"backend/internal/apperror"
	"backend/internal/security"
)

type RegisterInput struct {
	FullName string
	Email    string
	Password string
}

type RegisterOutput struct {
	ID       string
	FullName string
	Email    string
	Role     string
}

// Ví dụ transaction 2 module trong 1 lần ghi: xem mục 6 ARCHITECTURE_GUIDE.md. Usecase nhận thẳng
// *sql.DB, tự BeginTx/Rollback/Commit — không có TxManager/UnitOfWork, đây là pattern chuẩn của
// project này khi cần ghi nhiều bảng cùng lúc.
func (u *AuthUsecase) Register(ctx context.Context, input RegisterInput) (*RegisterOutput, error) {
	fullName := strings.TrimSpace(input.FullName)
	email := strings.ToLower(strings.TrimSpace(input.Email))

	exists, err := u.userReader.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if exists {
		return nil, apperror.BadRequest("Email đã được sử dụng", nil)
	}

	passwordHash, err := security.HashPassword(input.Password)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	defer tx.Rollback()

	user, err := u.userReader.CreateWithTx(ctx, tx, fullName, email)
	if err != nil {
		return nil, apperror.BadRequest("Dữ liệu đăng ký không hợp lệ", err)
	}

	if err := u.authRepo.CreateCredentialWithTx(ctx, tx, user.ID, passwordHash); err != nil {
		return nil, apperror.Internal(err)
	}

	if _, err := u.characterProvisioner.CreateDefaultWithTx(ctx, tx, user.ID, fullName); err != nil {
		return nil, apperror.Internal(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, apperror.Internal(err)
	}

	return &RegisterOutput{ID: user.ID, FullName: user.FullName, Email: user.Email, Role: user.Role}, nil
}
