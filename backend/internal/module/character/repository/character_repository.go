package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"backend/internal/module/character/entity"
	"backend/internal/module/character/port"
)

var _ port.CharacterRepository = (*CharacterRepository)(nil)

const characterColumns = `id::text, user_id::text, name, map_id::text, base_asset_key, coins, score, last_x, last_y`

const selectCharacterByUserIDQuery = `SELECT ` + characterColumns + ` FROM characters WHERE user_id = $1`

const insertDefaultCharacterQuery = `
	INSERT INTO characters (user_id, name, base_asset_key, map_id)
	VALUES ($1, $2, $3, $4)
	RETURNING ` + characterColumns

const updateCharacterMapIDQuery = `UPDATE characters SET map_id = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1`

const mapColumns = `id::text, code, name, tilemap_asset_key, tileset_asset_key, collision_asset_key, spawn_x, spawn_y, width, height, COALESCE(tile_size, 16), COALESCE(layer_names, ''), COALESCE(above_layer_name, ''), COALESCE(collision_layer_name, '')`

const selectMapByCodeQuery = `SELECT ` + mapColumns + ` FROM maps WHERE code = $1`

// CharacterRepository.defaultMapCode = GAME_DEFAULT_MAP_CODE (xem docs/Architecture.md mục 9.1).
// Đây là điểm cấu hình duy nhất quyết định character mới/cũ thuộc map nào.
type CharacterRepository struct {
	db             *sql.DB
	defaultMapCode string
}

func NewCharacterRepository(db *sql.DB, defaultMapCode string) *CharacterRepository {
	return &CharacterRepository{db: db, defaultMapCode: defaultMapCode}
}

func (r *CharacterRepository) FindByUserID(ctx context.Context, userID string) (*entity.Character, error) {
	return scanCharacter(r.db.QueryRowContext(ctx, selectCharacterByUserIDQuery, userID))
}

func (r *CharacterRepository) CreateWithTx(ctx context.Context, tx *sql.Tx, userID string, name string, baseAssetKey string) (*entity.Character, error) {
	var mapID *string

	mapInfo, err := scanMap(tx.QueryRowContext(ctx, selectMapByCodeQuery, r.defaultMapCode))
	if err == nil {
		mapID = &mapInfo.ID
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	return scanCharacter(tx.QueryRowContext(ctx, insertDefaultCharacterQuery, userID, name, baseAssetKey, mapID))
}

func (r *CharacterRepository) FindMapByCode(ctx context.Context, code string) (*entity.MapInfo, error) {
	return scanMap(r.db.QueryRowContext(ctx, selectMapByCodeQuery, code))
}

// SyncMapID đồng bộ map_id của character theo map mặc định hiện hành mỗi lần được gọi (login/bootstrap).
// Không ghi DB nếu đã khớp; không chặn nếu map mặc định
// chưa seed (giữ nguyên currentMapID).
func (r *CharacterRepository) SyncMapID(ctx context.Context, characterID string, currentMapID *string) (*string, error) {
	mapInfo, err := r.FindMapByCode(ctx, r.defaultMapCode)
	if errors.Is(err, sql.ErrNoRows) {
		return currentMapID, nil
	}
	if err != nil {
		return nil, err
	}

	if currentMapID != nil && *currentMapID == mapInfo.ID {
		return currentMapID, nil
	}

	if _, err := r.db.ExecContext(ctx, updateCharacterMapIDQuery, characterID, mapInfo.ID); err != nil {
		return nil, err
	}

	return &mapInfo.ID, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanCharacter(row rowScanner) (*entity.Character, error) {
	var c entity.Character
	var mapID sql.NullString
	var lastX, lastY sql.NullInt32

	if err := row.Scan(&c.ID, &c.UserID, &c.Name, &mapID, &c.BaseAssetKey, &c.Coins, &c.Score, &lastX, &lastY); err != nil {
		return nil, err
	}

	if mapID.Valid {
		c.MapID = &mapID.String
	}
	if lastX.Valid {
		v := int(lastX.Int32)
		c.LastX = &v
	}
	if lastY.Valid {
		v := int(lastY.Int32)
		c.LastY = &v
	}

	return &c, nil
}

func scanMap(row rowScanner) (*entity.MapInfo, error) {
	var m entity.MapInfo
	var collisionAssetKey sql.NullString
	var layerNamesRaw string

	if err := row.Scan(
		&m.ID, &m.Code, &m.Name, &m.TilemapAssetKey, &m.TilesetAssetKey, &collisionAssetKey,
		&m.SpawnX, &m.SpawnY, &m.Width, &m.Height, &m.TileSize,
		&layerNamesRaw, &m.AboveLayerName, &m.CollisionLayerName,
	); err != nil {
		return nil, err
	}

	if collisionAssetKey.Valid {
		m.CollisionAssetKey = &collisionAssetKey.String
	}
	m.LayerNames = parseLayerNames(layerNamesRaw)

	return &m, nil
}

func parseLayerNames(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
