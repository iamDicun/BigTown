package room

import (
	"context"
	"errors"
)

var ErrPlayerNotFound = errors.New("room: player not found")

// RoomStore bọc RAM state sau interface để usecase không phụ thuộc cứng vào implementation
// RAM cụ thể — xem docs/Realtime-Room-State-Decisions.md mục 4. MVP dùng MemoryRoomStore.
type RoomStore interface {
	JoinRoom(ctx context.Context, roomID string, player RoomPlayer) (*RoomSnapshot, error)
	LeaveRoom(ctx context.Context, roomID string, characterID string) error
	GetSnapshot(ctx context.Context, roomID string) (*RoomSnapshot, error)
	GetPlayer(ctx context.Context, roomID string, characterID string) (*RoomPlayer, error)
	MovePlayer(ctx context.Context, roomID string, characterID string, movement PlayerMovement) (*RoomPlayer, error)
}
