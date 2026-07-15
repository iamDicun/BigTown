package runtime

import "time"

type Direction string

const (
	DirectionUp    Direction = "up"
	DirectionDown  Direction = "down"
	DirectionLeft  Direction = "left"
	DirectionRight Direction = "right"
)

type NPCState string

const (
	NPCStateIdle    NPCState = "idle"
	NPCStateChasing NPCState = "chasing"
	NPCStateDead    NPCState = "dead"
)

type GameRoom struct {
	MapID   string
	Players map[string]*RoomPlayer
	NPCs    map[string]*RoomNPC
}

type RoomPlayer struct {
	CharacterID string
	ClientID    string
	X           int
	Y           int
	Direction   Direction
	Moving      bool
	CurrentHP   int
	WeaponID    *string
	AttackUntil time.Time
	LastSeenAt  time.Time
}

type RoomNPC struct {
	RuntimeID string
	SpawnID   string
	NPCTypeID string
	X         int
	Y         int
	CurrentHP int
	Alive     bool
	AIState   NPCState
	RespawnAt *time.Time
}
