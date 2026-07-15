package game

import "database/sql"

type GameModule struct {
	provider *Provider
}

func NewGameModule(db *sql.DB) *GameModule {
	return &GameModule{provider: NewProvider(db)}
}
