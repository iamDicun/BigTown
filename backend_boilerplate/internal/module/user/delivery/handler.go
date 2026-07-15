package delivery

import (
	"net/http"

	"backend/internal/module/user/usecase"
	"backend/internal/response"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	usecase *usecase.UserUsecase
}

func NewUserHandler(usecase *usecase.UserUsecase) *UserHandler {
	return &UserHandler{usecase: usecase}
}

func (h *UserHandler) GetUsers(ctx *gin.Context) {
	users, err := h.usecase.GetUsers(ctx.Request.Context())
	if err != nil {
		ctx.Error(err)
		return
	}

	responses := make([]UserResponse, 0, len(users))
	for _, u := range users {
		responses = append(responses, UserResponse{ID: u.ID, FullName: u.FullName, Email: u.Email, Role: u.Role})
	}

	ctx.JSON(http.StatusOK, response.SuccessResponse[[]UserResponse]{Success: true, Data: responses})
}
