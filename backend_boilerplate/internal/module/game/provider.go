package game

import (
	"database/sql"

	"backend/internal/module/game/delivery"
	"backend/internal/module/game/usecase"
)

type Provider struct {
	db *sql.DB

	usecase *usecase.GameUsecase
	handler *delivery.GameHandler
}

func NewProvider(db *sql.DB) *Provider {
	return &Provider{db: db}
}

func (p *Provider) Usecase() *usecase.GameUsecase {
	if p.usecase == nil {
		p.usecase = usecase.NewGameUsecase(p.db)
	}
	return p.usecase
}

func (p *Provider) Handler() *delivery.GameHandler {
	if p.handler == nil {
		p.handler = delivery.NewGameHandler(p.Usecase())
	}
	return p.handler
}
