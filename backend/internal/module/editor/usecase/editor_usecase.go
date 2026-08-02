package usecase

import (
	"context"
	"database/sql"
	"errors"

	"backend/internal/apperror"
	"backend/internal/module/editor/entity"
	"backend/internal/module/editor/port"
	"backend/internal/module/editor/room"
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

	liveCoins, _ := u.rooms.GetCoins(ctx, charInfo.ID)

	return &GetEditorDataOutput{
		Items:      items,
		Placements: placements,
		Coins:      liveCoins,
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

	a, err := u.rooms.Actor(input.MapCode)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if a == nil {
		return nil, apperror.NotFound("Không tìm thấy bản đồ", nil)
	}

	reply := make(chan room.CmdResult, 1)
	cmd := room.Cmd{
		Kind:    room.CmdPlace,
		CharID:  charInfo.ID,
		Item:    item,
		X:       input.X,
		Y:       input.Y,
		PlaceID: uuid.NewString(),
		Reply:   reply,
	}

	if err := a.SendCmd(cmd); err != nil {
		return nil, mapErr(err)
	}

	res := <-reply
	if res.Err != nil {
		return nil, mapErr(res.Err)
	}

	u.rooms.SetOnlineCoins(charInfo.ID, res.NewCoins)

	return &PlaceItemOutput{
		Placement: *res.Placement,
		NewCoins:  res.NewCoins,
	}, nil
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

func (u *EditorUsecase) ClaimCoinPickup(ctx context.Context, userID, mapCode, coinType string) (int, error) {
	if mapCode != "winter" && mapCode != "dark_village" {
		return 0, apperror.BadRequest("Bản đồ này không hỗ trợ nhặt coin", nil)
	}

	charInfo, err := u.charReader.GetByUserID(ctx, userID)
	if err != nil {
		return 0, apperror.NotFound("Không tìm thấy nhân vật", err)
	}

	var delta int
	switch coinType {
	case "gri":
		delta = 5
	case "ama":
		delta = 10
	case "azu":
		delta = 25
	case "roj":
		delta = 50
	case "gold":
		delta = 100
	default:
		delta = 10
	}

	// Credit delta coins to the resident character wallet in RAM
	newCoins, err := u.rooms.CreditCoins(ctx, mapCode, charInfo.ID, delta)
	if err != nil {
		return 0, apperror.Internal(err)
	}

	return newCoins, nil
}

