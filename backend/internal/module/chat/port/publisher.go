package port

import "context"

type RoomPublisher interface {
	PublishRoom(ctx context.Context, roomID string, event any) error
}
