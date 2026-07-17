package transport

import "backend/internal/module/realtime/room"

// Payload WebSocket theo đúng tên event đã chốt trong docs/Realtime-Room-State-Decisions.md mục 1
// và mục 6. Không dùng chung room.RoomPlayer làm wire format — tách DTO riêng theo
// docs/Phaser-Frontend-Guide.md mục 2 ("Không dùng chung một object cho DB entity, WebSocket
// event và Phaser sprite state").
type roomPlayerDTO struct {
	CharacterID string `json:"characterId"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
	Direction   string `json:"direction"`
	Moving      bool   `json:"moving"`
}

func toRoomPlayerDTO(p room.RoomPlayer) roomPlayerDTO {
	return roomPlayerDTO{
		CharacterID: p.CharacterID,
		X:           p.X,
		Y:           p.Y,
		Direction:   string(p.Direction),
		Moving:      p.Moving,
	}
}

func toRoomPlayerDTOs(players []room.RoomPlayer) []roomPlayerDTO {
	dtos := make([]roomPlayerDTO, 0, len(players))
	for _, p := range players {
		dtos = append(dtos, toRoomPlayerDTO(p))
	}
	return dtos
}

type roomSnapshotEvent struct {
	Type    string          `json:"type"`
	RoomID  string          `json:"roomId"`
	Players []roomPlayerDTO `json:"players"`
}

type playerJoinedEvent struct {
	Type   string        `json:"type"`
	RoomID string        `json:"roomId"`
	Player roomPlayerDTO `json:"player"`
}

type playerLeftEvent struct {
	Type        string `json:"type"`
	RoomID      string `json:"roomId"`
	CharacterID string `json:"characterId"`
}

type playerMoveEvent struct {
	Type        string `json:"type"`
	CharacterID string `json:"characterId"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
	Direction   string `json:"direction"`
	Moving      bool   `json:"moving"`
}

// playerMoveCommand là payload client gửi qua RPC method "player_move". Không nhận characterId
// từ client — server tự resolve từ UserID đã xác thực (xem RoomUsecase.MovePlayer).
type playerMoveCommand struct {
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Direction string `json:"direction"`
	Moving    bool   `json:"moving"`
}

// positionCorrectionEvent gửi riêng cho client qua personal channel khi movement bị reject —
// xem docs/Realtime-Room-State-Decisions.md mục 6 (payload ví dụ có characterId) và quyết định đã
// chốt (personal channel theo userID, không trả trong response RPC).
type positionCorrectionEvent struct {
	Type        string `json:"type"`
	CharacterID string `json:"characterId"`
	Reason      string `json:"reason"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
}
