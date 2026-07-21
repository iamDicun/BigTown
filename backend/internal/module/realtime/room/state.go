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
	// Name là display name của character (từ bảng characters), set 1 lần lúc JoinRoom — dùng để
	// hiển thị tên trên đầu nhân vật ở FE (xem transport/events.go roomPlayerDTO.Name).
	Name string
	// UserID dùng để tra cứu RAM O(1) theo userID (xem GameRoom.PlayersByUser) — tránh phải gọi
	// CharacterResolver (hit DB) trên mỗi player_move RPC chỉ để đổi userID -> characterID. Không
	// serialize field này ra wire format (xem transport/events.go roomPlayerDTO map thủ công).
	UserID string
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
