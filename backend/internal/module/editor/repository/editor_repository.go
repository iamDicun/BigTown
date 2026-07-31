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

func (r *EditorRepository) GetMapCodeByID(ctx context.Context, id string) (string, error) {
	query := `SELECT code FROM maps WHERE id = $1`
	var code string
	err := r.db.QueryRowContext(ctx, query, id).Scan(&code)
	if err != nil {
		return "", err
	}
	return code, nil
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
	query := `SELECT id::text, map_id::text, character_id::text, item_id::text, x, y, created_at FROM map_placements WHERE map_id = $1 ORDER BY created_at ASC`
	rows, err := r.db.QueryContext(ctx, query, mapID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var placements []entity.Placement
	for rows.Next() {
		var p entity.Placement
		if err := rows.Scan(&p.ID, &p.MapID, &p.CharacterID, &p.ItemID, &p.X, &p.Y, &p.CreatedAt); err != nil {
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

func (r *EditorRepository) GetPlacementByID(ctx context.Context, id string) (*entity.Placement, error) {
	query := `SELECT id::text, map_id::text, character_id::text, item_id::text, x, y, created_at FROM map_placements WHERE id = $1`
	var p entity.Placement
	err := r.db.QueryRowContext(ctx, query, id).Scan(&p.ID, &p.MapID, &p.CharacterID, &p.ItemID, &p.X, &p.Y, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *EditorRepository) PlaceItemWithTx(ctx context.Context, tx *sql.Tx, placement *entity.Placement) error {
	query := `INSERT INTO map_placements (map_id, character_id, item_id, x, y) VALUES ($1, $2, $3, $4, $5) RETURNING id::text, created_at`
	return tx.QueryRowContext(ctx, query, placement.MapID, placement.CharacterID, placement.ItemID, placement.X, placement.Y).Scan(&placement.ID, &placement.CreatedAt)
}

func (r *EditorRepository) PlaceItemWithIDAndTx(ctx context.Context, tx *sql.Tx, placement *entity.Placement) error {
	query := `INSERT INTO map_placements (id, map_id, character_id, item_id, x, y) VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := tx.ExecContext(ctx, query, placement.ID, placement.MapID, placement.CharacterID, placement.ItemID, placement.X, placement.Y)
	return err
}

func (r *EditorRepository) DeductCoinsWithTx(ctx context.Context, tx *sql.Tx, characterID string, amount int) error {
	query := `UPDATE characters SET coins = coins - $1 WHERE id = $2`
	res, err := tx.ExecContext(ctx, query, amount, characterID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("character not found")
	}
	return nil
}

func (r *EditorRepository) AddCoinsWithTx(ctx context.Context, tx *sql.Tx, characterID string, amount int) error {
	query := `UPDATE characters SET coins = coins + $1 WHERE id = $2`
	res, err := tx.ExecContext(ctx, query, amount, characterID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("character not found")
	}
	return nil
}

func (r *EditorRepository) DeletePlacementWithTx(ctx context.Context, tx *sql.Tx, id string) error {
	query := `DELETE FROM map_placements WHERE id = $1`
	res, err := tx.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("placement not found")
	}
	return nil
}

func (r *EditorRepository) InsertRewardEventWithTx(ctx context.Context, tx *sql.Tx, characterID string, amount int) error {
	query := `INSERT INTO reward_events (character_id, event_type, coin_delta) VALUES ($1, 'decoration_place', $2)`
	_, err := tx.ExecContext(ctx, query, characterID, -amount)
	return err
}
