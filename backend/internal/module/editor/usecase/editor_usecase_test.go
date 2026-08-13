package usecase

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"backend/internal/module/editor/entity"
	"backend/internal/module/editor/port"
	"backend/internal/module/editor/room"
)

type mockEditorRepo struct {
	getMapIDByCodeFn     func(ctx context.Context, code string) (string, error)
	getMapInfoByCodeFn   func(ctx context.Context, code string) (*entity.MapInfo, error)
	getDecorationItemsFn func(ctx context.Context) ([]entity.DecorationItem, error)
	getPlacementsByMapFn func(ctx context.Context, mapID string) ([]entity.Placement, error)
	getItemByIDFn        func(ctx context.Context, itemID string) (*entity.DecorationItem, error)
}

func (m *mockEditorRepo) GetMapIDByCode(ctx context.Context, code string) (string, error) {
	if m.getMapIDByCodeFn != nil {
		return m.getMapIDByCodeFn(ctx, code)
	}
	return "map-123", nil
}
func (m *mockEditorRepo) GetMapInfoByCode(ctx context.Context, code string) (*entity.MapInfo, error) {
	if m.getMapInfoByCodeFn != nil {
		return m.getMapInfoByCodeFn(ctx, code)
	}
	return &entity.MapInfo{ID: "map-123", Width: 100, Height: 100, TileSize: 32}, nil
}
func (m *mockEditorRepo) GetDecorationItems(ctx context.Context) ([]entity.DecorationItem, error) {
	if m.getDecorationItemsFn != nil {
		return m.getDecorationItemsFn(ctx)
	}
	return []entity.DecorationItem{{ID: "item-1", Name: "Tree", Price: 10}}, nil
}
func (m *mockEditorRepo) GetPlacementsByMap(ctx context.Context, mapID string) ([]entity.Placement, error) {
	if m.getPlacementsByMapFn != nil {
		return m.getPlacementsByMapFn(ctx, mapID)
	}
	return []entity.Placement{}, nil
}
func (m *mockEditorRepo) GetItemByID(ctx context.Context, itemID string) (*entity.DecorationItem, error) {
	if m.getItemByIDFn != nil {
		return m.getItemByIDFn(ctx, itemID)
	}
	if itemID == "item-1" {
		return &entity.DecorationItem{ID: "item-1", Name: "Tree", Price: 10}, nil
	}
	return nil, nil
}

var _ port.EditorRepository = (*mockEditorRepo)(nil)

type mockEditorCharReader struct {
	getByUserIDFn func(ctx context.Context, userID string) (*port.CharacterInfo, error)
	getCoinsFn    func(ctx context.Context, characterID string) (int, error)
}

func (m *mockEditorCharReader) GetByUserID(ctx context.Context, userID string) (*port.CharacterInfo, error) {
	if m.getByUserIDFn != nil {
		return m.getByUserIDFn(ctx, userID)
	}
	return &port.CharacterInfo{ID: "char-1", Coins: 100}, nil
}
func (m *mockEditorCharReader) GetCoins(ctx context.Context, characterID string) (int, error) {
	if m.getCoinsFn != nil {
		return m.getCoinsFn(ctx, characterID)
	}
	return 100, nil
}

var _ port.CharacterReader = (*mockEditorCharReader)(nil)

type mockEditorPublisher struct{}

func (m *mockEditorPublisher) PublishRoom(ctx context.Context, roomID string, event any) error {
	return nil
}

var _ port.RoomPublisher = (*mockEditorPublisher)(nil)

func TestMapErr(t *testing.T) {
	if mapErr(nil) != nil {
		t.Error("expected nil for nil error")
	}
	if err := mapErr(room.ErrOccupied); err == nil {
		t.Error("expected error for ErrOccupied")
	}
	if err := mapErr(room.ErrInsufficientCoins); err == nil {
		t.Error("expected error for ErrInsufficientCoins")
	}
	if err := mapErr(room.ErrNotOwner); err == nil {
		t.Error("expected error for ErrNotOwner")
	}
	if err := mapErr(room.ErrNotFound); err == nil {
		t.Error("expected error for ErrNotFound")
	}
}

func TestGetEditorData_Success(t *testing.T) {
	repo := &mockEditorRepo{}
	charReader := &mockEditorCharReader{}
	rm := room.NewRoomManager(nil, &mockEditorPublisher{}, repo, charReader)

	uc := NewEditorUsecase(nil, repo, charReader, &mockEditorPublisher{}, rm)

	data, err := uc.GetEditorData(context.Background(), "user-1", "map-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(data.Items))
	}
	if data.Coins != 100 {
		t.Errorf("expected 100 coins, got %d", data.Coins)
	}
}

func TestGetEditorData_CharacterNotFound(t *testing.T) {
	charReader := &mockEditorCharReader{
		getByUserIDFn: func(ctx context.Context, userID string) (*port.CharacterInfo, error) {
			return nil, sql.ErrNoRows
		},
	}
	rm := room.NewRoomManager(nil, &mockEditorPublisher{}, &mockEditorRepo{}, charReader)
	uc := NewEditorUsecase(nil, &mockEditorRepo{}, charReader, &mockEditorPublisher{}, rm)

	_, err := uc.GetEditorData(context.Background(), "user-unknown", "map-1")
	if err == nil {
		t.Error("expected error for non-existing character")
	}
}

func TestPlaceItem_Validation(t *testing.T) {
	repo := &mockEditorRepo{
		getItemByIDFn: func(ctx context.Context, itemID string) (*entity.DecorationItem, error) {
			return nil, nil // item not found
		},
	}
	charReader := &mockEditorCharReader{}
	rm := room.NewRoomManager(nil, &mockEditorPublisher{}, repo, charReader)
	uc := NewEditorUsecase(nil, repo, charReader, &mockEditorPublisher{}, rm)

	_, err := uc.PlaceItem(context.Background(), "user-1", PlaceItemInput{
		ItemID: "non-existent-item", MapCode: "map-1", X: 10, Y: 10,
	})
	if err == nil {
		t.Error("expected error for non-existent item")
	}
}

func TestDeletePlacement_UserNotFound(t *testing.T) {
	charReader := &mockEditorCharReader{
		getByUserIDFn: func(ctx context.Context, userID string) (*port.CharacterInfo, error) {
			return nil, errors.New("user not found")
		},
	}
	rm := room.NewRoomManager(nil, &mockEditorPublisher{}, &mockEditorRepo{}, charReader)
	uc := NewEditorUsecase(nil, &mockEditorRepo{}, charReader, &mockEditorPublisher{}, rm)

	_, err := uc.DeletePlacement(context.Background(), "user-unknown", "map-1", "place-1")
	if err == nil {
		t.Error("expected error for missing user character")
	}
}
