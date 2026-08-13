package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	characterentity "backend/internal/module/character/entity"
	"backend/internal/module/chat/entity"
	"backend/internal/module/chat/port"
)

type mockChatRepo struct {
	insertFn       func(ctx context.Context, roomID string, characterID string, message string, messageType string) (*entity.ChatMessage, error)
	insertWithIDFn func(ctx context.Context, id string, roomID string, characterID string, message string, messageType string, createdAt time.Time) error
	listRecentFn   func(ctx context.Context, roomID string, limit int) ([]entity.ChatMessage, error)
}

func (m *mockChatRepo) Insert(ctx context.Context, roomID string, characterID string, message string, messageType string) (*entity.ChatMessage, error) {
	if m.insertFn != nil {
		return m.insertFn(ctx, roomID, characterID, message, messageType)
	}
	return &entity.ChatMessage{}, nil
}
func (m *mockChatRepo) InsertWithID(ctx context.Context, id string, roomID string, characterID string, message string, messageType string, createdAt time.Time) error {
	if m.insertWithIDFn != nil {
		return m.insertWithIDFn(ctx, id, roomID, characterID, message, messageType, createdAt)
	}
	return nil
}
func (m *mockChatRepo) ListRecent(ctx context.Context, roomID string, limit int) ([]entity.ChatMessage, error) {
	if m.listRecentFn != nil {
		return m.listRecentFn(ctx, roomID, limit)
	}
	return []entity.ChatMessage{}, nil
}

var _ port.ChatRepository = (*mockChatRepo)(nil)

type mockPublisher struct {
	publishRoomFn func(ctx context.Context, roomID string, event any) error
}

func (m *mockPublisher) PublishRoom(ctx context.Context, roomID string, event any) error {
	if m.publishRoomFn != nil {
		return m.publishRoomFn(ctx, roomID, event)
	}
	return nil
}

var _ port.RoomPublisher = (*mockPublisher)(nil)

type mockCharacterReader struct {
	getByUserIDFn func(ctx context.Context, userID string) (*characterentity.Character, error)
}

func (m *mockCharacterReader) GetByUserID(ctx context.Context, userID string) (*characterentity.Character, error) {
	if m.getByUserIDFn != nil {
		return m.getByUserIDFn(ctx, userID)
	}
	return &characterentity.Character{ID: "char-1", UserID: userID, Name: "Test Player"}, nil
}

var _ port.CharacterReader = (*mockCharacterReader)(nil)

func TestSendMessage_Validation(t *testing.T) {
	uc := NewChatUsecase(&mockChatRepo{}, &mockPublisher{}, &mockCharacterReader{})

	// Empty RoomID
	_, err := uc.SendMessage(context.Background(), SendMessageInput{UserID: "user-1", RoomID: "", Message: "Hello"})
	if err == nil {
		t.Error("expected error for empty roomID")
	}

	// Empty Message
	_, err = uc.SendMessage(context.Background(), SendMessageInput{UserID: "user-1", RoomID: "room-1", Message: "   "})
	if err == nil {
		t.Error("expected error for empty message")
	}

	// Too Long Message (> 500 chars)
	longMsg := strings.Repeat("A", 501)
	_, err = uc.SendMessage(context.Background(), SendMessageInput{UserID: "user-1", RoomID: "room-1", Message: longMsg})
	if err == nil {
		t.Error("expected error for message > 500 chars")
	}
}

func TestSendMessage_Success(t *testing.T) {
	published := false
	pub := &mockPublisher{
		publishRoomFn: func(ctx context.Context, roomID string, event any) error {
			published = true
			return nil
		},
	}
	uc := NewChatUsecase(&mockChatRepo{}, pub, &mockCharacterReader{})

	msg, err := uc.SendMessage(context.Background(), SendMessageInput{
		UserID: "user-1", RoomID: "map-main", Message: "Hello BigTown!",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Message != "Hello BigTown!" {
		t.Errorf("expected 'Hello BigTown!', got %s", msg.Message)
	}
	if msg.CharacterName != "Test Player" {
		t.Errorf("expected Test Player, got %s", msg.CharacterName)
	}
	if !published {
		t.Error("expected message to be published to centrifuge")
	}
}

func TestSendMessage_PublishError(t *testing.T) {
	pub := &mockPublisher{
		publishRoomFn: func(ctx context.Context, roomID string, event any) error {
			return errors.New("centrifuge server down")
		},
	}
	uc := NewChatUsecase(&mockChatRepo{}, pub, &mockCharacterReader{})

	_, err := uc.SendMessage(context.Background(), SendMessageInput{
		UserID: "user-1", RoomID: "map-main", Message: "Hello",
	})
	if err == nil {
		t.Error("expected error when publish fails")
	}
}

func TestListRecentMessages_SuccessAndLimits(t *testing.T) {
	queryLimit := 0
	repo := &mockChatRepo{
		listRecentFn: func(ctx context.Context, roomID string, limit int) ([]entity.ChatMessage, error) {
			queryLimit = limit
			return []entity.ChatMessage{
				{ID: "m1", RoomID: roomID, Message: "Msg 1"},
				{ID: "m2", RoomID: roomID, Message: "Msg 2"},
			}, nil
		},
	}
	uc := NewChatUsecase(repo, &mockPublisher{}, &mockCharacterReader{})

	// Success with valid limit
	msgs, err := uc.ListRecentMessages(context.Background(), "room-1", 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 2 {
		t.Errorf("expected 2 messages, got %d", len(msgs))
	}
	if queryLimit != 20 {
		t.Errorf("expected query limit 20, got %d", queryLimit)
	}

	// Limit <= 0 should default to 50
	_, _ = uc.ListRecentMessages(context.Background(), "room-1", 0)
	if queryLimit != 50 {
		t.Errorf("expected default limit 50 for limit <= 0, got %d", queryLimit)
	}

	// Limit > 100 should cap to 100
	_, _ = uc.ListRecentMessages(context.Background(), "room-1", 200)
	if queryLimit != 100 {
		t.Errorf("expected max limit cap 100, got %d", queryLimit)
	}
}
