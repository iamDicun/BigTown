package chat

import (
	"database/sql"

	"backend/internal/module/chat/port"
)

type ChatModule struct {
	provider *Provider
}

func NewChatModule(db *sql.DB, publisher port.RoomPublisher, characters port.CharacterReader) *ChatModule {
	return &ChatModule{provider: NewProvider(db, publisher, characters)}
}
