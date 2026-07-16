package usecase

import "context"

type RealtimeUsecase struct{}

type BootstrapData struct {
	TickRateMS       int
	MapCode          string
	WebSocketPath    string
	DefaultRoomID    string
	DefaultChannel   string
	ProtocolFeatures []string
}

func NewRealtimeUsecase() *RealtimeUsecase {
	return &RealtimeUsecase{}
}

func (u *RealtimeUsecase) GetBootstrap(ctx context.Context) (*BootstrapData, error) {
	_ = ctx

	return &BootstrapData{
		TickRateMS:     100,
		MapCode:        "starter-town",
		WebSocketPath:  "/connection/websocket",
		DefaultRoomID:  "starter-town",
		DefaultChannel: "room:starter-town",
		ProtocolFeatures: []string{
			"centrifuge_transport",
			"room_channels",
			"realtime_movement",
			"chat_bubble",
			"npc_combat",
		},
	}, nil
}
