package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	characterentity "backend/internal/module/character/entity"
	"backend/internal/module/realtime/port"
	"backend/internal/module/realtime/room"
)

type mockCharacterResolver struct {
	getByUserIDFn func(ctx context.Context, userID string) (*characterentity.Character, error)
}

func (m *mockCharacterResolver) GetByUserID(ctx context.Context, userID string) (*characterentity.Character, error) {
	if m.getByUserIDFn != nil {
		return m.getByUserIDFn(ctx, userID)
	}
	return &characterentity.Character{
		ID: "c1", UserID: userID, Name: "Realtime Player", BaseAssetKey: "player", Coins: 500,
	}, nil
}

var _ port.CharacterResolver = (*mockCharacterResolver)(nil)

type mockMapReader struct {
	getDefaultMapFn func(ctx context.Context) (*characterentity.MapInfo, error)
	getMapByCodeFn  func(ctx context.Context, code string) (*characterentity.MapInfo, error)
}

func (m *mockMapReader) GetDefaultMap(ctx context.Context) (*characterentity.MapInfo, error) {
	if m.getDefaultMapFn != nil {
		return m.getDefaultMapFn(ctx)
	}
	return &characterentity.MapInfo{ID: "m1", Code: "main-map", Width: 100, Height: 100, TileSize: 32, SpawnX: 50, SpawnY: 50}, nil
}

func (m *mockMapReader) GetMapByCode(ctx context.Context, code string) (*characterentity.MapInfo, error) {
	if m.getMapByCodeFn != nil {
		return m.getMapByCodeFn(ctx, code)
	}
	if code == "main-map" {
		return &characterentity.MapInfo{ID: "m1", Code: "main-map", Width: 100, Height: 100, TileSize: 32, SpawnX: 50, SpawnY: 50}, nil
	}
	return nil, errors.New("map not found")
}

var _ port.MapReader = (*mockMapReader)(nil)

func TestDefaultRoomID(t *testing.T) {
	uc := NewRoomUsecase(room.NewActorRoomStore(), &mockCharacterResolver{}, &mockMapReader{})
	roomID, err := uc.DefaultRoomID(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if roomID != "main-map" {
		t.Errorf("expected main-map, got %s", roomID)
	}
}

func TestJoinAndLeaveRoom(t *testing.T) {
	store := room.NewActorRoomStore()

	uc := NewRoomUsecase(store, &mockCharacterResolver{}, &mockMapReader{})

	// Join
	snap, joined, isFirst, err := uc.JoinRoom(context.Background(), "main-map", "user-1", "client-1")
	if err != nil {
		t.Fatalf("unexpected join error: %v", err)
	}
	if snap == nil || joined == nil {
		t.Fatal("expected non-nil snapshot and joined player")
	}
	if !isFirst {
		t.Error("expected isFirst connection")
	}

	// Leave
	left, err := uc.LeaveRoom(context.Background(), "main-map", "user-1", "client-1")
	if err != nil {
		t.Fatalf("unexpected leave error: %v", err)
	}
	if left == nil {
		t.Error("expected left player info")
	}
}

func TestMovePlayer_Valid(t *testing.T) {
	store := room.NewActorRoomStore()

	uc := NewRoomUsecase(store, &mockCharacterResolver{}, &mockMapReader{})
	_, _, _, _ = uc.JoinRoom(context.Background(), "main-map", "user-1", "client-1")

	// Small valid move (50,50 -> 55,55)
	move := room.PlayerMovement{X: 55, Y: 55, Direction: room.DirectionDown, Moving: true}
	p, rej, err := uc.MovePlayer(context.Background(), "main-map", "user-1", move)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rej != nil {
		t.Fatalf("unexpected rejection: %s", rej.Reason)
	}
	if p.X != 55 || p.Y != 55 {
		t.Errorf("expected position (55,55), got (%d,%d)", p.X, p.Y)
	}
}

func TestMovePlayer_OutOfBounds(t *testing.T) {
	store := room.NewActorRoomStore()

	uc := NewRoomUsecase(store, &mockCharacterResolver{}, &mockMapReader{})
	_, _, _, _ = uc.JoinRoom(context.Background(), "main-map", "user-1", "client-1")

	// Out of bounds move (negative coordinates)
	move := room.PlayerMovement{X: -10, Y: -10, Direction: room.DirectionUp, Moving: true}
	_, rej, err := uc.MovePlayer(context.Background(), "main-map", "user-1", move)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rej == nil {
		t.Fatal("expected rejection for out of bounds move")
	}
}

func TestMovePlayer_TooFastAntiCheat(t *testing.T) {
	store := room.NewActorRoomStore()

	uc := NewRoomUsecase(store, &mockCharacterResolver{}, &mockMapReader{})
	_, _, _, _ = uc.JoinRoom(context.Background(), "main-map", "user-1", "client-1")

	// Teleport speed hack attempt (50,50 -> 2000, 2000 in 0ms)
	move := room.PlayerMovement{X: 2000, Y: 2000, Direction: room.DirectionRight, Moving: true}
	_, rej, err := uc.MovePlayer(context.Background(), "main-map", "user-1", move)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rej == nil {
		t.Fatal("expected rejection for speed limit violation (anti-cheat)")
	}
}

func TestMovePlayer_TooCloseProximity(t *testing.T) {
	store := room.NewActorRoomStore()

	resolver := &mockCharacterResolver{
		getByUserIDFn: func(ctx context.Context, userID string) (*characterentity.Character, error) {
			if userID == "u1" {
				return &characterentity.Character{ID: "c1", UserID: "u1", Name: "P1"}, nil
			}
			return &characterentity.Character{ID: "c2", UserID: "u2", Name: "P2"}, nil
		},
	}

	uc := NewRoomUsecase(store, resolver, &mockMapReader{})

	// Player 1 at (100, 100)
	_, _, _, _ = uc.JoinRoom(context.Background(), "main-map", "u1", "client-1")
	_, _ = store.MovePlayer(context.Background(), "main-map", "u1", room.PlayerMovement{X: 100, Y: 100})

	// Player 2 joins at (50, 50)
	_, _, _, _ = uc.JoinRoom(context.Background(), "main-map", "u2", "client-2")
	_, _ = store.MovePlayer(context.Background(), "main-map", "u2", room.PlayerMovement{X: 50, Y: 50})

	// Player 2 tries to move directly on top of Player 1 (105, 105 - distance < 24px)
	time.Sleep(10 * time.Millisecond)
	move := room.PlayerMovement{X: 105, Y: 105, Direction: room.DirectionDown, Moving: true}
	_, rej, err := uc.MovePlayer(context.Background(), "main-map", "u2", move)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rej == nil {
		t.Fatal("expected rejection when moving too close (< 24px) to another player")
	}
}
