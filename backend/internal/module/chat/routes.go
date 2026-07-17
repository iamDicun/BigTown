package chat

import (
	"backend/internal/module/chat/delivery"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, handler *delivery.ChatHandler) {
	rooms := r.Group("/rooms/:roomId/chat")
	rooms.GET("/messages", handler.GetMessages)
	rooms.POST("/messages", handler.SendMessage)
}

func (m *ChatModule) RegisterPublicRoutes(r *gin.RouterGroup) {}

func (m *ChatModule) RegisterProtectedRoutes(r *gin.RouterGroup) {
	RegisterRoutes(r, m.provider.Handler())
}
