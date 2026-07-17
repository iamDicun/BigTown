package delivery

import (
	"net/http"
	"strconv"

	"backend/internal/apperror"
	"backend/internal/module/chat/entity"
	"backend/internal/module/chat/usecase"
	"backend/internal/response"

	"github.com/gin-gonic/gin"
)

type ChatHandler struct {
	usecase *usecase.ChatUsecase
}

func NewChatHandler(usecase *usecase.ChatUsecase) *ChatHandler {
	return &ChatHandler{usecase: usecase}
}

func (h *ChatHandler) SendMessage(ctx *gin.Context) {
	var input SendChatMessageRequest
	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.Error(apperror.BadRequest("Dữ liệu chat không hợp lệ", err))
		return
	}

	userIDValue, ok := ctx.Get("user_id")
	if !ok {
		ctx.Error(apperror.Unauthorized("Thiếu user_id", nil))
		return
	}

	saved, err := h.usecase.SendMessage(ctx.Request.Context(), usecase.SendMessageInput{
		UserID:  userIDValue.(string),
		RoomID:  ctx.Param("roomId"),
		Message: input.Message,
	})
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusCreated, response.SuccessResponse[ChatMessageResponse]{
		Success: true,
		Data:    toChatMessageResponse(*saved),
	})
}

func (h *ChatHandler) GetMessages(ctx *gin.Context) {
	limit := 0
	if raw := ctx.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			ctx.Error(apperror.BadRequest("limit không hợp lệ", err))
			return
		}
		limit = parsed
	}

	messages, err := h.usecase.ListRecentMessages(ctx.Request.Context(), ctx.Param("roomId"), limit)
	if err != nil {
		ctx.Error(err)
		return
	}

	responses := make([]ChatMessageResponse, 0, len(messages))
	for _, msg := range messages {
		responses = append(responses, toChatMessageResponse(msg))
	}

	ctx.JSON(http.StatusOK, response.SuccessResponse[[]ChatMessageResponse]{Success: true, Data: responses})
}

func toChatMessageResponse(msg entity.ChatMessage) ChatMessageResponse {
	return ChatMessageResponse{
		ID:            msg.ID,
		RoomID:        msg.RoomID,
		CharacterID:   msg.CharacterID,
		CharacterName: msg.CharacterName,
		Message:       msg.Message,
		MessageType:   msg.MessageType,
		CreatedAt:     msg.CreatedAt,
	}
}
