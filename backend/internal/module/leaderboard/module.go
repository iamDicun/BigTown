package leaderboard

import "database/sql"

type LeaderboardModule struct {
	provider *Provider
}

func NewLeaderboardModule(db *sql.DB) *LeaderboardModule {
	return &LeaderboardModule{provider: NewProvider(db)}
}
