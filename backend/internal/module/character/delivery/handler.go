package delivery

import (
	"net/http"

	"backend/internal/apperror"
	"backend/internal/module/character/usecase"
	"backend/internal/response"

	"github.com/gin-gonic/gin"
)

const defaultCharacterName = "Player"

type CharacterHandler struct {
	usecase *usecase.CharacterUsecase
}

func NewCharacterHandler(usecase *usecase.CharacterUsecase) *CharacterHandler {
	return &CharacterHandler{usecase: usecase}
}

func (h *CharacterHandler) GetMe(ctx *gin.Context) {
	userIDValue, ok := ctx.Get("user_id")
	if !ok {
		ctx.Error(apperror.Unauthorized("Thiếu user_id", nil))
		return
	}

	character, err := h.usecase.GetOrCreateForUser(ctx.Request.Context(), userIDValue.(string), defaultCharacterName)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, response.SuccessResponse[CharacterResponse]{
		Success: true,
		Data: CharacterResponse{
			ID:           character.ID,
			Name:         character.Name,
			MapID:        character.MapID,
			BaseAssetKey: character.BaseAssetKey,
			Coins:        character.Coins,
			Score:        character.Score,
			LastX:        character.LastX,
			LastY:        character.LastY,
		},
	})
}
