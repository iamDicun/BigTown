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

	character, err := h.usecase.GetByUserID(ctx.Request.Context(), userIDValue.(string))
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

func (h *CharacterHandler) ListOptions(ctx *gin.Context) {
	options := h.usecase.ListOptions()
	data := make([]CharacterOptionResponse, 0, len(options))
	for _, option := range options {
		data = append(data, CharacterOptionResponse{
			Name:         option.Name,
			BaseAssetKey: option.BaseAssetKey,
			PreviewURL:   option.PreviewURL,
			Spritesheet: SpritesheetConfigResponse{
				FrameWidth:    option.Spritesheet.FrameWidth,
				FrameHeight:   option.Spritesheet.FrameHeight,
				Columns:       option.Spritesheet.Columns,
				RowIdleDown:   option.Spritesheet.RowIdleDown,
				RowWalkDown:   option.Spritesheet.RowWalkDown,
				RowIdleUp:     option.Spritesheet.RowIdleUp,
				RowWalkUp:     option.Spritesheet.RowWalkUp,
				RowWalkSide:   option.Spritesheet.RowWalkSide,
				WalkFrameRate: option.Spritesheet.WalkFrameRate,
				IdleFrameRate: option.Spritesheet.IdleFrameRate,
			},
		})
	}

	ctx.JSON(http.StatusOK, response.SuccessResponse[[]CharacterOptionResponse]{Success: true, Data: data})
}

func (h *CharacterHandler) Create(ctx *gin.Context) {
	userIDValue, ok := ctx.Get("user_id")
	if !ok {
		ctx.Error(apperror.Unauthorized("Thiếu user_id", nil))
		return
	}

	var req CreateCharacterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Error(apperror.BadRequest("Dữ liệu tạo nhân vật không hợp lệ", err))
		return
	}

	character, err := h.usecase.CreateForUser(ctx.Request.Context(), userIDValue.(string), req.Name, req.BaseAssetKey)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusCreated, response.SuccessResponse[CharacterResponse]{
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
