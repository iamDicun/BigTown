package delivery

type BootstrapResponse struct {
	TickRateMS       int      `json:"tick_rate_ms"`
	MapCode          string   `json:"map_code"`
	WebSocketPath    string   `json:"websocket_path"`
	DefaultRoomID    string   `json:"default_room_id"`
	DefaultChannel   string   `json:"default_channel"`
	ProtocolFeatures []string `json:"protocol_features"`
}
