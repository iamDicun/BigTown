package editor

import (
	"database/sql"

	charPort "backend/internal/module/character/port"
	"backend/internal/module/editor/port"
)

type EditorModule struct {
	provider *Provider
}

func NewEditorModule(db *sql.DB, publisher port.RoomPublisher, charRepo charPort.CharacterRepository) *EditorModule {
	return &EditorModule{provider: NewProvider(db, charRepo, publisher)}
}
