package port

import (
	"context"

	"backend/internal/module/chat/entity"
)

type ChatRepository interface {
	Insert(ctx context.Context, roomID string, characterID string, message string, messageType string) (*entity.ChatMessage, error)
	ListRecent(ctx context.Context, roomID string, limit int) ([]entity.ChatMessage, error)
}
