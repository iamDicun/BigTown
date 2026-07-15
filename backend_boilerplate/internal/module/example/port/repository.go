package port

import (
	"context"

	"backend/internal/module/example/entity"
)

type ItemRepository interface {
	FindAll(ctx context.Context) ([]entity.Item, error)
	FindByID(ctx context.Context, id string) (*entity.Item, error)
	Create(ctx context.Context, name string) (*entity.Item, error)
}
