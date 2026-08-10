package character

import (
	"database/sql"

	"backend/internal/module/character/port"
	"backend/internal/module/character/usecase"
)

type CharacterModule struct {
	provider *Provider
}

func NewCharacterModule(db *sql.DB, users port.UserReader, defaultMapCode string, startingCoins int) *CharacterModule {
	return &CharacterModule{provider: NewProvider(db, users, defaultMapCode, startingCoins)}
}

// Usecase() cho phép module khác (chat, realtime) tái dùng cùng CharacterUsecase để get-or-create
// character mà không phải tự dựng lại provider riêng.
func (m *CharacterModule) Usecase() *usecase.CharacterUsecase {
	return m.provider.Usecase()
}

func (m *CharacterModule) Repository() port.CharacterRepository {
	return m.provider.Repository()
}
