package delivery

import (
	"net/http"

	"backend/internal/module/game/usecase"
	"backend/internal/response"

	"github.com/gin-gonic/gin"
)

type GameHandler struct {
	usecase *usecase.GameUsecase
}

func NewGameHandler(usecase *usecase.GameUsecase) *GameHandler {
	return &GameHandler{usecase: usecase}
}

func (h *GameHandler) GetBootstrap(ctx *gin.Context) {
	data, err := h.usecase.GetBootstrap(ctx.Request.Context())
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, response.SuccessResponse[BootstrapResponse]{
		Success: true,
		Data: BootstrapResponse{
			TickRateMS: data.TickRateMS,
			MapCode:    data.MapCode,
			Features:   data.Features,
		},
	})
}

func (h *GameHandler) GetLeaderboard(ctx *gin.Context) {
	entries, err := h.usecase.GetLeaderboard(ctx.Request.Context())
	if err != nil {
		ctx.Error(err)
		return
	}

	responses := make([]LeaderboardEntryResponse, 0, len(entries))
	for _, entry := range entries {
		responses = append(responses, LeaderboardEntryResponse{
			CharacterID: entry.CharacterID,
			Name:        entry.Name,
			Score:       entry.Score,
		})
	}

	ctx.JSON(http.StatusOK, response.SuccessResponse[[]LeaderboardEntryResponse]{Success: true, Data: responses})
}
