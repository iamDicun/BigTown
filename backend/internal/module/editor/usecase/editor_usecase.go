package usecase

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"

	"backend/internal/apperror"
	"backend/internal/module/editor/entity"
	"backend/internal/module/editor/port"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

type EditorUsecase struct {
	db         *sql.DB
	repo       port.EditorRepository
	charReader port.CharacterReader
	publisher  port.RoomPublisher
}

func NewEditorUsecase(db *sql.DB, repo port.EditorRepository, charReader port.CharacterReader, publisher port.RoomPublisher) *EditorUsecase {
	return &EditorUsecase{
		db:         db,
		repo:       repo,
		charReader: charReader,
		publisher:  publisher,
	}
}

type GetEditorDataOutput struct {
	Items      []entity.DecorationItem `json:"items"`
	Placements []entity.Placement      `json:"placements"`
	Coins      int                     `json:"coins"`
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

	placements, err := u.repo.GetPlacementsByMap(ctx, mapID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if placements == nil {
		placements = make([]entity.Placement, 0)
	}

	return &GetEditorDataOutput{
		Items:      items,
		Placements: placements,
		Coins:      charInfo.Coins,
	}, nil
}

type PlaceItemInput struct {
	ItemID  string `json:"item_id" binding:"required"`
	MapCode string `json:"map_code" binding:"required"`
	X       int    `json:"x"`
	Y       int    `json:"y"`
}

type PlaceItemOutput struct {
	Placement entity.Placement `json:"placement"`
	NewCoins  int              `json:"new_coins"`
}

func validatePlacement(x, y, mapWidth, mapHeight, tileSize int) error {
	if tileSize <= 0 {
		return errors.New("tileSize must be positive")
	}
	if x%tileSize != 0 || y%tileSize != 0 {
		return errors.New("toạ độ không khớp snap grid")
	}
	if x < 0 || x >= mapWidth || y < 0 || y >= mapHeight {
		return errors.New("toạ độ vượt quá giới hạn bản đồ")
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func (u *EditorUsecase) PlaceItem(ctx context.Context, userID string, input PlaceItemInput) (*PlaceItemOutput, error) {
	charInfo, err := u.charReader.GetByUserID(ctx, userID)
	if err != nil {
		return nil, apperror.NotFound("Không tìm thấy nhân vật", err)
	}

	mapInfo, err := u.repo.GetMapInfoByCode(ctx, input.MapCode)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if mapInfo == nil {
		return nil, apperror.NotFound("Không tìm thấy bản đồ", nil)
	}

	item, err := u.repo.GetItemByID(ctx, input.ItemID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if item == nil {
		return nil, apperror.BadRequest("Vật phẩm không tồn tại", nil)
	}

	// Validate coordinates server-side (P5)
	if err := validatePlacement(input.X, input.Y, mapInfo.Width, mapInfo.Height, mapInfo.TileSize); err != nil {
		return nil, apperror.BadRequest(err.Error(), nil)
	}

	placementID := uuid.NewString()

	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	defer tx.Rollback()

	// 1. Trừ coins atomic (P1)
	newCoins, err := u.repo.DeductCoinsGuardedWithTx(ctx, tx, charInfo.ID, item.Price)
	if err != nil {
		if errors.Is(err, port.ErrInsufficientCoins) {
			return nil, apperror.BadRequest("Không đủ coins để mua vật phẩm này", nil)
		}
		return nil, apperror.Internal(err)
	}

	// 2. Insert placement và check trùng coordinates (P2)
	placement := &entity.Placement{
		ID:          placementID,
		MapID:       mapInfo.ID,
		CharacterID: charInfo.ID,
		ItemID:      input.ItemID,
		X:           input.X,
		Y:           input.Y,
	}
	if err := u.repo.PlaceItemWithIDAndTx(ctx, tx, placement); err != nil {
		if isUniqueViolation(err) {
			return nil, apperror.BadRequest("Ô này đã có vật thể", nil)
		}
		return nil, apperror.Internal(err)
	}

	// 3. Log event (P6)
	if err := u.repo.InsertRewardEventWithTx(ctx, tx, charInfo.ID, "decoration_place", -item.Price); err != nil {
		return nil, apperror.Internal(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, apperror.Internal(err)
	}

	placement.CreatedAt = time.Now()
	out := &PlaceItemOutput{
		Placement: *placement,
		NewCoins:  newCoins,
	}

	// 4. Broadcast sau commit (P2)
	event := map[string]interface{}{
		"type":      "decoration_placed",
		"placement": out.Placement,
	}
	if err := u.publisher.PublishRoom(ctx, input.MapCode, event); err != nil {
		log.Printf("[EditorUsecase] Failed to broadcast decoration_placed: %v", err)
	}

	return out, nil
}

func (u *EditorUsecase) DeletePlacement(ctx context.Context, userID string, placementID string) (int, error) {
	charInfo, err := u.charReader.GetByUserID(ctx, userID)
	if err != nil {
		return 0, apperror.NotFound("Không tìm thấy nhân vật", err)
	}

	placement, err := u.repo.GetPlacementByID(ctx, placementID)
	if err != nil {
		return 0, apperror.Internal(err)
	}
	if placement == nil {
		return 0, apperror.NotFound("Vật phẩm không tồn tại hoặc đã bị xóa", nil)
	}

	if placement.CharacterID != charInfo.ID {
		return 0, apperror.Forbidden("Bạn không có quyền xóa vật phẩm này", nil)
	}

	item, err := u.repo.GetItemByID(ctx, placement.ItemID)
	if err != nil {
		return 0, apperror.Internal(err)
	}
	if item == nil {
		return 0, apperror.Internal(errors.New("item not found"))
	}

	// Fetch mapCode for Centrifuge broadcasting
	mapCode, err := u.repo.GetMapCodeByID(ctx, placement.MapID)
	if err != nil {
		log.Printf("[EditorUsecase] Failed to get map code for ID %s: %v", placement.MapID, err)
	}

	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, apperror.Internal(err)
	}
	defer tx.Rollback()

	// 1. DELETE placement trước để guard double deletion (P3)
	if err := u.repo.DeletePlacementWithTx(ctx, tx, placementID); err != nil {
		if err.Error() == "placement not found" {
			return 0, apperror.NotFound("Vật phẩm không tồn tại hoặc đã bị xóa", nil)
		}
		return 0, apperror.Internal(err)
	}

	// 2. Refund coins 100%
	newCoins, err := u.repo.AddCoinsGuardedWithTx(ctx, tx, charInfo.ID, item.Price)
	if err != nil {
		return 0, apperror.Internal(err)
	}

	// 3. Log refund event
	if err := u.repo.InsertRewardEventWithTx(ctx, tx, charInfo.ID, "decoration_refund", item.Price); err != nil {
		return 0, apperror.Internal(err)
	}

	if err := tx.Commit(); err != nil {
		return 0, apperror.Internal(err)
	}

	// 4. Broadcast sau commit
	if mapCode != "" {
		event := map[string]interface{}{
			"type":        "decoration_deleted",
			"placementId": placementID,
		}
		if err := u.publisher.PublishRoom(ctx, mapCode, event); err != nil {
			log.Printf("[EditorUsecase] Failed to broadcast decoration_deleted: %v", err)
		}
	}

	return newCoins, nil
}

