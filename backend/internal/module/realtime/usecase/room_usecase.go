package usecase

import (
	"context"
	"math"
	"time"

	"backend/internal/module/realtime/port"
	"backend/internal/module/realtime/room"
)

const (
	defaultCharacterName = "Player"

	// minDistancePx/tileSize khớp quyết định đã chốt trong docs/Realtime-Room-State-Decisions.md
	// mục 5 và docs/Movement-Chat-Spawn-Plan.md (pixel/free movement, minDistance 24px).
	minDistancePx = 24.0
	tileSize      = 16

	// maxSpeedPxPerSec có slack so với PLAYER_SPEED=120px/s ở FE (xem GameScene.ts) để chịu
	// jitter mạng/throttle 100ms, không phải giới hạn gameplay thật.
	maxSpeedPxPerSec = 400.0

	spawnSearchStep     = 8
	spawnSearchMaxRings = 24
)

// RoomUsecase validate movement/join/leave trước khi RoomStore RAM được cập nhật — xem
// docs/Realtime-Room-State-Decisions.md mục 6 (server authoritative movement).
type RoomUsecase struct {
	store      room.RoomStore
	characters port.CharacterResolver
	maps       port.MapReader
}

func NewRoomUsecase(store room.RoomStore, characters port.CharacterResolver, maps port.MapReader) *RoomUsecase {
	return &RoomUsecase{store: store, characters: characters, maps: maps}
}

// MovementRejection tương ứng event `player_position_correction` — xem
// docs/Realtime-Room-State-Decisions.md mục 6 (payload ví dụ có kèm characterId).
type MovementRejection struct {
	CharacterID string
	Reason      string
	X           int
	Y           int
}

// DefaultRoomID trả roomID của map mặc định hiện hành. MVP chỉ có 1 room/map tại một thời điểm
// (xem docs/Architecture.md mục 9.1) nên OnDisconnect có thể dùng thẳng giá trị này mà không cần
// track client đang subscribe channel nào.
func (u *RoomUsecase) DefaultRoomID(ctx context.Context) (string, error) {
	mapInfo, err := u.maps.GetDefaultMap(ctx)
	if err != nil {
		return "", err
	}
	return mapInfo.Code, nil
}

// JoinRoom trả về cả snapshot đầy đủ (để gửi room_snapshot cho chính client vừa join) lẫn player
// vừa join (để transport broadcast player_joined mà không phải tìm lại trong snapshot). clientID
// là Centrifuge connection ID (client.ID()) — lưu vào RoomPlayer.ClientID theo đúng model đã chốt
// trong docs/Realtime-Room-State-Decisions.md mục 3.
func (u *RoomUsecase) JoinRoom(ctx context.Context, roomID string, userID string, clientID string) (*room.RoomSnapshot, *room.RoomPlayer, error) {
	character, err := u.characters.GetOrCreateForUser(ctx, userID, defaultCharacterName)
	if err != nil {
		return nil, nil, err
	}

	mapInfo, err := u.maps.GetDefaultMap(ctx)
	if err != nil {
		return nil, nil, err
	}

	existing, err := u.store.GetSnapshot(ctx, roomID)
	if err != nil {
		return nil, nil, err
	}

	spawnX, spawnY := resolveSpawnPosition(mapInfo.SpawnX, mapInfo.SpawnY, existing)

	player := room.RoomPlayer{
		CharacterID: character.ID,
		ClientID:    clientID,
		X:           spawnX,
		Y:           spawnY,
		Direction:   room.DirectionDown,
		Moving:      false,
	}

	snapshot, err := u.store.JoinRoom(ctx, roomID, player)
	if err != nil {
		return nil, nil, err
	}

	return snapshot, &player, nil
}

// LeaveRoom trả lại player vừa rời đi (để transport broadcast `player_left`); trả nil, nil nếu
// user chưa từng join room đó (không phải lỗi).
func (u *RoomUsecase) LeaveRoom(ctx context.Context, roomID string, userID string) (*room.RoomPlayer, error) {
	character, err := u.characters.GetOrCreateForUser(ctx, userID, defaultCharacterName)
	if err != nil {
		return nil, err
	}

	player, err := u.store.GetPlayer(ctx, roomID, character.ID)
	if err != nil {
		return nil, nil
	}

	if err := u.store.LeaveRoom(ctx, roomID, character.ID); err != nil {
		return nil, err
	}

	return player, nil
}

// MovePlayer validate: user đúng chủ character, character đang trong room, tốc độ hợp lý, trong
// map bounds, không trùng vị trí player khác (minDistancePx) — xem
// docs/Realtime-Room-State-Decisions.md mục 6.
func (u *RoomUsecase) MovePlayer(ctx context.Context, roomID string, userID string, movement room.PlayerMovement) (*room.RoomPlayer, *MovementRejection, error) {
	character, err := u.characters.GetOrCreateForUser(ctx, userID, defaultCharacterName)
	if err != nil {
		return nil, nil, err
	}

	current, err := u.store.GetPlayer(ctx, roomID, character.ID)
	if err != nil {
		return nil, &MovementRejection{CharacterID: character.ID, Reason: "not_joined"}, nil
	}

	mapInfo, err := u.maps.GetDefaultMap(ctx)
	if err != nil {
		return nil, nil, err
	}

	maxX := mapInfo.Width*tileSize - 1
	maxY := mapInfo.Height*tileSize - 1
	if movement.X < 0 || movement.Y < 0 || movement.X > maxX || movement.Y > maxY {
		return nil, &MovementRejection{CharacterID: character.ID, Reason: "out_of_bounds", X: current.X, Y: current.Y}, nil
	}

	elapsed := time.Since(current.LastSeenAt).Seconds()
	if elapsed < 0.01 {
		elapsed = 0.01
	}
	if distance(current.X, current.Y, movement.X, movement.Y) > maxSpeedPxPerSec*elapsed {
		return nil, &MovementRejection{CharacterID: character.ID, Reason: "too_fast", X: current.X, Y: current.Y}, nil
	}

	snapshot, err := u.store.GetSnapshot(ctx, roomID)
	if err != nil {
		return nil, nil, err
	}
	for _, other := range snapshot.Players {
		if other.CharacterID == character.ID {
			continue
		}
		if distance(movement.X, movement.Y, other.X, other.Y) < minDistancePx {
			return nil, &MovementRejection{CharacterID: character.ID, Reason: "occupied", X: current.X, Y: current.Y}, nil
		}
	}

	updated, err := u.store.MovePlayer(ctx, roomID, character.ID, movement)
	if err != nil {
		return nil, nil, err
	}

	return updated, nil, nil
}

func distance(x1, y1, x2, y2 int) float64 {
	dx := float64(x1 - x2)
	dy := float64(y1 - y2)
	return math.Sqrt(dx*dx + dy*dy)
}

// resolveSpawnPosition né vị trí đã có player khác đứng (minDistancePx) bằng cách dò vòng xoắn ốc
// quanh spawn point mặc định của map — xem docs/Movement-Chat-Spawn-Plan.md mục 0.1 (nhiều player
// join cùng lúc không được đè lên nhau).
func resolveSpawnPosition(spawnX, spawnY int, existing *room.RoomSnapshot) (int, int) {
	if existing == nil || !isOccupied(spawnX, spawnY, existing.Players) {
		return spawnX, spawnY
	}

	for ring := 1; ring <= spawnSearchMaxRings; ring++ {
		offset := ring * spawnSearchStep
		candidates := [][2]int{
			{spawnX + offset, spawnY},
			{spawnX - offset, spawnY},
			{spawnX, spawnY + offset},
			{spawnX, spawnY - offset},
			{spawnX + offset, spawnY + offset},
			{spawnX - offset, spawnY - offset},
			{spawnX + offset, spawnY - offset},
			{spawnX - offset, spawnY + offset},
		}
		for _, c := range candidates {
			if !isOccupied(c[0], c[1], existing.Players) {
				return c[0], c[1]
			}
		}
	}

	return spawnX, spawnY
}

func isOccupied(x, y int, players []room.RoomPlayer) bool {
	for _, p := range players {
		if distance(x, y, p.X, p.Y) < minDistancePx {
			return true
		}
	}
	return false
}
