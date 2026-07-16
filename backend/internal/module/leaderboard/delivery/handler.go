package delivery

import (
	"net/http"

	"backend/internal/module/leaderboard/usecase"
	"backend/internal/response"

	"github.com/gin-gonic/gin"
)

type LeaderboardHandler struct {
	usecase *usecase.LeaderboardUsecase
}

func NewLeaderboardHandler(usecase *usecase.LeaderboardUsecase) *LeaderboardHandler {
	return &LeaderboardHandler{usecase: usecase}
}

func (h *LeaderboardHandler) GetLeaderboard(ctx *gin.Context) {
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
