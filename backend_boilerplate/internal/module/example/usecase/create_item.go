package usecase

import (
	"context"
	"strings"

	"backend/internal/apperror"
	"backend/internal/module/example/entity"
)

type CreateItemInput struct {
	Name string
}

func (u *ExampleUsecase) CreateItem(ctx context.Context, input CreateItemInput) (*entity.Item, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, apperror.BadRequest("Tên item không được để trống", nil)
	}

	item, err := u.repo.Create(ctx, name)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	return item, nil
}
