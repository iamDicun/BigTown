package usecase

import (
	"context"
	"log"

	characterentity "backend/internal/module/character/entity"
	"backend/internal/module/realtime/port"
)

type RealtimeUsecase struct {
	mapReader port.MapReader
	npcReader port.NPCReader
}

type BootstrapData struct {
	TickRateMS       int
	MapCode          string
	WebSocketPath    string
	DefaultRoomID    string
	DefaultChannel   string
	ProtocolFeatures []string

	TilemapAssetKey    string
	TilesetAssetKey    string
	SpawnX             int
	SpawnY             int
	MapWidth           int
	MapHeight          int
	TileSize           int
	LayerNames         []string
	AboveLayerName     string
	CollisionLayerName string
	MusicAssetKey      string
	NPCSpawns          []characterentity.NPCSpawn
}

func NewRealtimeUsecase(mapReader port.MapReader, npcReader port.NPCReader) *RealtimeUsecase {
	return &RealtimeUsecase{mapReader: mapReader, npcReader: npcReader}
}

func (u *RealtimeUsecase) GetBootstrap(ctx context.Context, mapCode string) (*BootstrapData, error) {
	var mapInfo *characterentity.MapInfo
	var err error

	if mapCode != "" {
		mapInfo, err = u.mapReader.GetMapByCode(ctx, mapCode)
	} else {
		mapInfo, err = u.mapReader.GetDefaultMap(ctx)
	}
	if err != nil {
		return nil, err
	}

	var npcSpawns []characterentity.NPCSpawn
	if u.npcReader != nil {
		npcSpawns, err = u.npcReader.GetNPCSpawnsByMapCode(ctx, mapInfo.Code)
		if err != nil {
			log.Printf("realtime: failed to load NPC spawns for map %s: %v", mapInfo.Code, err)
			npcSpawns = nil
		}
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
		TilemapAssetKey:    mapInfo.TilemapAssetKey,
		TilesetAssetKey:    mapInfo.TilesetAssetKey,
		SpawnX:             mapInfo.SpawnX,
		SpawnY:             mapInfo.SpawnY,
		MapWidth:           mapInfo.Width,
		MapHeight:          mapInfo.Height,
		TileSize:           mapInfo.TileSize,
		LayerNames:         mapInfo.LayerNames,
		AboveLayerName:     mapInfo.AboveLayerName,
		CollisionLayerName: mapInfo.CollisionLayerName,
		MusicAssetKey:      mapInfo.MusicAssetKey,
		NPCSpawns:          npcSpawns,
	}, nil
}
