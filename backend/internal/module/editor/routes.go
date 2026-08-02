package editor

import (
	"backend/internal/module/editor/delivery"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, handler *delivery.EditorHandler) {
	r.GET("/editor", handler.GetEditorData)
	r.POST("/editor/place", handler.PlaceItem)
	r.DELETE("/editor/place/:id", handler.DeletePlacement)
	r.POST("/editor/coin-pickup", handler.CoinPickup)
}

func (m *EditorModule) RegisterPublicRoutes(r *gin.RouterGroup) {}

func (m *EditorModule) RegisterProtectedRoutes(r *gin.RouterGroup) {
	RegisterRoutes(r, m.provider.Handler())
}
