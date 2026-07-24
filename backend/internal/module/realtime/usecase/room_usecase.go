package usecase

import (
	"context"
	"math"
	"time"

	"backend/internal/module/realtime/port"
	"backend/internal/module/realtime/room"
)

const (
	// minDistancePx khớp quyết định đã chốt trong docs/Realtime-Room-State-Decisions.md
	// mục 5 và docs/Movement-Chat-Spawn-Plan.md (pixel/free movement, minDistance 24px).
	minDistancePx = 24.0

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

// JoinRoom trả về cả snapshot đầy đủ (để gửi room_snapshot cho chính client vừa join), player thực
// sự đang có trong room sau lệnh này (để transport broadcast player_joined nếu cần) và
// isFirstConnection. clientID là Centrifuge connection ID (client.ID()) — lưu vào RoomPlayer.ClientID
// theo đúng model đã chốt trong docs/Realtime-Room-State-Decisions.md mục 3.
//
// spawnX/spawnY tính từ GetSnapshot đọc TRƯỚC khi gọi store.JoinRoom nên chỉ là "ứng viên" — nếu
// character này đã có connection khác giữ chỗ trong room (race giữa 2 connection join gần như đồng
// thời của cùng user), MemoryRoomStore.JoinRoom sẽ tự bỏ qua ứng viên này và giữ nguyên vị trí cũ,
// đồng thời trả isFirstConnection=false atomic — không phụ thuộc vào phép tính ở đây.
func (u *RoomUsecase) JoinRoom(ctx context.Context, roomID string, userID string, clientID string) (*room.RoomSnapshot, *room.RoomPlayer, bool, error) {
	character, err := u.characters.GetByUserID(ctx, userID)
	if err != nil {
		return nil, nil, false, err
	}

	mapInfo, err := u.maps.GetMapByCode(ctx, roomID)
	if err != nil {
		return nil, nil, false, err
	}

	existing, err := u.store.GetSnapshot(ctx, roomID)
	if err != nil {
		return nil, nil, false, err
	}

	spawnX, spawnY := resolveSpawnPosition(mapInfo.SpawnX, mapInfo.SpawnY, existing, character.ID)

	candidate := room.RoomPlayer{
		CharacterID:  character.ID,
		Name:         character.Name,
		UserID:       userID,
		ClientID:     clientID,
		BaseAssetKey: character.BaseAssetKey,
		X:            spawnX,
		Y:            spawnY,
		Direction:    room.DirectionDown,
		Moving:       false,
	}

	snapshot, joined, isFirstConnection, err := u.store.JoinRoom(ctx, roomID, candidate)
	if err != nil {
		return nil, nil, false, err
	}

	return snapshot, joined, isFirstConnection, nil
}

// LeaveRoom trả lại player vừa rời đi (để transport broadcast `player_left`); trả nil, nil nếu
// user chưa từng join room đó (không phải lỗi).
func (u *RoomUsecase) LeaveRoom(ctx context.Context, roomID string, userID string, clientID string) (*room.RoomPlayer, error) {
	character, err := u.characters.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	player, removed, err := u.store.LeaveRoom(ctx, roomID, character.ID, clientID)
	if err != nil {
		return nil, err
	}
	if !removed {
		return nil, nil
	}

	return player, nil
}

// MovePlayer validate: user đúng chủ character, character đang trong room, tốc độ hợp lý, trong
// map bounds, không trùng vị trí player khác (minDistancePx) — xem
// docs/Realtime-Room-State-Decisions.md mục 6.
//
// Chạy hoàn toàn trong RAM (GetPlayerByUserID + RoomStore), không gọi CharacterResolver (DB) —
// hàm này được gọi trên mỗi player_move RPC (10 lần/giây/player đang di chuyển), nên 1 round-trip
// DB mỗi tick sẽ dồn queue rất nhanh khi DB có độ trễ thật (khác hẳn lúc chạy local, DB cùng máy).
func (u *RoomUsecase) MovePlayer(ctx context.Context, roomID string, userID string, movement room.PlayerMovement) (*room.RoomPlayer, *MovementRejection, error) {
	current, err := u.store.GetPlayerByUserID(ctx, roomID, userID)
	if err != nil {
		return nil, &MovementRejection{Reason: "not_joined"}, nil
	}
	character := current.CharacterID

	mapInfo, err := u.maps.GetMapByCode(ctx, roomID)
	if err != nil {
		return nil, nil, err
	}

	maxX := mapInfo.MaxPixelX()
	maxY := mapInfo.MaxPixelY()
	if movement.X < 0 || movement.Y < 0 || movement.X > maxX || movement.Y > maxY {
		return nil, &MovementRejection{CharacterID: character, Reason: "out_of_bounds", X: current.X, Y: current.Y}, nil
	}

	elapsed := time.Since(current.LastSeenAt).Seconds()
	if elapsed < 0.01 {
		elapsed = 0.01
	}
	if distance(current.X, current.Y, movement.X, movement.Y) > maxSpeedPxPerSec*elapsed {
		return nil, &MovementRejection{CharacterID: character, Reason: "too_fast", X: current.X, Y: current.Y}, nil
	}

	snapshot, err := u.store.GetSnapshot(ctx, roomID)
	if err != nil {
		return nil, nil, err
	}
	for _, other := range snapshot.Players {
		if other.CharacterID == character {
			continue
		}
		if distance(movement.X, movement.Y, other.X, other.Y) < minDistancePx {
			return nil, &MovementRejection{CharacterID: character, Reason: "occupied", X: current.X, Y: current.Y}, nil
		}
	}

	updated, err := u.store.MovePlayer(ctx, roomID, character, movement)
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
// join cùng lúc không được đè lên nhau). excludeCharacterID loại trừ chính character đang join khỏi
// occupancy check — nếu character này đã có 1 entry trong snapshot (do 1 connection khác của cùng
// user join trước), không được coi bản thân mình là "đang chiếm chỗ" rồi tự dịch đi nơi khác.
func resolveSpawnPosition(spawnX, spawnY int, existing *room.RoomSnapshot, excludeCharacterID string) (int, int) {
	if existing == nil || !isOccupied(spawnX, spawnY, existing.Players, excludeCharacterID) {
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
			if !isOccupied(c[0], c[1], existing.Players, excludeCharacterID) {
				return c[0], c[1]
			}
		}
	}

	return spawnX, spawnY
}

func isOccupied(x, y int, players []room.RoomPlayer, excludeCharacterID string) bool {
	for _, p := range players {
		if p.CharacterID == excludeCharacterID {
			continue
		}
		if distance(x, y, p.X, p.Y) < minDistancePx {
			return true
		}
	}
	return false
}

type WarpDestination struct {
	MapCode string
	X       int
	Y       int
}

func (u *RoomUsecase) WarpPlayer(ctx context.Context, roomID string, userID string, destMap string, destX int, destY int) (*WarpDestination, error) {
	_, err := u.maps.GetMapByCode(ctx, destMap)
	if err != nil {
		return nil, err
	}

	character, err := u.characters.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if _, _, err := u.store.LeaveRoom(ctx, roomID, character.ID, ""); err != nil {
		return nil, err
	}

	return &WarpDestination{MapCode: destMap, X: destX, Y: destY}, nil
}
