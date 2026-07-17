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
	defaultMapCode string

	repo    port.CharacterRepository
	usecase *usecase.CharacterUsecase
	handler *delivery.CharacterHandler
}

func NewProvider(db *sql.DB, defaultMapCode string) *Provider {
	return &Provider{db: db, defaultMapCode: defaultMapCode}
}

func (p *Provider) Repository() port.CharacterRepository {
	if p.repo == nil {
		p.repo = repository.NewCharacterRepository(p.db, p.defaultMapCode)
	}
	return p.repo
}

func (p *Provider) Usecase() *usecase.CharacterUsecase {
	if p.usecase == nil {
		p.usecase = usecase.NewCharacterUsecase(p.db, p.Repository(), p.defaultMapCode)
	}
	return p.usecase
}

func (p *Provider) Handler() *delivery.CharacterHandler {
	if p.handler == nil {
		p.handler = delivery.NewCharacterHandler(p.Usecase())
	}
	return p.handler
}
