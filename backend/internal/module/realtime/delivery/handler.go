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
	data, err := h.usecase.GetBootstrap(ctx.Request.Context())
	if err != nil {
		ctx.Error(err)
		return
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
		},
	})
}
