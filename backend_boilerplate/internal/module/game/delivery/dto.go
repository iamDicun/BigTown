package delivery

type BootstrapResponse struct {
	TickRateMS int      `json:"tick_rate_ms"`
	MapCode    string   `json:"map_code"`
	Features   []string `json:"features"`
}

type LeaderboardEntryResponse struct {
	CharacterID string `json:"character_id"`
	Name        string `json:"name"`
	Score       int    `json:"score"`
}
