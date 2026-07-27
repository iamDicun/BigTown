package character

import (
	"database/sql"

	"backend/internal/module/character/delivery"
	"backend/internal/module/character/port"
	"backend/internal/module/character/repository"
	"backend/internal/module/character/usecase"
)

type Provider struct {
	db             *sql.DB
	users          port.UserReader
	defaultMapCode string
	startingCoins  int

	repo    port.CharacterRepository
	usecase *usecase.CharacterUsecase
	handler *delivery.CharacterHandler
}

func NewProvider(db *sql.DB, users port.UserReader, defaultMapCode string, startingCoins int) *Provider {
	return &Provider{db: db, users: users, defaultMapCode: defaultMapCode, startingCoins: startingCoins}
}

func (p *Provider) Repository() port.CharacterRepository {
	if p.repo == nil {
		p.repo = repository.NewCharacterRepository(p.db, p.defaultMapCode, p.startingCoins)
	}
	return p.repo
}

func (p *Provider) Usecase() *usecase.CharacterUsecase {
	if p.usecase == nil {
		p.usecase = usecase.NewCharacterUsecase(p.db, p.Repository(), p.users, p.defaultMapCode)
	}
	return p.usecase
}

func (p *Provider) Handler() *delivery.CharacterHandler {
	if p.handler == nil {
		p.handler = delivery.NewCharacterHandler(p.Usecase())
	}
	return p.handler
}
