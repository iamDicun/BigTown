package room

import (
	"context"
	"testing"
)

func TestActorRoomStore_JoinAndGet(t *testing.T) {
	store := NewActorRoomStore()

	ctx := context.Background()
	player := RoomPlayer{
		CharacterID: "c1", UserID: "u1", Name: "Player 1", BaseAssetKey: "player", ClientID: "client-1", X: 100, Y: 200,
	}

	snap, joined, isFirst, err := store.JoinRoom(ctx, "room-1", player)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isFirst {
		t.Error("expected first player to be marked as new player in room")
	}
	if snap.RoomID != "room-1" {
		t.Errorf("expected room-1, got %s", snap.RoomID)
	}
	if joined.CharacterID != "c1" {
		t.Errorf("expected character c1, got %s", joined.CharacterID)
	}

	// Double check GetPlayer
	p, err := store.GetPlayer(ctx, "room-1", "c1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name != "Player 1" {
		t.Errorf("expected Player 1, got %s", p.Name)
	}
	if p.X != 100 || p.Y != 200 {
		t.Errorf("expected coords (100, 200), got (%d, %d)", p.X, p.Y)
	}
}

func TestActorRoomStore_MovePlayer(t *testing.T) {
	store := NewActorRoomStore()

	ctx := context.Background()
	player := RoomPlayer{CharacterID: "c1", UserID: "u1", Name: "Player 1", ClientID: "client-1", X: 10, Y: 10}
	_, _, _, _ = store.JoinRoom(ctx, "room-1", player)

	move := PlayerMovement{X: 50, Y: 60, Direction: DirectionRight, Moving: true}
	p, err := store.MovePlayer(ctx, "room-1", "c1", move)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.X != 50 || p.Y != 60 {
		t.Errorf("expected coords (50, 60), got (%d, %d)", p.X, p.Y)
	}
	if p.Direction != DirectionRight {
		t.Errorf("expected direction right, got %s", p.Direction)
	}
}

func TestActorRoomStore_MultipleClientsAndLeave(t *testing.T) {
	store := NewActorRoomStore()

	ctx := context.Background()
	player1 := RoomPlayer{CharacterID: "c1", UserID: "u1", Name: "Player 1", ClientID: "client-1"}
	player2 := RoomPlayer{CharacterID: "c1", UserID: "u1", Name: "Player 1", ClientID: "client-2"}

	// Client 1 joins
	_, _, isFirst1, _ := store.JoinRoom(ctx, "room-1", player1)
	if !isFirst1 {
		t.Error("expected first client to be new")
	}

	// Client 2 joins with same character (multi tab)
	_, _, isFirst2, _ := store.JoinRoom(ctx, "room-1", player2)
	if isFirst2 {
		t.Error("expected second client for same character to not be isFirst")
	}

	// Leave client 1 -> player still in room because client 2 is connected
	_, removed1, _ := store.LeaveRoom(ctx, "room-1", "c1", "client-1")
	if removed1 {
		t.Error("player should remain in room when client 2 is still active")
	}

	// Leave client 2 -> player removed
	_, removed2, _ := store.LeaveRoom(ctx, "room-1", "c1", "client-2")
	if !removed2 {
		t.Error("player should be removed when last client leaves")
	}
}
