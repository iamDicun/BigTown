package room

import (
	"context"
	"errors"
)

var ErrPlayerNotFound = errors.New("room: player not found")

// RoomStore bọc RAM state sau interface để usecase không phụ thuộc cứng vào implementation
// RAM cụ thể — xem docs/Realtime-Room-State-Decisions.md mục 4. MVP dùng MemoryRoomStore.
type RoomStore interface {
	// JoinRoom trả thêm joined (player thực sự đang có trong room sau lệnh này) và isFirstConnection
	// (true nếu đây là lần đầu character này xuất hiện trong room, false nếu đã có connection khác
	// giữ chỗ sẵn). Cả 2 giá trị được tính atomic trong cùng 1 lock với thao tác insert/lookup — usecase
	// không được tự suy luận isFirstConnection từ 1 GetSnapshot gọi trước đó (race giữa 2 connection
	// join gần như đồng thời của cùng 1 user, xem MemoryRoomStore.JoinRoom).
	JoinRoom(ctx context.Context, roomID string, player RoomPlayer) (snapshot *RoomSnapshot, joined *RoomPlayer, isFirstConnection bool, err error)
	LeaveRoom(ctx context.Context, roomID string, characterID string, clientID string) (*RoomPlayer, bool, error)
	GetSnapshot(ctx context.Context, roomID string) (*RoomSnapshot, error)
	GetPlayer(ctx context.Context, roomID string, characterID string) (*RoomPlayer, error)
	MovePlayer(ctx context.Context, roomID string, characterID string, movement PlayerMovement) (*RoomPlayer, error)
}
