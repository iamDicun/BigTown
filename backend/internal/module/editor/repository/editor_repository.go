package repository

import (
	"context"
	"database/sql"
	"errors"

	"backend/internal/module/editor/entity"
	"backend/internal/module/editor/port"
)

var _ port.EditorRepository = (*EditorRepository)(nil)

type EditorRepository struct {
	db *sql.DB
}

func NewEditorRepository(db *sql.DB) *EditorRepository {
	return &EditorRepository{db: db}
}

func (r *EditorRepository) GetMapIDByCode(ctx context.Context, code string) (string, error) {
	query := `SELECT id::text FROM maps WHERE code = $1`
	var id string
	err := r.db.QueryRowContext(ctx, query, code).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (r *EditorRepository) GetDecorationItems(ctx context.Context) ([]entity.DecorationItem, error) {
	query := `SELECT id::text, code, name, type, asset_key, price, COALESCE(metadata_json::text, '{}') FROM items WHERE type = 'decoration' ORDER BY price ASC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []entity.DecorationItem
	for rows.Next() {
		var item entity.DecorationItem
		if err := rows.Scan(&item.ID, &item.Code, &item.Name, &item.Type, &item.AssetKey, &item.Price, &item.MetadataJSON); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *EditorRepository) GetPlacementsByMap(ctx context.Context, mapID string) ([]entity.Placement, error) {
	query := `SELECT id::text, map_id::text, character_id::text, item_id::text, x, y, COALESCE(rotation, 0), created_at FROM map_placements WHERE map_id = $1 ORDER BY created_at ASC`
	rows, err := r.db.QueryContext(ctx, query, mapID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var placements []entity.Placement
	for rows.Next() {
		var p entity.Placement
		if err := rows.Scan(&p.ID, &p.MapID, &p.CharacterID, &p.ItemID, &p.X, &p.Y, &p.Rotation, &p.CreatedAt); err != nil {
			return nil, err
		}
		placements = append(placements, p)
	}
	return placements, nil
}

func (r *EditorRepository) GetItemByID(ctx context.Context, itemID string) (*entity.DecorationItem, error) {
	query := `SELECT id::text, code, name, type, asset_key, price, COALESCE(metadata_json::text, '{}') FROM items WHERE id = $1`
	var item entity.DecorationItem
	err := r.db.QueryRowContext(ctx, query, itemID).Scan(&item.ID, &item.Code, &item.Name, &item.Type, &item.AssetKey, &item.Price, &item.MetadataJSON)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *EditorRepository) GetMapInfoByCode(ctx context.Context, code string) (*entity.MapInfo, error) {
	query := `SELECT id::text, width, height, tile_size FROM maps WHERE code = $1`
	var m entity.MapInfo
	err := r.db.QueryRowContext(ctx, query, code).Scan(&m.ID, &m.Width, &m.Height, &m.TileSize)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}
