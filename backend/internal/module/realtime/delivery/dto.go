package delivery

type BootstrapResponse struct {
	TickRateMS       int      `json:"tick_rate_ms"`
	MapCode          string   `json:"map_code"`
	WebSocketPath    string   `json:"websocket_path"`
	DefaultRoomID    string   `json:"default_room_id"`
	DefaultChannel   string   `json:"default_channel"`
	ProtocolFeatures []string `json:"protocol_features"`

	TilemapAssetKey string   `json:"tilemap_asset_key"`
	TilesetAssetKey string   `json:"tileset_asset_key"`
	SpawnX            int      `json:"spawn_x"`
	SpawnY            int      `json:"spawn_y"`
	MapWidth          int      `json:"map_width"`
	MapHeight         int      `json:"map_height"`
	TileSize          int      `json:"tile_size"`
	LayerNames        []string `json:"layer_names,omitempty"`
	AboveLayerName  string   `json:"above_layer_name,omitempty"`
	CollisionLayerName string `json:"collision_layer_name,omitempty"`
}
