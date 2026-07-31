package port

import (
	"context"
	"database/sql"

	"backend/internal/module/editor/entity"
)

type EditorRepository interface {
	GetMapIDByCode(ctx context.Context, code string) (string, error)
	GetMapCodeByID(ctx context.Context, id string) (string, error)
	GetDecorationItems(ctx context.Context) ([]entity.DecorationItem, error)
	GetPlacementsByMap(ctx context.Context, mapID string) ([]entity.Placement, error)
	GetItemByID(ctx context.Context, itemID string) (*entity.DecorationItem, error)
	GetPlacementByID(ctx context.Context, id string) (*entity.Placement, error)
	PlaceItemWithTx(ctx context.Context, tx *sql.Tx, placement *entity.Placement) error
	PlaceItemWithIDAndTx(ctx context.Context, tx *sql.Tx, placement *entity.Placement) error
	DeductCoinsWithTx(ctx context.Context, tx *sql.Tx, characterID string, amount int) error
	AddCoinsWithTx(ctx context.Context, tx *sql.Tx, characterID string, amount int) error
	DeletePlacementWithTx(ctx context.Context, tx *sql.Tx, id string) error
	InsertRewardEventWithTx(ctx context.Context, tx *sql.Tx, characterID string, amount int) error
}
