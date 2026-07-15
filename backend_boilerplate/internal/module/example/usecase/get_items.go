package usecase

import (
	"context"

	"backend/internal/apperror"
	"backend/internal/module/example/entity"
)

func (u *ExampleUsecase) GetItems(ctx context.Context) ([]entity.Item, error) {
	items, err := u.repo.FindAll(ctx)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	return items, nil
}
