package repository

import (
	"context"
	"database/sql"

	"backend/internal/module/leaderboard/entity"
	"backend/internal/module/leaderboard/port"
)

var _ port.LeaderboardRepository = (*LeaderboardRepository)(nil)

type LeaderboardRepository struct {
	db *sql.DB
}

func NewLeaderboardRepository(db *sql.DB) *LeaderboardRepository {
	return &LeaderboardRepository{db: db}
}

func (r *LeaderboardRepository) GetLeaderboard(ctx context.Context, limit int) ([]entity.Entry, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id::text, name, score
		FROM characters
		ORDER BY score DESC, name ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := []entity.Entry{}
	for rows.Next() {
		var entry entity.Entry
		if err := rows.Scan(&entry.CharacterID, &entry.Name, &entry.Score); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}
