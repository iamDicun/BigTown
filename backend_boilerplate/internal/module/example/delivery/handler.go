package delivery

import (
	"net/http"

	"backend/internal/apperror"
	"backend/internal/module/example/entity"
	"backend/internal/module/example/usecase"
	"backend/internal/response"

	"github.com/gin-gonic/gin"
)

type ExampleHandler struct {
	usecase *usecase.ExampleUsecase
}

func NewExampleHandler(usecase *usecase.ExampleUsecase) *ExampleHandler {
	return &ExampleHandler{usecase: usecase}
}

func (h *ExampleHandler) GetItems(ctx *gin.Context) {
	items, err := h.usecase.GetItems(ctx.Request.Context())
	if err != nil {
		ctx.Error(err)
		return
	}

	responses := make([]ItemResponse, 0, len(items))
	for _, it := range items {
		responses = append(responses, itemResponseFromEntity(it))
	}

	ctx.JSON(http.StatusOK, response.SuccessResponse[[]ItemResponse]{Success: true, Data: responses})
}

func (h *ExampleHandler) GetItemByID(ctx *gin.Context) {
	id := ctx.Param("id")

	item, err := h.usecase.GetItemByID(ctx.Request.Context(), id)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, response.SuccessResponse[ItemResponse]{Success: true, Data: itemResponseFromEntity(*item)})
}

func (h *ExampleHandler) CreateItem(ctx *gin.Context) {
	var input CreateItemRequest
	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.Error(apperror.BadRequest("Dữ liệu không hợp lệ", err))
		return
	}

	item, err := h.usecase.CreateItem(ctx.Request.Context(), usecase.CreateItemInput{Name: input.Name})
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusCreated, response.SuccessResponse[ItemResponse]{Success: true, Data: itemResponseFromEntity(*item)})
}

func itemResponseFromEntity(it entity.Item) ItemResponse {
	return ItemResponse{ID: it.ID, Name: it.Name, Status: it.Status}
}
