package room

import "time"

// Direction/RoomPlayer/GameRoom theo đúng model RAM đã chốt trong
// docs/Realtime-Room-State-Decisions.md mục 3. NPC runtime chưa có ở đây — enemy combat
// (npc_types/map_npc_spawns) là việc của phase sau (xem docs/Movement-Chat-Spawn-Plan.md mục I).
type Direction string

const (
	DirectionUp    Direction = "up"
	DirectionDown  Direction = "down"
	DirectionLeft  Direction = "left"
	DirectionRight Direction = "right"
)

type RoomPlayer struct {
	CharacterID string
	// ClientID là Centrifuge connection ID (client.ID()), không phải UserID — 1 user có thể mở
	// nhiều connection (nhiều tab), ClientID cho biết room-player entry này hiện thuộc connection
	// nào. Đúng theo model đã chốt trong docs/Realtime-Room-State-Decisions.md mục 3.
	ClientID   string
	X          int
	Y          int
	Direction  Direction
	Moving     bool
	LastSeenAt time.Time
}

type GameRoom struct {
	ID      string
	Players map[string]*RoomPlayer // key: CharacterID
}

type RoomSnapshot struct {
	RoomID  string
	Players []RoomPlayer
}

// PlayerMovement là proposed movement từ client, chưa qua validate — xem RoomUsecase.MovePlayer.
type PlayerMovement struct {
	X         int
	Y         int
	Direction Direction
	Moving    bool
}
