package leaderboard

import (
	"database/sql"

	"backend/internal/module/leaderboard/delivery"
	"backend/internal/module/leaderboard/port"
	"backend/internal/module/leaderboard/repository"
	"backend/internal/module/leaderboard/usecase"
)

type Provider struct {
	db *sql.DB

	repo    port.LeaderboardRepository
	usecase *usecase.LeaderboardUsecase
	handler *delivery.LeaderboardHandler
}

func NewProvider(db *sql.DB) *Provider {
	return &Provider{db: db}
}

func (p *Provider) Repository() port.LeaderboardRepository {
	if p.repo == nil {
		p.repo = repository.NewLeaderboardRepository(p.db)
	}
	return p.repo
}

func (p *Provider) Usecase() *usecase.LeaderboardUsecase {
	if p.usecase == nil {
		p.usecase = usecase.NewLeaderboardUsecase(p.Repository())
	}
	return p.usecase
}

func (p *Provider) Handler() *delivery.LeaderboardHandler {
	if p.handler == nil {
		p.handler = delivery.NewLeaderboardHandler(p.Usecase())
	}
	return p.handler
}
