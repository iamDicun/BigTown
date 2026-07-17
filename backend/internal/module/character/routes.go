package character

import (
	"backend/internal/module/character/delivery"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, handler *delivery.CharacterHandler) {
	r.GET("/characters/me", handler.GetMe)
}

func (m *CharacterModule) RegisterPublicRoutes(r *gin.RouterGroup) {}

func (m *CharacterModule) RegisterProtectedRoutes(r *gin.RouterGroup) {
	RegisterRoutes(r, m.provider.Handler())
}
