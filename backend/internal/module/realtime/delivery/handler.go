package delivery

import (
	"net/http"

	"backend/internal/module/realtime/usecase"
	"backend/internal/response"

	"github.com/gin-gonic/gin"
)

type RealtimeHandler struct {
	usecase *usecase.RealtimeUsecase
}

func NewRealtimeHandler(usecase *usecase.RealtimeUsecase) *RealtimeHandler {
	return &RealtimeHandler{usecase: usecase}
}

func (h *RealtimeHandler) GetBootstrap(ctx *gin.Context) {
	mapCode := ctx.Query("map_code")
	data, err := h.usecase.GetBootstrap(ctx.Request.Context(), mapCode)
	if err != nil {
		ctx.Error(err)
		return
	}

	npcSpawns := make([]NPCSpawnDto, 0, len(data.NPCSpawns))
	for _, s := range data.NPCSpawns {
		npcSpawns = append(npcSpawns, NPCSpawnDto{
			ID:           s.ID,
			Code:         s.Code,
			Name:         s.Name,
			AssetKey:     s.AssetKey,
			SpawnX:       s.SpawnX,
			SpawnY:       s.SpawnY,
			SpawnGroup:   s.SpawnGroup,
			MetadataJSON: s.MetadataJSON,
		})
	}

	ctx.JSON(http.StatusOK, response.SuccessResponse[BootstrapResponse]{
		Success: true,
		Data: BootstrapResponse{
			TickRateMS:       data.TickRateMS,
			MapCode:          data.MapCode,
			WebSocketPath:    data.WebSocketPath,
			DefaultRoomID:    data.DefaultRoomID,
			DefaultChannel:   data.DefaultChannel,
			ProtocolFeatures: data.ProtocolFeatures,
			TilemapAssetKey:  data.TilemapAssetKey,
			TilesetAssetKey:  data.TilesetAssetKey,
			SpawnX:           data.SpawnX,
			SpawnY:           data.SpawnY,
			MapWidth:         data.MapWidth,
			MapHeight:        data.MapHeight,
			TileSize:         data.TileSize,
			LayerNames:       data.LayerNames,
			AboveLayerName:   data.AboveLayerName,
			CollisionLayerName: data.CollisionLayerName,
			MusicAssetKey:      data.MusicAssetKey,
			NPCSpawns:          npcSpawns,
		},
	})
}
