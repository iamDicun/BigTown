package port

import (
	"context"
	"time"

	"backend/internal/module/chat/entity"
)

type ChatRepository interface {
	Insert(ctx context.Context, roomID string, characterID string, message string, messageType string) (*entity.ChatMessage, error)
	InsertWithID(ctx context.Context, id string, roomID string, characterID string, message string, messageType string, createdAt time.Time) error
	ListRecent(ctx context.Context, roomID string, limit int) ([]entity.ChatMessage, error)
}
