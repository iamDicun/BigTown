package room

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestRoomStore_Parity(t *testing.T) {
	ctx := context.Background()
	roomID := "parity-room-1"

	p1 := RoomPlayer{
		CharacterID:  "char-1",
		Name:         "Player One",
		UserID:       "user-1",
		ClientID:     "client-1a",
		BaseAssetKey: "sprite-1",
		X:            100,
		Y:            200,
		Direction:    DirectionDown,
		Moving:       false,
	}

	p2 := RoomPlayer{
		CharacterID:  "char-2",
		Name:         "Player Two",
		UserID:       "user-2",
		ClientID:     "client-2a",
		BaseAssetKey: "sprite-2",
		X:            150,
		Y:            250,
		Direction:    DirectionUp,
		Moving:       true,
	}

	// Helper to run a sequence of operations and return results
	runSequence := func(s RoomStore) (snap1, snap2 *RoomSnapshot, jp1, jp2 *RoomPlayer, isFirst1, isFirst2 bool, gp1, gp2 *RoomPlayer, movePlayer *RoomPlayer, snap3 *RoomSnapshot, leftPlayer *RoomPlayer, wasRemoved bool, err error) {
		// 1. Join p1
		snap1, jp1, isFirst1, err = s.JoinRoom(ctx, roomID, p1)
		if err != nil {
			return
		}
		// 2. Join p2
		snap2, jp2, isFirst2, err = s.JoinRoom(ctx, roomID, p2)
		if err != nil {
			return
		}
		// 3. Get p1
		gp1, err = s.GetPlayer(ctx, roomID, p1.CharacterID)
		if err != nil {
			return
		}
		// 4. Get p2 by UserID
		gp2, err = s.GetPlayerByUserID(ctx, roomID, p2.UserID)
		if err != nil {
			return
		}
		// 5. Move p1
		movePlayer, err = s.MovePlayer(ctx, roomID, p1.CharacterID, PlayerMovement{
			X:         300,
			Y:         400,
			Direction: DirectionRight,
			Moving:    true,
		})
		if err != nil {
			return
		}
		// 6. Get Snapshot
		snap3, err = s.GetSnapshot(ctx, roomID)
		if err != nil {
			return
		}
		// 7. Leave p2
		leftPlayer, wasRemoved, err = s.LeaveRoom(ctx, roomID, p2.CharacterID, p2.ClientID)
		return
	}

	memStore := NewMemoryRoomStore()
	actStore := NewActorRoomStore()

	// Run on MemoryRoomStore
	msnap1, msnap2, mjp1, mjp2, mis1, mis2, mgp1, mgp2, mmove, msnap3, mleft, mrem, merr := runSequence(memStore)
	if merr != nil {
		t.Fatalf("MemoryRoomStore failed: %v", merr)
	}

	// Run on ActorRoomStore
	asnap1, asnap2, ajp1, ajp2, ais1, ais2, agp1, agp2, amove, asnap3, aleft, arem, aerr := runSequence(actStore)
	if aerr != nil {
		t.Fatalf("ActorRoomStore failed: %v", aerr)
	}

	// Helper to compare snapshot players ignoring LastSeenAt
	compareSnaps := func(t *testing.T, label string, s1, s2 *RoomSnapshot) {
		t.Helper()
		if s1.RoomID != s2.RoomID {
			t.Errorf("%s RoomID mismatch: %v vs %v", label, s1.RoomID, s2.RoomID)
		}
		if len(s1.Players) != len(s2.Players) {
			t.Errorf("%s Players len mismatch: %d vs %d", label, len(s1.Players), len(s2.Players))
			return
		}
		// Create maps for order-independent comparison
		m1 := make(map[string]RoomPlayer)
		m2 := make(map[string]RoomPlayer)
		for _, p := range s1.Players {
			p.LastSeenAt = time.Time{} // normalize time
			m1[p.CharacterID] = p
		}
		for _, p := range s2.Players {
			p.LastSeenAt = time.Time{} // normalize time
			m2[p.CharacterID] = p
		}
		if !reflect.DeepEqual(m1, m2) {
			t.Errorf("%s Players content mismatch: %+v vs %+v", label, m1, m2)
		}
	}

	// Helper to compare players ignoring LastSeenAt
	comparePlayers := func(t *testing.T, label string, p1, p2 *RoomPlayer) {
		t.Helper()
		cp1 := *p1
		cp2 := *p2
		cp1.LastSeenAt = time.Time{}
		cp2.LastSeenAt = time.Time{}
		if !reflect.DeepEqual(cp1, cp2) {
			t.Errorf("%s player mismatch: %+v vs %+v", label, cp1, cp2)
		}
	}

	// Assertions
	compareSnaps(t, "Join p1 snapshot", msnap1, asnap1)
	compareSnaps(t, "Join p2 snapshot", msnap2, asnap2)
	compareSnaps(t, "Move p1 snapshot", msnap3, asnap3)

	comparePlayers(t, "Join p1 return", mjp1, ajp1)
	comparePlayers(t, "Join p2 return", mjp2, ajp2)
	comparePlayers(t, "Get p1 return", mgp1, agp1)
	comparePlayers(t, "Get p2 by UserID return", mgp2, agp2)
	comparePlayers(t, "Move p1 return", mmove, amove)
	comparePlayers(t, "Leave p2 return", mleft, aleft)

	if mis1 != ais1 {
		t.Errorf("isFirstConnection 1 mismatch: %t vs %t", mis1, ais1)
	}
	if mis2 != ais2 {
		t.Errorf("isFirstConnection 2 mismatch: %t vs %t", mis2, ais2)
	}
	if mrem != arem {
		t.Errorf("wasRemoved mismatch: %t vs %t", mrem, arem)
	}
}

func TestActorRoomStore_Concurrency(t *testing.T) {
	ctx := context.Background()
	store := NewActorRoomStore()

	numRooms := 5
	numPlayersPerRoom := 20
	var wg sync.WaitGroup

	// Concurrently join players
	for r := 0; r < numRooms; r++ {
		roomID := fmt.Sprintf("concur-room-%d", r)
		for p := 0; p < numPlayersPerRoom; p++ {
			wg.Add(1)
			go func(roomID string, playerID int) {
				defer wg.Done()

				charID := fmt.Sprintf("char-%s-%d", roomID, playerID)
				userID := fmt.Sprintf("user-%s-%d", roomID, playerID)
				clientID := fmt.Sprintf("client-%s-%d", roomID, playerID)

				player := RoomPlayer{
					CharacterID:  charID,
					Name:         fmt.Sprintf("Player %d", playerID),
					UserID:       userID,
					ClientID:     clientID,
					BaseAssetKey: "sprite",
					X:            0,
					Y:            0,
					Direction:    DirectionDown,
					Moving:       false,
				}

				// Join Room
				_, _, _, err := store.JoinRoom(ctx, roomID, player)
				if err != nil {
					t.Errorf("JoinRoom failed: %v", err)
					return
				}

				// Move Player
				_, err = store.MovePlayer(ctx, roomID, charID, PlayerMovement{
					X:         10 * playerID,
					Y:         10 * playerID,
					Direction: DirectionRight,
					Moving:    true,
				})
				if err != nil {
					t.Errorf("MovePlayer failed: %v", err)
					return
				}

				// Get Player
				gp, err := store.GetPlayer(ctx, roomID, charID)
				if err != nil {
					t.Errorf("GetPlayer failed: %v", err)
					return
				}
				if gp.X != 10*playerID {
					t.Errorf("Expected player X to be %d, got %d", 10*playerID, gp.X)
				}
			}(roomID, p)
		}
	}

	wg.Wait()

	// Verify snapshots at the end
	for r := 0; r < numRooms; r++ {
		roomID := fmt.Sprintf("concur-room-%d", r)
		snap, err := store.GetSnapshot(ctx, roomID)
		if err != nil {
			t.Fatalf("GetSnapshot failed: %v", err)
		}
		if len(snap.Players) != numPlayersPerRoom {
			t.Errorf("Room %s has %d players, expected %d", roomID, len(snap.Players), numPlayersPerRoom)
		}
	}
}
