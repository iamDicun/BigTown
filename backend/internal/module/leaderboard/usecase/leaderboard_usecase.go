package usecase

import (
	"context"

	"backend/internal/module/leaderboard/entity"
	"backend/internal/module/leaderboard/port"
)

type LeaderboardUsecase struct {
	repo port.LeaderboardRepository
}

func NewLeaderboardUsecase(repo port.LeaderboardRepository) *LeaderboardUsecase {
	return &LeaderboardUsecase{repo: repo}
}

func (u *LeaderboardUsecase) GetLeaderboard(ctx context.Context) ([]entity.Entry, error) {
	return u.repo.GetLeaderboard(ctx, 10)
}
