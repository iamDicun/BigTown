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
	CharacterID  string
	Name         string
	UserID       string
	ClientID     string
	BaseAssetKey string
	X            int
	Y            int
	Direction    Direction
	Moving       bool
	LastSeenAt   time.Time
}

type GameRoom struct {
	ID      string
	Players map[string]*RoomPlayer         // key: CharacterID
	Clients map[string]map[string]struct{} // key: CharacterID -> ClientID set
	// PlayersByUser là index phụ cho MovePlayer: đổi userID -> characterID hoàn toàn trong RAM,
	// không cần hỏi DB mỗi tick di chuyển. Được ghi lúc JoinRoom, xoá lúc LeaveRoom thực sự xoá
	// player khỏi room (không phải khi chỉ 1 trong nhiều ClientID rớt).
	PlayersByUser map[string]string // key: UserID -> CharacterID
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
