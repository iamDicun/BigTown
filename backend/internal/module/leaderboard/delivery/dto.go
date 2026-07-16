package delivery

type LeaderboardEntryResponse struct {
	CharacterID string `json:"character_id"`
	Name        string `json:"name"`
	Score       int    `json:"score"`
}
