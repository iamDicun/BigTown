package usecase

import (
	"context"
	"database/sql"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"backend/internal/database"
	"backend/internal/module/editor/port"
	"backend/internal/module/editor/repository"
	"backend/internal/module/editor/room"
	"backend/internal/platform/config"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

// Helper setup DB
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	_, b, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(b), "../../../../..")
	envPath := filepath.Join(projectRoot, ".env")
	_ = godotenv.Load(envPath)

	cfg := config.Load()
	pg := database.NewPostgresDB(cfg.Database)

	return pg.DB, func() {
		pg.Close()
	}
}

func TestPlaceItem_Concurrently(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// 1. Tạo app_user test
	var userID string
	err := db.QueryRowContext(ctx, "INSERT INTO app_user (full_name, email, role) VALUES ('Test User', 'test_concurrent@example.com', 'User') RETURNING id::text").Scan(&userID)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	defer db.ExecContext(ctx, "DELETE FROM app_user WHERE id = $1", userID)

	// Lấy map village_adventure
	var mapID string
	err = db.QueryRowContext(ctx, "SELECT id::text FROM maps WHERE code = 'village_adventure'").Scan(&mapID)
	if err != nil {
		t.Fatalf("Failed to get map village_adventure: %v", err)
	}

	// 2. Tạo character test với 90 coins
	var charID string
	err = db.QueryRowContext(ctx, "INSERT INTO characters (user_id, name, map_id, base_asset_key, coins) VALUES ($1, 'Test Character', $2, 'sprite-1', 90) RETURNING id::text", userID, mapID).Scan(&charID)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}
	defer db.ExecContext(ctx, "DELETE FROM characters WHERE id = $1", charID)

	// Lấy item deco_chest (giá 90 coins)
	var itemID string
	var itemPrice int
	err = db.QueryRowContext(ctx, "SELECT id::text, price FROM items WHERE code = 'deco_chest'").Scan(&itemID, &itemPrice)
	if err != nil {
		t.Fatalf("Failed to get item deco_chest: %v", err)
	}

	repo := repository.NewEditorRepository(db)
	charReader := &mockCharReader{charID: charID, db: db}
	publisher := &mockRoomPublisher{}

	rooms := room.NewRoomManager(db, publisher, repo, charReader)
	uc := NewEditorUsecase(db, repo, charReader, publisher, rooms)

	// Test 1: N=50 concurrent place requests từ 1 character chỉ có đủ coin mua 1 món (90 coins)
	n := 50
	var wg sync.WaitGroup
	wg.Add(n)

	successChan := make(chan *PlaceItemOutput, n)
	errChan := make(chan error, n)

	input := PlaceItemInput{
		ItemID:  itemID,
		MapCode: "village_adventure",
		X:       160,
		Y:       160,
	}

	for i := 0; i < n; i++ {
		go func(idx int) {
			defer wg.Done()
			req := input
			req.X = 16 * (idx + 1)
			res, err := uc.PlaceItem(ctx, userID, req)
			if err != nil {
				errChan <- err
			} else {
				successChan <- res
			}
		}(i)
	}

	wg.Wait()
	close(successChan)
	close(errChan)

	var successes []*PlaceItemOutput
	for s := range successChan {
		successes = append(successes, s)
	}

	var errorsList []error
	for e := range errChan {
		errorsList = append(errorsList, e)
	}

	if len(successes) != 1 {
		t.Errorf("Expected exactly 1 successful placement due to coin guard, got %d successes and %d errors", len(successes), len(errorsList))
	}

	// Trigger flush to DB
	uc.rooms.Shutdown()

	// Coin không được âm
	var finalCoins int
	err = db.QueryRowContext(ctx, "SELECT coins FROM characters WHERE id = $1", charID).Scan(&finalCoins)
	if err != nil {
		t.Fatalf("Failed to query final coins: %v", err)
	}

	if finalCoins < 0 {
		t.Errorf("Coins dropped below zero: %d", finalCoins)
	}

	if finalCoins != 0 {
		t.Errorf("Expected final coins to be 0, got %d", finalCoins)
	}

	// Dọn dẹp map_placements
	db.ExecContext(ctx, "DELETE FROM map_placements WHERE character_id = $1", charID)
}

func TestPlaceItem_DuplicateTile_Concurrently(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// 1. Tạo 2 app_users
	var userID1, userID2 string
	err := db.QueryRowContext(ctx, "INSERT INTO app_user (full_name, email, role) VALUES ('Test User 1', 'test1_dup@example.com', 'User') RETURNING id::text").Scan(&userID1)
	if err != nil {
		t.Fatalf("Failed to create user 1: %v", err)
	}
	defer db.ExecContext(ctx, "DELETE FROM app_user WHERE id = $1", userID1)

	err = db.QueryRowContext(ctx, "INSERT INTO app_user (full_name, email, role) VALUES ('Test User 2', 'test2_dup@example.com', 'User') RETURNING id::text").Scan(&userID2)
	if err != nil {
		t.Fatalf("Failed to create user 2: %v", err)
	}
	defer db.ExecContext(ctx, "DELETE FROM app_user WHERE id = $1", userID2)

	var mapID string
	err = db.QueryRowContext(ctx, "SELECT id::text FROM maps WHERE code = 'village_adventure'").Scan(&mapID)
	if err != nil {
		t.Fatalf("Failed to get map village_adventure: %v", err)
	}

	// 2. Tạo 2 characters với đủ coins (1000 coins mỗi char)
	var charID1, charID2 string
	err = db.QueryRowContext(ctx, "INSERT INTO characters (user_id, name, map_id, base_asset_key, coins) VALUES ($1, 'Test Character 1', $2, 'sprite-1', 1000) RETURNING id::text", userID1, mapID).Scan(&charID1)
	if err != nil {
		t.Fatalf("Failed to create character 1: %v", err)
	}
	defer db.ExecContext(ctx, "DELETE FROM characters WHERE id = $1", charID1)

	err = db.QueryRowContext(ctx, "INSERT INTO characters (user_id, name, map_id, base_asset_key, coins) VALUES ($1, 'Test Character 2', $2, 'sprite-2', 1000) RETURNING id::text", userID2, mapID).Scan(&charID2)
	if err != nil {
		t.Fatalf("Failed to create character 2: %v", err)
	}
	defer db.ExecContext(ctx, "DELETE FROM characters WHERE id = $1", charID2)

	var itemID string
	err = db.QueryRowContext(ctx, "SELECT id::text FROM items WHERE code = 'deco_chest'").Scan(&itemID)
	if err != nil {
		t.Fatalf("Failed to get item deco_chest: %v", err)
	}

	repo := repository.NewEditorRepository(db)
	charReader1 := &mockCharReader{charID: charID1, db: db}
	charReader2 := &mockCharReader{charID: charID2, db: db}
	publisher := &mockRoomPublisher{}

	rooms1 := room.NewRoomManager(db, publisher, repo, charReader1)
	uc1 := NewEditorUsecase(db, repo, charReader1, publisher, rooms1)
	uc2 := NewEditorUsecase(db, repo, charReader2, publisher, rooms1) // Share the same room manager!

	// Test 2: 2 request place vào cùng 1 ô (map, x, y) đồng thời. Chỉ được đúng 1 thành công.
	var wg sync.WaitGroup
	wg.Add(2)

	successChan := make(chan string, 2)
	errChan := make(chan error, 2)

	input := PlaceItemInput{
		ItemID:  itemID,
		MapCode: "village_adventure",
		X:       160,
		Y:       160,
	}

	go func() {
		defer wg.Done()
		res, err := uc1.PlaceItem(ctx, userID1, input)
		if err != nil {
			errChan <- err
		} else {
			successChan <- res.Placement.ID
		}
	}()

	go func() {
		defer wg.Done()
		res, err := uc2.PlaceItem(ctx, userID2, input)
		if err != nil {
			errChan <- err
		} else {
			successChan <- res.Placement.ID
		}
	}()

	wg.Wait()
	close(successChan)
	close(errChan)

	var successes []string
	for s := range successChan {
		successes = append(successes, s)
	}

	var errorsList []error
	for e := range errChan {
		errorsList = append(errorsList, e)
	}

	if len(successes) != 1 {
		t.Errorf("Expected exactly 1 successful placement on same coordinate, got %d successes and %d errors", len(successes), len(errorsList))
	}

	// Trigger flush to DB
	uc1.rooms.Shutdown()

	// Dọn dẹp map_placements
	db.ExecContext(ctx, "DELETE FROM map_placements WHERE character_id IN ($1, $2)", charID1, charID2)
}

func TestDeletePlacement_DoubleClick_Concurrently(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// 1. Tạo user và character với 0 coins
	var userID string
	err := db.QueryRowContext(ctx, "INSERT INTO app_user (full_name, email, role) VALUES ('Test User', 'test_del@example.com', 'User') RETURNING id::text").Scan(&userID)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	defer db.ExecContext(ctx, "DELETE FROM app_user WHERE id = $1", userID)

	var mapID string
	err = db.QueryRowContext(ctx, "SELECT id::text FROM maps WHERE code = 'village_adventure'").Scan(&mapID)
	if err != nil {
		t.Fatalf("Failed to get map village_adventure: %v", err)
	}

	var charID string
	err = db.QueryRowContext(ctx, "INSERT INTO characters (user_id, name, map_id, base_asset_key, coins) VALUES ($1, 'Test Character', $2, 'sprite-1', 0) RETURNING id::text", userID, mapID).Scan(&charID)
	if err != nil {
		t.Fatalf("Failed to create character: %v", err)
	}
	defer db.ExecContext(ctx, "DELETE FROM characters WHERE id = $1", charID)

	var itemID string
	var itemPrice int
	err = db.QueryRowContext(ctx, "SELECT id::text, price FROM items WHERE code = 'deco_chest'").Scan(&itemID, &itemPrice)
	if err != nil {
		t.Fatalf("Failed to get item deco_chest: %v", err)
	}

	// 2. Tạo sẵn placement trong DB
	placementID := uuid.NewString()
	_, err = db.ExecContext(ctx, "INSERT INTO map_placements (id, map_id, character_id, item_id, x, y) VALUES ($1, $2, $3, $4, 160, 160)", placementID, mapID, charID, itemID)
	if err != nil {
		t.Fatalf("Failed to insert placement: %v", err)
	}
	defer db.ExecContext(ctx, "DELETE FROM map_placements WHERE id = $1", placementID)

	repo := repository.NewEditorRepository(db)
	charReader := &mockCharReader{charID: charID, db: db}
	publisher := &mockRoomPublisher{}

	rooms := room.NewRoomManager(db, publisher, repo, charReader)
	uc := NewEditorUsecase(db, repo, charReader, publisher, rooms)

	// Test 3: Double click delete cùng placement concurrently. Chỉ được refund 1 lần.
	var wg sync.WaitGroup
	wg.Add(2)

	coinsChan := make(chan int, 2)
	errChan := make(chan error, 2)

	go func() {
		defer wg.Done()
		coins, err := uc.DeletePlacement(ctx, userID, "village_adventure", placementID)
		if err != nil {
			errChan <- err
		} else {
			coinsChan <- coins
		}
	}()

	go func() {
		defer wg.Done()
		coins, err := uc.DeletePlacement(ctx, userID, "village_adventure", placementID)
		if err != nil {
			errChan <- err
		} else {
			coinsChan <- coins
		}
	}()

	wg.Wait()
	close(coinsChan)
	close(errChan)

	var successes []int
	for c := range coinsChan {
		successes = append(successes, c)
	}

	var errorsList []error
	for e := range errChan {
		errorsList = append(errorsList, e)
	}

	if len(successes) != 1 {
		t.Errorf("Expected exactly 1 successful delete, got %d successes and %d errors", len(successes), len(errorsList))
	}

	// Trigger flush to DB
	uc.rooms.Shutdown()

	var finalCoins int
	err = db.QueryRowContext(ctx, "SELECT coins FROM characters WHERE id = $1", charID).Scan(&finalCoins)
	if err != nil {
		t.Fatalf("Failed to query final coins: %v", err)
	}

	if finalCoins != itemPrice {
		t.Errorf("Expected refund of exactly 1 item price (%d), got final coins: %d", itemPrice, finalCoins)
	}
}

type mockCharReader struct {
	charID string
	db     *sql.DB
}

func (m *mockCharReader) GetByUserID(ctx context.Context, userID string) (*port.CharacterInfo, error) {
	var coins int
	err := m.db.QueryRowContext(ctx, "SELECT coins FROM characters WHERE id = $1", m.charID).Scan(&coins)
	if err != nil {
		return nil, err
	}
	return &port.CharacterInfo{
		ID:    m.charID,
		Coins: coins,
	}, nil
}

func (m *mockCharReader) GetCoins(ctx context.Context, characterID string) (int, error) {
	var coins int
	err := m.db.QueryRowContext(ctx, "SELECT coins FROM characters WHERE id = $1", characterID).Scan(&coins)
	return coins, err
}

type mockRoomPublisher struct {
	mu     sync.Mutex
	events []map[string]any
}

func (m *mockRoomPublisher) PublishRoom(ctx context.Context, roomID string, data interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if evt, ok := data.(map[string]any); ok {
		m.events = append(m.events, evt)
	}
	return nil
}
