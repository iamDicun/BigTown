package usecase

import (
	"context"
	"database/sql"
	"errors"

	"backend/internal/apperror"
	"backend/internal/module/example/entity"
)

func (u *ExampleUsecase) GetItemByID(ctx context.Context, id string) (*entity.Item, error) {
	item, err := u.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperror.NotFound("Không tìm thấy item", err)
		}
		return nil, apperror.Internal(err)
	}
	return item, nil
}
