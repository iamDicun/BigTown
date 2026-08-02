package port

import (
	"context"

	"backend/internal/module/editor/entity"
)

type EditorRepository interface {
	GetMapIDByCode(ctx context.Context, code string) (string, error)
	GetMapInfoByCode(ctx context.Context, code string) (*entity.MapInfo, error)
	GetDecorationItems(ctx context.Context) ([]entity.DecorationItem, error)
	GetPlacementsByMap(ctx context.Context, mapID string) ([]entity.Placement, error)
	GetItemByID(ctx context.Context, itemID string) (*entity.DecorationItem, error)
}
