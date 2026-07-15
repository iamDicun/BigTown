package game

import (
	"backend/internal/module/game/delivery"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, handler *delivery.GameHandler) {
	game := r.Group("/game")
	game.GET("/bootstrap", handler.GetBootstrap)
	game.GET("/leaderboard", handler.GetLeaderboard)
}

func (m *GameModule) RegisterPublicRoutes(r *gin.RouterGroup) {}

func (m *GameModule) RegisterProtectedRoutes(r *gin.RouterGroup) {
	RegisterRoutes(r, m.provider.Handler())
}
