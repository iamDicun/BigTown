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

type SpritesheetConfigResponse struct {
	FrameWidth    int `json:"frame_width"`
	FrameHeight   int `json:"frame_height"`
	Columns       int `json:"columns"`
	RowIdleDown   int `json:"row_idle_down"`
	RowWalkDown   int `json:"row_walk_down"`
	RowIdleUp     int `json:"row_idle_up"`
	RowWalkUp     int `json:"row_walk_up"`
	RowWalkSide   int `json:"row_walk_side"`
	WalkFrameRate int `json:"walk_frame_rate"`
	IdleFrameRate int `json:"idle_frame_rate"`
}

type CharacterOptionResponse struct {
	Name         string                    `json:"name"`
	BaseAssetKey string                    `json:"base_asset_key"`
	PreviewURL   string                    `json:"preview_url"`
	Spritesheet  SpritesheetConfigResponse `json:"spritesheet"`
}

type CreateCharacterRequest struct {
	Name         string `json:"name" binding:"required"`
	BaseAssetKey string `json:"base_asset_key" binding:"required"`
}
