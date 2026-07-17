package usecase

import (
	"context"
	"strings"
	"time"

	"backend/internal/apperror"
	"backend/internal/module/chat/entity"
	"backend/internal/module/chat/port"
)

const (
	defaultMessageType  = "text"
	maxMessageLength    = 500
	defaultHistoryLimit = 50
	maxHistoryLimit     = 100
	defaultSenderName   = "Player"
)

type ChatUsecase struct {
	repo       port.ChatRepository
	publisher  port.RoomPublisher
	characters port.CharacterReader
}

func NewChatUsecase(repo port.ChatRepository, publisher port.RoomPublisher, characters port.CharacterReader) *ChatUsecase {
	return &ChatUsecase{repo: repo, publisher: publisher, characters: characters}
}

type SendMessageInput struct {
	UserID  string
	RoomID  string
	Message string
}

// RoomChatEvent là payload broadcast qua Centrifuge, đúng shape đã chốt trong
// docs/Realtime-Room-State-Decisions.md mục 9.3.
type RoomChatEvent struct {
	Type        string    `json:"type"`
	RoomID      string    `json:"roomId"`
	CharacterID string    `json:"characterId"`
	DisplayName string    `json:"displayName"`
	Message     string    `json:"message"`
	SentAt      time.Time `json:"sentAt"`
}

func (u *ChatUsecase) SendMessage(ctx context.Context, input SendMessageInput) (*entity.ChatMessage, error) {
	roomID := strings.TrimSpace(input.RoomID)
	if roomID == "" {
		return nil, apperror.BadRequest("Thiếu room_id", nil)
	}

	message := strings.TrimSpace(input.Message)
	if message == "" {
		return nil, apperror.BadRequest("Nội dung chat không được để trống", nil)
	}
	if len(message) > maxMessageLength {
		return nil, apperror.BadRequest("Nội dung chat quá dài", nil)
	}

	character, err := u.characters.GetOrCreateForUser(ctx, input.UserID, defaultSenderName)
	if err != nil {
		return nil, err
	}

	saved, err := u.repo.Insert(ctx, roomID, character.ID, message, defaultMessageType)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	saved.CharacterName = character.Name

	event := RoomChatEvent{
		Type:        "player_chat",
		RoomID:      saved.RoomID,
		CharacterID: saved.CharacterID,
		DisplayName: character.Name,
		Message:     saved.Message,
		SentAt:      saved.CreatedAt,
	}

	if err := u.publisher.PublishRoom(ctx, saved.RoomID, event); err != nil {
		return nil, apperror.Internal(err)
	}

	return saved, nil
}

func (u *ChatUsecase) ListRecentMessages(ctx context.Context, roomID string, limit int) ([]entity.ChatMessage, error) {
	roomID = strings.TrimSpace(roomID)
	if roomID == "" {
		return nil, apperror.BadRequest("Thiếu room_id", nil)
	}
	if limit <= 0 {
		limit = defaultHistoryLimit
	}
	if limit > maxHistoryLimit {
		limit = maxHistoryLimit
	}

	messages, err := u.repo.ListRecent(ctx, roomID, limit)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	return messages, nil
}
