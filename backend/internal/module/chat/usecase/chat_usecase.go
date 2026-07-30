package usecase

import (
	"context"
	"log"
	"strings"
	"time"

	"backend/internal/apperror"
	"backend/internal/module/chat/entity"
	"backend/internal/module/chat/port"
	"github.com/google/uuid"
)

const (
	defaultMessageType  = "text"
	maxMessageLength    = 500
	defaultHistoryLimit = 50
	maxHistoryLimit     = 100
)

type dbWriteTask struct {
	ID          string
	RoomID      string
	CharacterID string
	Message     string
	MessageType string
	CreatedAt   time.Time
}

type ChatUsecase struct {
	repo       port.ChatRepository
	publisher  port.RoomPublisher
	characters port.CharacterReader
	writeChan  chan dbWriteTask
}

func NewChatUsecase(repo port.ChatRepository, publisher port.RoomPublisher, characters port.CharacterReader) *ChatUsecase {
	u := &ChatUsecase{
		repo:       repo,
		publisher:  publisher,
		characters: characters,
		writeChan:  make(chan dbWriteTask, 10000),
	}
	u.startBackgroundWorkers(5)
	return u
}

func (u *ChatUsecase) startBackgroundWorkers(count int) {
	for i := 0; i < count; i++ {
		go func(workerID int) {
			// Sử dụng context.Background() vì context của HTTP request sẽ bị cancel khi request kết thúc
			ctx := context.Background()
			for task := range u.writeChan {
				err := u.repo.InsertWithID(ctx, task.ID, task.RoomID, task.CharacterID, task.Message, task.MessageType, task.CreatedAt)
				if err != nil {
					log.Printf("[Chat-Async-DB-Worker-%d] ERROR inserting chat message: %v", workerID, err)
				}
			}
		}(i)
	}
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

	character, err := u.characters.GetByUserID(ctx, input.UserID)
	if err != nil {
		return nil, err
	}

	// Sinh ID và CreatedAt ngay lập tức để trả về cho Client và broadcast realtime
	msgID := uuid.NewString()
	now := time.Now()

	event := RoomChatEvent{
		Type:        "player_chat",
		RoomID:      roomID,
		CharacterID: character.ID,
		DisplayName: character.Name,
		Message:     message,
		SentAt:      now,
	}

	// Phát tin nhắn realtime ngay lập tức
	if err := u.publisher.PublishRoom(ctx, roomID, event); err != nil {
		return nil, apperror.Internal(err)
	}

	// Đẩy vào hàng đợi ghi DB bất đồng bộ
	select {
	case u.writeChan <- dbWriteTask{
		ID:          msgID,
		RoomID:      roomID,
		CharacterID: character.ID,
		Message:     message,
		MessageType: defaultMessageType,
		CreatedAt:   now,
	}:
	default:
		// Dự phòng nếu hàng đợi đầy (hệ thống quá tải cực hạn)
		log.Printf("[Chat-Usecase] WARNING: chat db queue full, dropping write task for message: %s", msgID)
	}

	saved := &entity.ChatMessage{
		ID:            msgID,
		RoomID:        roomID,
		CharacterID:   character.ID,
		CharacterName: character.Name,
		Message:       message,
		MessageType:   defaultMessageType,
		CreatedAt:     now,
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
