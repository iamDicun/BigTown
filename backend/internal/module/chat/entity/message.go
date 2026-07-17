package entity

import "time"

type ChatMessage struct {
	ID            string
	RoomID        string
	CharacterID   string
	CharacterName string
	Message       string
	MessageType   string
	CreatedAt     time.Time
}
