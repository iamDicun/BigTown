package repository

import (
	"context"
	"database/sql"

	"backend/internal/module/chat/entity"
	"backend/internal/module/chat/port"
)

var _ port.ChatRepository = (*ChatRepository)(nil)

type ChatRepository struct {
	db *sql.DB
}

func NewChatRepository(db *sql.DB) *ChatRepository {
	return &ChatRepository{db: db}
}

func (r *ChatRepository) Insert(ctx context.Context, roomID string, characterID string, message string, messageType string) (*entity.ChatMessage, error) {
	var msg entity.ChatMessage
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO chat_messages (room_id, character_id, message, message_type)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text, room_id, character_id::text, message, message_type, created_at
	`, roomID, characterID, message, messageType).Scan(
		&msg.ID, &msg.RoomID, &msg.CharacterID, &msg.Message, &msg.MessageType, &msg.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

func (r *ChatRepository) ListRecent(ctx context.Context, roomID string, limit int) ([]entity.ChatMessage, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT cm.id::text, cm.room_id, cm.character_id::text, c.name, cm.message, cm.message_type, cm.created_at
		FROM chat_messages cm
		JOIN characters c ON c.id = cm.character_id
		WHERE cm.room_id = $1
		ORDER BY cm.created_at DESC
		LIMIT $2
	`, roomID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []entity.ChatMessage{}
	for rows.Next() {
		var msg entity.ChatMessage
		if err := rows.Scan(&msg.ID, &msg.RoomID, &msg.CharacterID, &msg.CharacterName, &msg.Message, &msg.MessageType, &msg.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Query lấy mới nhất trước (DESC) để LIMIT đúng N tin gần nhất; đảo lại thành thứ tự thời
	// gian tăng dần trước khi trả cho FE để ChatPanel không phải tự sort lại.
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}
