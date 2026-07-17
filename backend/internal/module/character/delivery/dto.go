package delivery

type CharacterResponse struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	MapID        *string `json:"map_id"`
	BaseAssetKey string  `json:"base_asset_key"`
	Coins        int     `json:"coins"`
	Score        int     `json:"score"`
	LastX        *int    `json:"last_x"`
	LastY        *int    `json:"last_y"`
}
