package room

import (
	"context"
	"sync"
	"time"
)

var _ RoomStore = (*MemoryRoomStore)(nil)

// MemoryRoomStore là implementation RAM cho MVP — 1 process, không cần đồng bộ giữa nhiều node
// (xem docs/Architecture.md mục 10 về hướng scale sau này qua Redis broker + room ownership).
type MemoryRoomStore struct {
	mu    sync.Mutex
	rooms map[string]*GameRoom
}

func NewMemoryRoomStore() *MemoryRoomStore {
	return &MemoryRoomStore{rooms: make(map[string]*GameRoom)}
}

func (s *MemoryRoomStore) JoinRoom(ctx context.Context, roomID string, player RoomPlayer) (*RoomSnapshot, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	room := s.getOrCreateRoomLocked(roomID)
	player.LastSeenAt = time.Now()
	room.Players[player.CharacterID] = &player

	return snapshotLocked(room), nil
}

func (s *MemoryRoomStore) LeaveRoom(ctx context.Context, roomID string, characterID string) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	room, ok := s.rooms[roomID]
	if !ok {
		return nil
	}
	delete(room.Players, characterID)

	return nil
}

func (s *MemoryRoomStore) GetSnapshot(ctx context.Context, roomID string) (*RoomSnapshot, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	room, ok := s.rooms[roomID]
	if !ok {
		return &RoomSnapshot{RoomID: roomID}, nil
	}

	return snapshotLocked(room), nil
}

func (s *MemoryRoomStore) GetPlayer(ctx context.Context, roomID string, characterID string) (*RoomPlayer, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	room, ok := s.rooms[roomID]
	if !ok {
		return nil, ErrPlayerNotFound
	}
	player, ok := room.Players[characterID]
	if !ok {
		return nil, ErrPlayerNotFound
	}

	playerCopy := *player
	return &playerCopy, nil
}

func (s *MemoryRoomStore) MovePlayer(ctx context.Context, roomID string, characterID string, movement PlayerMovement) (*RoomPlayer, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	room, ok := s.rooms[roomID]
	if !ok {
		return nil, ErrPlayerNotFound
	}
	player, ok := room.Players[characterID]
	if !ok {
		return nil, ErrPlayerNotFound
	}

	player.X = movement.X
	player.Y = movement.Y
	player.Direction = movement.Direction
	player.Moving = movement.Moving
	player.LastSeenAt = time.Now()

	playerCopy := *player
	return &playerCopy, nil
}

func (s *MemoryRoomStore) getOrCreateRoomLocked(roomID string) *GameRoom {
	r, ok := s.rooms[roomID]
	if !ok {
		r = &GameRoom{ID: roomID, Players: make(map[string]*RoomPlayer)}
		s.rooms[roomID] = r
	}
	return r
}

func snapshotLocked(room *GameRoom) *RoomSnapshot {
	players := make([]RoomPlayer, 0, len(room.Players))
	for _, p := range room.Players {
		players = append(players, *p)
	}
	return &RoomSnapshot{RoomID: room.ID, Players: players}
}
