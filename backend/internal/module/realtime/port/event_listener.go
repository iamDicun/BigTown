package port

import "context"

type RoomEventListener interface {
	OnPlayerJoin(ctx context.Context, roomID string, characterID string, coins int) error
	OnPlayerLeave(ctx context.Context, roomID string, characterID string) error
}
