package delivery

import (
	"net/http"

	"backend/internal/apperror"
	"backend/internal/module/editor/usecase"
	"backend/internal/response"

	"github.com/gin-gonic/gin"
)

type EditorHandler struct {
	usecase *usecase.EditorUsecase
}

func NewEditorHandler(u *usecase.EditorUsecase) *EditorHandler {
	return &EditorHandler{usecase: u}
}

func (h *EditorHandler) GetEditorData(ctx *gin.Context) {
	userID, ok := ctx.Get("user_id")
	if !ok {
		ctx.Error(apperror.Unauthorized("Thiếu user_id", nil))
		return
	}

	mapCode := ctx.Query("map_code")
	if mapCode == "" {
		ctx.Error(apperror.BadRequest("Thiếu map_code", nil))
		return
	}

	res, err := h.usecase.GetEditorData(ctx.Request.Context(), userID.(string), mapCode)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, response.SuccessResponse[*usecase.GetEditorDataOutput]{
		Success: true,
		Data:    res,
	})
}

func (h *EditorHandler) PlaceItem(ctx *gin.Context) {
	userID, ok := ctx.Get("user_id")
	if !ok {
		ctx.Error(apperror.Unauthorized("Thiếu user_id", nil))
		return
	}

	var req usecase.PlaceItemInput
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Error(apperror.BadRequest("Dữ liệu đặt vật phẩm không hợp lệ", err))
		return
	}

	res, err := h.usecase.PlaceItem(ctx.Request.Context(), userID.(string), req)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, response.SuccessResponse[*usecase.PlaceItemOutput]{
		Success: true,
		Data:    res,
	})
}

func (h *EditorHandler) DeletePlacement(ctx *gin.Context) {
	userID, ok := ctx.Get("user_id")
	if !ok {
		ctx.Error(apperror.Unauthorized("Thiếu user_id", nil))
		return
	}

	placementID := ctx.Param("id")
	if placementID == "" {
		ctx.Error(apperror.BadRequest("Thiếu ID vật phẩm cần xóa", nil))
		return
	}

	mapCode := ctx.Query("map_code")
	if mapCode == "" {
		ctx.Error(apperror.BadRequest("Thiếu map_code", nil))
		return
	}

	newCoins, err := h.usecase.DeletePlacement(ctx.Request.Context(), userID.(string), mapCode, placementID)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, response.SuccessResponse[gin.H]{
		Success: true,
		Data: gin.H{
			"new_coins": newCoins,
		},
	})
}

func (h *EditorHandler) CoinPickup(ctx *gin.Context) {
	userID, ok := ctx.Get("user_id")
	if !ok {
		ctx.Error(apperror.Unauthorized("Thiếu user_id", nil))
		return
	}

	var input struct {
		MapCode string `json:"map_code" binding:"required"`
		CoinID  string `json:"coin_id" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.Error(apperror.BadRequest("Dữ liệu không hợp lệ", err))
		return
	}

	newCoins, err := h.usecase.ClaimCoinPickup(ctx.Request.Context(), userID.(string), input.MapCode, input.CoinID)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, response.SuccessResponse[gin.H]{
		Success: true,
		Data: gin.H{
			"new_coins": newCoins,
		},
	})
}
