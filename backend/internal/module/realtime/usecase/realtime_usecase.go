package usecase

import (
	"context"

	"backend/internal/module/realtime/port"
)

type RealtimeUsecase struct {
	mapReader port.MapReader
}

type BootstrapData struct {
	TickRateMS       int
	MapCode          string
	WebSocketPath    string
	DefaultRoomID    string
	DefaultChannel   string
	ProtocolFeatures []string

	TilemapAssetKey   string
	TilesetAssetKey   string
	SpawnX            int
	SpawnY            int
	MapWidth          int
	MapHeight         int
	TileSize          int
	LayerNames        []string
	AboveLayerName    string
	CollisionLayerName string
}

func NewRealtimeUsecase(mapReader port.MapReader) *RealtimeUsecase {
	return &RealtimeUsecase{mapReader: mapReader}
}

func (u *RealtimeUsecase) GetBootstrap(ctx context.Context) (*BootstrapData, error) {
	mapInfo, err := u.mapReader.GetDefaultMap(ctx)
	if err != nil {
		return nil, err
	}

	return &BootstrapData{
		TickRateMS:     100,
		MapCode:        mapInfo.Code,
		WebSocketPath:  "/connection/websocket",
		DefaultRoomID:  mapInfo.Code,
		DefaultChannel: "room:" + mapInfo.Code,
		ProtocolFeatures: []string{
			"centrifuge_transport",
			"room_channels",
			"realtime_movement",
			"chat_bubble",
			"npc_combat",
		},
		TilemapAssetKey:   mapInfo.TilemapAssetKey,
		TilesetAssetKey:   mapInfo.TilesetAssetKey,
		SpawnX:            mapInfo.SpawnX,
		SpawnY:            mapInfo.SpawnY,
		MapWidth:       mapInfo.Width,
		MapHeight:      mapInfo.Height,
		TileSize:       mapInfo.TileSize,
		LayerNames:     mapInfo.LayerNames,
		AboveLayerName:    mapInfo.AboveLayerName,
		CollisionLayerName: mapInfo.CollisionLayerName,
	}, nil
}
