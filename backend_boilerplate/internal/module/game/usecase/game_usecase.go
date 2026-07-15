package usecase

import (
	"context"
	"database/sql"
)

type GameUsecase struct {
	db *sql.DB
}

type BootstrapData struct {
	TickRateMS int
	MapCode    string
	Features   []string
}

type LeaderboardEntry struct {
	CharacterID string
	Name        string
	Score       int
}

func NewGameUsecase(db *sql.DB) *GameUsecase {
	return &GameUsecase{db: db}
}

func (u *GameUsecase) GetBootstrap(ctx context.Context) (*BootstrapData, error) {
	_ = ctx
	_ = u.db

	return &BootstrapData{
		TickRateMS: 100,
		MapCode:    "starter-town",
		Features: []string{
			"realtime_movement",
			"avatar_equipment",
			"npc_combat",
			"chat_bubble",
			"leaderboard",
		},
	}, nil
}

func (u *GameUsecase) GetLeaderboard(ctx context.Context) ([]LeaderboardEntry, error) {
	_ = ctx
	_ = u.db

	return []LeaderboardEntry{}, nil
}
