package usecase

import (
	"context"
	"database/sql"
	"errors"

	"backend/internal/apperror"
	"backend/internal/module/editor/entity"
	"backend/internal/module/editor/port"
	"backend/internal/module/editor/room"
	"encoding/json"
	"github.com/google/uuid"
)

type EditorUsecase struct {
	db         *sql.DB
	repo       port.EditorRepository
	charReader port.CharacterReader
	publisher  port.RoomPublisher
	rooms      *room.RoomManager
}

func NewEditorUsecase(db *sql.DB, repo port.EditorRepository, charReader port.CharacterReader, publisher port.RoomPublisher, rooms *room.RoomManager) *EditorUsecase {
	return &EditorUsecase{
		db:         db,
		repo:       repo,
		charReader: charReader,
		publisher:  publisher,
		rooms:      rooms,
	}
}

type GetEditorDataOutput struct {
	Items        []entity.DecorationItem `json:"items"`
	Placements   []entity.Placement      `json:"placements"`
	Coins        int                     `json:"coins"`
	SpawnedCoins []room.SpawnedCoin      `json:"spawned_coins"`
}

func (u *EditorUsecase) GetEditorData(ctx context.Context, userID string, mapCode string) (*GetEditorDataOutput, error) {
	charInfo, err := u.charReader.GetByUserID(ctx, userID)
	if err != nil {
		return nil, apperror.NotFound("Không tìm thấy nhân vật cho user", err)
	}

	mapID, err := u.repo.GetMapIDByCode(ctx, mapCode)
	if err != nil {
		return nil, apperror.NotFound("Không tìm thấy bản đồ", err)
	}

	items, err := u.repo.GetDecorationItems(ctx)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if items == nil {
		items = make([]entity.DecorationItem, 0)
	}

	var placements []entity.Placement
	if live := u.rooms.GetPlacements(mapCode); live != nil {
		placements = live
	} else {
		dbP, err := u.repo.GetPlacementsByMap(ctx, mapID)
		if err != nil {
			return nil, apperror.Internal(err)
		}
		placements = dbP
	}
	if placements == nil {
		placements = make([]entity.Placement, 0)
	}

	liveCoins, _ := u.rooms.GetCoins(ctx, charInfo.ID)
	spawnedCoins := u.rooms.GetSpawnedCoins(mapCode)
	if spawnedCoins == nil {
		spawnedCoins = make([]room.SpawnedCoin, 0)
	}

	return &GetEditorDataOutput{
		Items:        items,
		Placements:   placements,
		Coins:        liveCoins,
		SpawnedCoins: spawnedCoins,
	}, nil
}

type PlaceItemInput struct {
	ItemID   string `json:"item_id" binding:"required"`
	MapCode  string `json:"map_code" binding:"required"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	Rotation int    `json:"rotation"`
}

type PlaceItemOutput struct {
	Placement entity.Placement `json:"placement"`
	NewCoins  int              `json:"new_coins"`
}

func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, room.ErrOccupied) {
		return apperror.BadRequest("Ô này đã có vật thể", nil)
	}
	if errors.Is(err, room.ErrInsufficientCoins) {
		return apperror.BadRequest("Không đủ coins để mua vật phẩm này", nil)
	}
	if errors.Is(err, room.ErrNotOwner) {
		return apperror.Forbidden("Bạn không có quyền xóa vật phẩm này", nil)
	}
	if errors.Is(err, room.ErrNotFound) {
		return apperror.NotFound("Vật phẩm không tồn tại hoặc đã bị xóa", nil)
	}
	if errors.Is(err, room.ErrBusy) {
		return apperror.Internal(errors.New("hệ thống đang bận"))
	}
	return apperror.Internal(err)
}

func (u *EditorUsecase) PlaceItem(ctx context.Context, userID string, input PlaceItemInput) (*PlaceItemOutput, error) {
	charInfo, err := u.charReader.GetByUserID(ctx, userID)
	if err != nil {
		return nil, apperror.NotFound("Không tìm thấy nhân vật", err)
	}

	item, err := u.repo.GetItemByID(ctx, input.ItemID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if item == nil {
		return nil, apperror.BadRequest("Vật phẩm không tồn tại", nil)
	}

	mapID, err := u.repo.GetMapIDByCode(ctx, input.MapCode)
	if err != nil {
		return nil, apperror.NotFound("Không tìm thấy bản đồ", err)
	}

	// Validate coordinates (snap grid + bounds) — gọi actor để kiểm tra
	a, err := u.rooms.Actor(input.MapCode)
	if err != nil || a == nil {
		return nil, apperror.NotFound("Không tìm thấy bản đồ", nil)
	}
	if err := a.ValidatePlacement(input.X, input.Y); err != nil {
		return nil, mapErr(err)
	}

	placementID := uuid.NewString()

	// DB transaction: trừ coin + insert placement (atomic, DB tự serialize)
	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	defer tx.Rollback()

	// 1. Advisory Lock trên map_id + x + y để chống race condition
	lockQuery := `SELECT pg_advisory_xact_lock(hashtext($1::text), ($2 << 16) | $3)`
	if _, err := tx.ExecContext(ctx, lockQuery, mapID, input.X, input.Y); err != nil {
		return nil, apperror.Internal(err)
	}

	// 2. Query các placement hiện tại ở ô tọa độ này
	checkQuery := `SELECT p.item_id, i.metadata_json FROM map_placements p JOIN items i ON p.item_id = i.id WHERE p.map_id = $1 AND p.x = $2 AND p.y = $3`
	rows, err := tx.QueryContext(ctx, checkQuery, mapID, input.X, input.Y)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	defer rows.Close()

	var existingCount int
	var tileHasCollision bool
	for rows.Next() {
		existingCount++
		var itemID string
		var metadataJSON string
		if err := rows.Scan(&itemID, &metadataJSON); err != nil {
			return nil, apperror.Internal(err)
		}
		if parseMetadataCollides(metadataJSON) {
			tileHasCollision = true
		}
	}

	// 3. Thực hiện validation logic stacking + collision tương tự actor
	if existingCount >= 2 {
		return nil, apperror.BadRequest("Ô này đã có vật thể", nil)
	}

	if tileHasCollision {
		newCollides := parseMetadataCollides(item.MetadataJSON)
		if newCollides || existingCount > 0 {
			return nil, apperror.BadRequest("Ô này đã có vật thể", nil)
		}
	}

	// 4. Trừ coin
	var newCoins int
	coinQuery := `UPDATE characters SET coins = coins - $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2 AND coins >= $1 RETURNING coins`
	if err := tx.QueryRowContext(ctx, coinQuery, item.Price, charInfo.ID).Scan(&newCoins); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperror.BadRequest("Không đủ coins để mua vật phẩm này", nil)
		}
		return nil, apperror.Internal(err)
	}

	// 5. Insert placement vào DB
	placeQuery := `INSERT INTO map_placements (id, map_id, character_id, item_id, x, y, rotation)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`
	var insertedID string
	if err := tx.QueryRowContext(ctx, placeQuery, placementID, mapID, charInfo.ID, item.ID, input.X, input.Y, input.Rotation).Scan(&insertedID); err != nil {
		return nil, apperror.Internal(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, apperror.Internal(err)
	}

	u.rooms.SetOnlineCoins(charInfo.ID, newCoins)

	p := &entity.Placement{
		ID:          insertedID,
		MapID:       mapID,
		CharacterID: charInfo.ID,
		ItemID:      item.ID,
		X:           input.X,
		Y:           input.Y,
		Rotation:    input.Rotation,
	}
	u.rooms.RegisterPlacement(input.MapCode, p, charInfo.ID, newCoins, item.Price)

	return &PlaceItemOutput{Placement: *p, NewCoins: newCoins}, nil
}

func (u *EditorUsecase) DeletePlacement(ctx context.Context, userID, mapCode, placementID string) (int, error) {
	charInfo, err := u.charReader.GetByUserID(ctx, userID)
	if err != nil {
		return 0, apperror.NotFound("Không tìm thấy nhân vật", err)
	}

	a, err := u.rooms.Actor(mapCode)
	if err != nil {
		return 0, apperror.Internal(err)
	}
	if a == nil {
		return 0, apperror.NotFound("Không tìm thấy bản đồ", nil)
	}

	reply := make(chan room.CmdResult, 1)
	cmd := room.Cmd{
		Kind:     room.CmdDelete,
		CharID:   charInfo.ID,
		TargetID: placementID,
		Reply:    reply,
	}

	if err := a.SendCmd(cmd); err != nil {
		return 0, mapErr(err)
	}

	res := <-reply
	if res.Err != nil {
		return 0, mapErr(res.Err)
	}

	u.rooms.SetOnlineCoins(charInfo.ID, res.NewCoins)

	return res.NewCoins, nil
}

func (u *EditorUsecase) ClaimCoinPickup(ctx context.Context, userID, mapCode, coinID string) (int, error) {
	if mapCode != "winter" && mapCode != "dark_village" {
		return 0, apperror.BadRequest("Bản đồ này không hỗ trợ nhặt coin", nil)
	}

	charInfo, err := u.charReader.GetByUserID(ctx, userID)
	if err != nil {
		return 0, apperror.NotFound("Không tìm thấy nhân vật", err)
	}

	newCoins, err := u.rooms.ClaimCoin(ctx, mapCode, charInfo.ID, coinID)
	if err != nil {
		return 0, mapErr(err)
	}

	return newCoins, nil
}

func parseMetadataCollides(metadataJSON string) bool {
	if metadataJSON == "" || metadataJSON == "{}" {
		return false
	}
	var meta struct {
		Collides bool `json:"collides"`
	}
	if err := json.Unmarshal([]byte(metadataJSON), &meta); err != nil {
		return false
	}
	return meta.Collides
}

