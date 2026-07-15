package usecase

import (
	"context"

	"backend/internal/apperror"
	"backend/internal/module/user/entity"
)

func (u *UserUsecase) GetUsers(ctx context.Context) ([]entity.User, error) {
	users, err := u.repo.FindAll(ctx)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	return users, nil
}
