package delivery

import "time"

type SendChatMessageRequest struct {
	Message string `json:"message" binding:"required"`
}

type ChatMessageResponse struct {
	ID            string    `json:"id"`
	RoomID        string    `json:"room_id"`
	CharacterID   string    `json:"character_id"`
	CharacterName string    `json:"character_name"`
	Message       string    `json:"message"`
	MessageType   string    `json:"message_type"`
	CreatedAt     time.Time `json:"created_at"`
}
