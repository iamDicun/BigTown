package room

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"backend/internal/module/editor/entity"
	"backend/internal/module/editor/port"
)

type mockCharReader struct {
	coins map[string]int
}

func (m *mockCharReader) GetByUserID(ctx context.Context, userID string) (*port.CharacterInfo, error) {
	return nil, nil
}

func (m *mockCharReader) GetCoins(ctx context.Context, characterID string) (int, error) {
	return m.coins[characterID], nil
}

type mockRoomPublisher struct {
	events []any
}

func (m *mockRoomPublisher) PublishRoom(ctx context.Context, roomID string, data interface{}) error {
	m.events = append(m.events, data)
	return nil
}

type mockEditorRepo struct {
	placements []entity.Placement
}

func (m *mockEditorRepo) GetMapIDByCode(ctx context.Context, code string) (string, error) {
	return "map-1", nil
}
func (m *mockEditorRepo) GetMapInfoByCode(ctx context.Context, code string) (*entity.MapInfo, error) {
	return &entity.MapInfo{ID: "map-1", Width: 1000, Height: 1000, TileSize: 16}, nil
}
func (m *mockEditorRepo) GetDecorationItems(ctx context.Context) ([]entity.DecorationItem, error) {
	return nil, nil
}
func (m *mockEditorRepo) GetPlacementsByMap(ctx context.Context, mapID string) ([]entity.Placement, error) {
	return m.placements, nil
}
func (m *mockEditorRepo) GetItemByID(ctx context.Context, itemID string) (*entity.DecorationItem, error) {
	return nil, nil
}

func TestMapActor_InmemorySafety(t *testing.T) {
	charID := "char-1"
	charReader := &mockCharReader{coins: map[string]int{charID: 90}}
	repo := &mockEditorRepo{}
	publisher := &mockRoomPublisher{}
	dirty := make(chan persistOp, 100)

	resolver := func(ctx context.Context, id string) (int, error) {
		return charReader.GetCoins(ctx, id)
	}
	actor := NewMapActor("map-1", "village", 1000, 1000, 16, charReader, repo, dirty, publisher, resolver)

	item := &entity.DecorationItem{ID: "item-1", Price: 90}

	// 1. Chạy N=50 concurrent commands đặt cùng 1 ô, hoặc các ô khác nhau
	n := 50
	var wg sync.WaitGroup
	wg.Add(n)

	successChan := make(chan CmdResult, n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			reply := make(chan CmdResult, 1)
			cmd := Cmd{
				Kind:    CmdPlace,
				CharID:  charID,
				Item:    item,
				X:       16,
				Y:       16,
				PlaceID: "p-1",
				Reply:   reply,
			}
			_ = actor.SendCmd(cmd)
			res := <-reply
			successChan <- res
		}()
	}

	wg.Wait()
	close(successChan)

	successCount := 0
	occupiedCount := 0
	for res := range successChan {
		if res.Err == nil {
			successCount++
		} else if errors.Is(res.Err, ErrOccupied) {
			occupiedCount++
		}
	}

	if successCount != 1 {
		t.Errorf("Expected exactly 1 success for duplicate coordinate placement, got %d", successCount)
	}

	// 2. Chạy test coin safety: Đặt 2 items ở toạ độ khác nhau (đòi hỏi 180 coins nhưng chỉ có 90 coins)
	reply1 := make(chan CmdResult, 1)
	actor.SendCmd(Cmd{
		Kind:    CmdPlace,
		CharID:  charID,
		Item:    item,
		X:       32,
		Y:       32,
		PlaceID: "p-2",
		Reply:   reply1,
	})
	res1 := <-reply1

	if !errors.Is(res1.Err, ErrInsufficientCoins) {
		t.Errorf("Expected ErrInsufficientCoins for second item, got: %v", res1.Err)
	}

	close(actor.cmds)
}

func TestMapActor_WalletResidency_Lifecycle(t *testing.T) {
	charID := "char-resident"
	// Khởi tạo DB mock trả về 1000 coins nếu lazy load
	charReader := &mockCharReader{coins: map[string]int{charID: 1000}}
	repo := &mockEditorRepo{}
	publisher := &mockRoomPublisher{}
	dirty := make(chan persistOp, 100)

	resolver := func(ctx context.Context, id string) (int, error) {
		return charReader.GetCoins(ctx, id)
	}
	actor := NewMapActor("map-1", "village", 1000, 1000, 16, charReader, repo, dirty, publisher, resolver)
	defer close(actor.cmds)

	item := &entity.DecorationItem{ID: "item-1", Price: 40}

	// 1. Join room với 100 coins (ghi đè giá trị lazy load 1000 coins trong DB)
	actor.cmds <- Cmd{
		Kind:   CmdJoin,
		CharID: charID,
		Coins:  100,
	}

	// 2. Đặt 1 item giá 40 coins. Ví trong memory sẽ còn 60 coins.
	placeReply := make(chan CmdResult, 1)
	actor.cmds <- Cmd{
		Kind:    CmdPlace,
		CharID:  charID,
		Item:    item,
		X:       16,
		Y:       16,
		PlaceID: "p-1",
		Reply:   placeReply,
	}

	res := <-placeReply
	if res.Err != nil {
		t.Fatalf("Failed to place item: %v", res.Err)
	}
	if res.NewCoins != 60 {
		t.Errorf("Expected remaining coins to be 60, got %d", res.NewCoins)
	}

	// 3. Leave room -> trigger evict và flush
	actor.cmds <- Cmd{
		Kind:   CmdLeave,
		CharID: charID,
	}

	// Chờ opFlushWallet bắn sang dirty channel
	var op persistOp
	select {
	case op = <-dirty:
		// có thể nhận được opPlace trước
		if op.Kind == opPlace {
			// nhận tiếp
			op = <-dirty
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for write-behind flush operation")
	}

	if op.Kind != opFlushWallet {
		t.Errorf("Expected flush wallet op, got: %v", op.Kind)
	}
	if op.NewCoins != 60 {
		t.Errorf("Expected flushed coins to be 60, got %d", op.NewCoins)
	}

	// 4. Verify eviction: Đặt item tiếp theo ở toạ độ khác.
	// Vì ví đã bị evict, actor buộc phải lazy load từ DB mock (đang trả về 1000 coins).
	placeReply2 := make(chan CmdResult, 1)
	actor.cmds <- Cmd{
		Kind:    CmdPlace,
		CharID:  charID,
		Item:    item,
		X:       32,
		Y:       32,
		PlaceID: "p-2",
		Reply:   placeReply2,
	}

	res2 := <-placeReply2
	if res2.Err != nil {
		t.Fatalf("Failed to place second item: %v", res2.Err)
	}
	if res2.NewCoins != 960 {
		t.Errorf("Eviction failed! Expected coins loaded from DB to be 960, got %d", res2.NewCoins)
	}
}

func TestMapActor_CmdCredit(t *testing.T) {
	charID := "char-credit-test"
	charReader := &mockCharReader{coins: map[string]int{charID: 100}}
	repo := &mockEditorRepo{}
	publisher := &mockRoomPublisher{}
	dirty := make(chan persistOp, 100)

	resolver := func(ctx context.Context, id string) (int, error) {
		return charReader.GetCoins(ctx, id)
	}
	actor := NewMapActor("map-1", "village", 1000, 1000, 16, charReader, repo, dirty, publisher, resolver)
	defer close(actor.cmds)

	// Test credit 50 coins via CmdCredit
	reply := make(chan CmdResult, 1)
	actor.cmds <- Cmd{
		Kind:   CmdCredit,
		CharID: charID,
		Coins:  50,
		Reply:  reply,
	}

	res := <-reply
	if res.Err != nil {
		t.Fatalf("CmdCredit failed: %v", res.Err)
	}
	if res.NewCoins != 150 {
		t.Errorf("Expected new coins to be 150, got %d", res.NewCoins)
	}

	// Verify dirty opFlushWallet was generated
	select {
	case op := <-dirty:
		if op.Kind != opFlushWallet {
			t.Errorf("Expected opFlushWallet, got: %v", op.Kind)
		}
		if op.NewCoins != 150 {
			t.Errorf("Expected flushed coins to be 150, got %d", op.NewCoins)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for write-behind opFlushWallet")
	}
}
