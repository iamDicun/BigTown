package user

import (
	"database/sql"

	"backend/internal/module/user/delivery"
	"backend/internal/module/user/port"
	"backend/internal/module/user/repository"
	"backend/internal/module/user/usecase"
)

type Provider struct {
	db *sql.DB

	repo    port.UserRepository
	usecase *usecase.UserUsecase
	handler *delivery.UserHandler
}

func NewProvider(db *sql.DB) *Provider {
	return &Provider{db: db}
}

func (p *Provider) Repository() port.UserRepository {
	if p.repo == nil {
		p.repo = repository.NewUserRepository(p.db)
	}
	return p.repo
}

func (p *Provider) Usecase() *usecase.UserUsecase {
	if p.usecase == nil {
		p.usecase = usecase.NewUserUsecase(p.Repository())
	}
	return p.usecase
}

func (p *Provider) Handler() *delivery.UserHandler {
	if p.handler == nil {
		p.handler = delivery.NewUserHandler(p.Usecase())
	}
	return p.handler
}
