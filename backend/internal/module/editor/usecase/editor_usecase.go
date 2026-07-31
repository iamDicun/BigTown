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
)

type editorTaskType string

const (
	taskPlace  editorTaskType = "place"
	taskDelete editorTaskType = "delete"
)

type editorDbTask struct {
	taskType    editorTaskType
	id          string
	mapID       string
	characterID string
	itemID      string
	x           int
	y           int
	price       int
}

type EditorUsecase struct {
	db         *sql.DB
	repo       port.EditorRepository
	charReader port.CharacterReader
	publisher  port.RoomPublisher
	writeChan  chan editorDbTask
}

func NewEditorUsecase(db *sql.DB, repo port.EditorRepository, charReader port.CharacterReader, publisher port.RoomPublisher) *EditorUsecase {
	u := &EditorUsecase{
		db:         db,
		repo:       repo,
		charReader: charReader,
		publisher:  publisher,
		writeChan:  make(chan editorDbTask, 10000),
	}
	u.startBackgroundWorkers(3)
	return u
}

func (u *EditorUsecase) startBackgroundWorkers(count int) {
	for i := 0; i < count; i++ {
		go func(workerID int) {
			ctx := context.Background()
			for task := range u.writeChan {
				var err error
				if task.taskType == taskPlace {
					err = u.executePlaceTask(ctx, task)
				} else if task.taskType == taskDelete {
					err = u.executeDeleteTask(ctx, task)
				}
				if err != nil {
					log.Printf("[Editor-Async-DB-Worker-%d] ERROR executing task %s: %v", workerID, task.taskType, err)
				}
			}
		}(i)
	}
}

func (u *EditorUsecase) executePlaceTask(ctx context.Context, task editorDbTask) error {
	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Deduct coins
	err = u.repo.DeductCoinsWithTx(ctx, tx, task.characterID, task.price)
	if err != nil {
		return err
	}

	// 2. Add placement
	placement := &entity.Placement{
		ID:          task.id,
		MapID:       task.mapID,
		CharacterID: task.characterID,
		ItemID:      task.itemID,
		X:           task.x,
		Y:           task.y,
	}
	err = u.repo.PlaceItemWithIDAndTx(ctx, tx, placement)
	if err != nil {
		return err
	}

	// 3. Log event
	err = u.repo.InsertRewardEventWithTx(ctx, tx, task.characterID, task.price)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (u *EditorUsecase) executeDeleteTask(ctx context.Context, task editorDbTask) error {
	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Refund coins 100%
	err = u.repo.AddCoinsWithTx(ctx, tx, task.characterID, task.price)
	if err != nil {
		return err
	}

	// 2. Delete placement
	err = u.repo.DeletePlacementWithTx(ctx, tx, task.id)
	if err != nil {
		return err
	}

	// 3. Log refund event
	err = u.repo.InsertRewardEventWithTx(ctx, tx, task.characterID, -task.price)
	if err != nil {
		return err
	}

	return tx.Commit()
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

func (u *EditorUsecase) PlaceItem(ctx context.Context, userID string, input PlaceItemInput) (*PlaceItemOutput, error) {
	charInfo, err := u.charReader.GetByUserID(ctx, userID)
	if err != nil {
		return nil, apperror.NotFound("Không tìm thấy nhân vật", err)
	}

	mapID, err := u.repo.GetMapIDByCode(ctx, input.MapCode)
	if err != nil {
		return nil, apperror.NotFound("Không tìm thấy bản đồ", err)
	}

	item, err := u.repo.GetItemByID(ctx, input.ItemID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if item == nil {
		return nil, apperror.BadRequest("Vật phẩm không tồn tại", nil)
	}

	if charInfo.Coins < item.Price {
		return nil, apperror.BadRequest("Không đủ coins để mua vật phẩm này", nil)
	}

	placementID := uuid.NewString()

	// Push async placement task
	select {
	case u.writeChan <- editorDbTask{
		taskType:    taskPlace,
		id:          placementID,
		mapID:       mapID,
		characterID: charInfo.ID,
		itemID:      input.ItemID,
		x:           input.X,
		y:           input.Y,
		price:       item.Price,
	}:
	default:
		log.Printf("[EditorUsecase] Queue full, dropping place task for placement: %s", placementID)
		return nil, apperror.Internal(errors.New("editor queue full"))
	}

	newCoins := charInfo.Coins - item.Price

	out := &PlaceItemOutput{
		Placement: entity.Placement{
			ID:          placementID,
			MapID:       mapID,
			CharacterID: charInfo.ID,
			ItemID:      input.ItemID,
			X:           input.X,
			Y:           input.Y,
			CreatedAt:   time.Now(),
		},
		NewCoins:  newCoins,
	}

	// Broadcast placement to the map channel
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

	// Push async delete task
	select {
	case u.writeChan <- editorDbTask{
		taskType:    taskDelete,
		id:          placementID,
		characterID: charInfo.ID,
		price:       item.Price,
	}:
	default:
		log.Printf("[EditorUsecase] Queue full, dropping delete task for placement: %s", placementID)
		return 0, apperror.Internal(errors.New("editor queue full"))
	}

	newCoins := charInfo.Coins + item.Price

	// Broadcast deletion to the map channel
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
