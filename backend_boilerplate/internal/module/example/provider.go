package example

import (
	"database/sql"

	"backend/internal/module/example/delivery"
	"backend/internal/module/example/port"
	"backend/internal/module/example/repository"
	"backend/internal/module/example/usecase"
)

type Provider struct {
	db *sql.DB // không dùng vì repository ở đây là in-memory, giữ field để khớp convention provider của các module thật

	repo    port.ItemRepository
	usecase *usecase.ExampleUsecase
	handler *delivery.ExampleHandler
}

func NewProvider(db *sql.DB) *Provider {
	return &Provider{db: db}
}

func (p *Provider) Repository() port.ItemRepository {
	if p.repo == nil {
		p.repo = repository.NewMemoryItemRepository()
	}
	return p.repo
}

func (p *Provider) Usecase() *usecase.ExampleUsecase {
	if p.usecase == nil {
		p.usecase = usecase.NewExampleUsecase(p.Repository())
	}
	return p.usecase
}

func (p *Provider) Handler() *delivery.ExampleHandler {
	if p.handler == nil {
		p.handler = delivery.NewExampleHandler(p.Usecase())
	}
	return p.handler
}
