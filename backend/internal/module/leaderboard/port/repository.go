package port

import (
	"context"

	"backend/internal/module/leaderboard/entity"
)

type LeaderboardRepository interface {
	GetLeaderboard(ctx context.Context, limit int) ([]entity.Entry, error)
}
