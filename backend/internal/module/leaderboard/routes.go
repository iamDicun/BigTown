package leaderboard

import (
	"backend/internal/module/leaderboard/delivery"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, handler *delivery.LeaderboardHandler) {
	r.GET("/leaderboard", handler.GetLeaderboard)
}

func (m *LeaderboardModule) RegisterPublicRoutes(r *gin.RouterGroup) {}

func (m *LeaderboardModule) RegisterProtectedRoutes(r *gin.RouterGroup) {
	RegisterRoutes(r, m.provider.Handler())
}
