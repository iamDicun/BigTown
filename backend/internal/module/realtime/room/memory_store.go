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

func (s *MemoryRoomStore) JoinRoom(ctx context.Context, roomID string, player RoomPlayer) (*RoomSnapshot, *RoomPlayer, bool, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	room := s.getOrCreateRoomLocked(roomID)
	// Check-và-insert phải nằm trong cùng 1 lock: nếu 2 connection của cùng user (vd ChatPanel +
	// GameScene) gọi JoinRoom gần như đồng thời, người gọi sau phải thấy đúng player đã tồn tại
	// (không phải tự tính lại vị trí/spawn), và chỉ 1 trong 2 được coi là "lần đầu" — không dựa vào
	// GetSnapshot đọc trước đó ở tầng usecase vì giữa 2 lock đó có thể có connection khác chen vào.
	if existing, ok := room.Players[player.CharacterID]; ok {
		existing.LastSeenAt = time.Now()
		addClientLocked(room, player.CharacterID, player.ClientID)
		setUserIndexLocked(room, existing.UserID, player.CharacterID)
		existingCopy := *existing
		return snapshotLocked(room), &existingCopy, false, nil
	}

	player.LastSeenAt = time.Now()
	room.Players[player.CharacterID] = &player
	addClientLocked(room, player.CharacterID, player.ClientID)
	setUserIndexLocked(room, player.UserID, player.CharacterID)

	joinedCopy := player
	return snapshotLocked(room), &joinedCopy, true, nil
}

func (s *MemoryRoomStore) LeaveRoom(ctx context.Context, roomID string, characterID string, clientID string) (*RoomPlayer, bool, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	room, ok := s.rooms[roomID]
	if !ok {
		return nil, false, nil
	}
	player, ok := room.Players[characterID]
	if !ok {
		return nil, false, nil
	}

	if clients, ok := room.Clients[characterID]; ok {
		delete(clients, clientID)
		if len(clients) > 0 {
			playerCopy := *player
			return &playerCopy, false, nil
		}
	}

	playerCopy := *player
	delete(room.Players, characterID)
	delete(room.Clients, characterID)
	if room.PlayersByUser != nil {
		delete(room.PlayersByUser, player.UserID)
	}

	return &playerCopy, true, nil
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

func (s *MemoryRoomStore) GetPlayerByUserID(ctx context.Context, roomID string, userID string) (*RoomPlayer, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()

	room, ok := s.rooms[roomID]
	if !ok {
		return nil, ErrPlayerNotFound
	}
	characterID, ok := room.PlayersByUser[userID]
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
		r = &GameRoom{
			ID:            roomID,
			Players:       make(map[string]*RoomPlayer),
			Clients:       make(map[string]map[string]struct{}),
			PlayersByUser: make(map[string]string),
		}
		s.rooms[roomID] = r
	}
	return r
}

func setUserIndexLocked(room *GameRoom, userID string, characterID string) {
	if room.PlayersByUser == nil {
		room.PlayersByUser = make(map[string]string)
	}
	room.PlayersByUser[userID] = characterID
}

func addClientLocked(room *GameRoom, characterID string, clientID string) {
	if room.Clients == nil {
		room.Clients = make(map[string]map[string]struct{})
	}
	if room.Clients[characterID] == nil {
		room.Clients[characterID] = make(map[string]struct{})
	}
	room.Clients[characterID][clientID] = struct{}{}
}

func snapshotLocked(room *GameRoom) *RoomSnapshot {
	players := make([]RoomPlayer, 0, len(room.Players))
	for _, p := range room.Players {
		players = append(players, *p)
	}
	return &RoomSnapshot{RoomID: room.ID, Players: players}
}
