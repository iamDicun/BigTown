package editor

import (
	"database/sql"

	charPort "backend/internal/module/character/port"
	"backend/internal/module/editor/port"
	"backend/internal/module/editor/room"
)

type EditorModule struct {
	provider *Provider
}

func NewEditorModule(db *sql.DB, publisher port.RoomPublisher, charRepo charPort.CharacterRepository) *EditorModule {
	return &EditorModule{provider: NewProvider(db, charRepo, publisher)}
}

func (m *EditorModule) Shutdown() {
	m.provider.RoomManager().Shutdown()
}

func (m *EditorModule) RoomManager() *room.RoomManager {
	return m.provider.RoomManager()
}
