package usecase

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"backend/internal/module/character/entity"
	"backend/internal/module/character/port"
	userentity "backend/internal/module/user/entity"
)

type mockCharacterRepo struct {
	findByUserIDFn           func(ctx context.Context, userID string) (*entity.Character, error)
	createWithTxFn           func(ctx context.Context, tx *sql.Tx, userID string, name string, baseAssetKey string) (*entity.Character, error)
	findMapByCodeFn          func(ctx context.Context, code string) (*entity.MapInfo, error)
	syncMapIDFn              func(ctx context.Context, characterID string, currentMapID *string) (*string, error)
	findNPCSpawnsByMapCodeFn func(ctx context.Context, mapCode string) ([]entity.NPCSpawn, error)
}

func (m *mockCharacterRepo) FindByUserID(ctx context.Context, userID string) (*entity.Character, error) {
	if m.findByUserIDFn != nil {
		return m.findByUserIDFn(ctx, userID)
	}
	return nil, sql.ErrNoRows
}
func (m *mockCharacterRepo) CreateWithTx(ctx context.Context, tx *sql.Tx, userID string, name string, baseAssetKey string) (*entity.Character, error) {
	if m.createWithTxFn != nil {
		return m.createWithTxFn(ctx, tx, userID, name, baseAssetKey)
	}
	return nil, nil
}
func (m *mockCharacterRepo) FindMapByCode(ctx context.Context, code string) (*entity.MapInfo, error) {
	if m.findMapByCodeFn != nil {
		return m.findMapByCodeFn(ctx, code)
	}
	return nil, sql.ErrNoRows
}
func (m *mockCharacterRepo) SyncMapID(ctx context.Context, characterID string, currentMapID *string) (*string, error) {
	if m.syncMapIDFn != nil {
		return m.syncMapIDFn(ctx, characterID, currentMapID)
	}
	return currentMapID, nil
}
func (m *mockCharacterRepo) FindNPCSpawnsByMapCode(ctx context.Context, mapCode string) ([]entity.NPCSpawn, error) {
	if m.findNPCSpawnsByMapCodeFn != nil {
		return m.findNPCSpawnsByMapCodeFn(ctx, mapCode)
	}
	return []entity.NPCSpawn{}, nil
}

var _ port.CharacterRepository = (*mockCharacterRepo)(nil)

type mockUserReader struct{}

func (m *mockUserReader) FindByID(ctx context.Context, id string) (*userentity.User, error) {
	return &userentity.User{ID: id}, nil
}

func TestListOptions(t *testing.T) {
	uc := NewCharacterUsecase(nil, &mockCharacterRepo{}, &mockUserReader{}, "map-1")
	opts := uc.ListOptions()
	if len(opts) == 0 {
		t.Fatal("expected non-empty character options")
	}
}

func TestGetByUserID_NotFound(t *testing.T) {
	repo := &mockCharacterRepo{
		findByUserIDFn: func(ctx context.Context, userID string) (*entity.Character, error) {
			return nil, sql.ErrNoRows
		},
	}
	uc := NewCharacterUsecase(nil, repo, &mockUserReader{}, "map-1")

	_, err := uc.GetByUserID(context.Background(), "user-unknown")
	if err == nil {
		t.Fatal("expected error for non-existing character")
	}
}

func TestGetByUserID_SuccessAndCache(t *testing.T) {
	callCount := 0
	mapID := "map-uuid-1"
	repo := &mockCharacterRepo{
		findByUserIDFn: func(ctx context.Context, userID string) (*entity.Character, error) {
			callCount++
			return &entity.Character{ID: "char-1", UserID: userID, Name: "Hero", MapID: &mapID}, nil
		},
		syncMapIDFn: func(ctx context.Context, characterID string, currentMapID *string) (*string, error) {
			return currentMapID, nil
		},
	}
	uc := NewCharacterUsecase(nil, repo, &mockUserReader{}, "map-1")

	// Call 1: Load from DB
	char1, err := uc.GetByUserID(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if char1.Name != "Hero" {
		t.Errorf("expected Hero, got %s", char1.Name)
	}

	// Call 2: Should hit RAM Cache (callCount should stay 1)
	char2, err := uc.GetByUserID(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}
	if char2.ID != "char-1" {
		t.Errorf("expected char-1, got %s", char2.ID)
	}
	if callCount != 1 {
		t.Errorf("expected 1 DB call due to caching, got %d", callCount)
	}
}

func TestCreateForUser_ValidationErrors(t *testing.T) {
	uc := NewCharacterUsecase(nil, &mockCharacterRepo{}, &mockUserReader{}, "map-1")

	// Empty name
	_, err := uc.CreateForUser(context.Background(), "user-1", "   ", "player")
	if err == nil {
		t.Error("expected error for empty name")
	}

	// Invalid asset key
	_, err = uc.CreateForUser(context.Background(), "user-1", "Hero", "invalid_key")
	if err == nil {
		t.Error("expected error for invalid asset key")
	}
}

func TestCreateForUser_AlreadyExists(t *testing.T) {
	repo := &mockCharacterRepo{
		findByUserIDFn: func(ctx context.Context, userID string) (*entity.Character, error) {
			return &entity.Character{ID: "existing-char"}, nil
		},
	}
	uc := NewCharacterUsecase(nil, repo, &mockUserReader{}, "map-1")

	_, err := uc.CreateForUser(context.Background(), "user-1", "Hero", "player")
	if err == nil {
		t.Error("expected error when user already has character")
	}
}

func TestCreateForUser_Success(t *testing.T) {
	repo := &mockCharacterRepo{
		findByUserIDFn: func(ctx context.Context, userID string) (*entity.Character, error) {
			return nil, sql.ErrNoRows
		},
		createWithTxFn: func(ctx context.Context, tx *sql.Tx, userID string, name string, baseAssetKey string) (*entity.Character, error) {
			return &entity.Character{ID: "new-char", UserID: userID, Name: name, BaseAssetKey: baseAssetKey}, nil
		},
	}

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectCommit()

	uc := NewCharacterUsecase(db, repo, &mockUserReader{}, "map-1")

	char, err := uc.CreateForUser(context.Background(), "user-1", "New Hero", "player")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if char.Name != "New Hero" {
		t.Errorf("expected New Hero, got %s", char.Name)
	}
}

func TestGetDefaultMap(t *testing.T) {
	repo := &mockCharacterRepo{
		findMapByCodeFn: func(ctx context.Context, code string) (*entity.MapInfo, error) {
			return &entity.MapInfo{ID: "map-1", Code: code, Name: "Default Town"}, nil
		},
	}
	uc := NewCharacterUsecase(nil, repo, &mockUserReader{}, "default-town")

	m, err := uc.GetDefaultMap(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Code != "default-town" {
		t.Errorf("expected default-town, got %s", m.Code)
	}
}
