package example

import (
	"backend/internal/module/example/delivery"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, handler *delivery.ExampleHandler) {
	r.GET("/example/items", handler.GetItems)
	r.GET("/example/items/:id", handler.GetItemByID)
	r.POST("/example/items", handler.CreateItem)
}

func (m *ExampleModule) RegisterPublicRoutes(r *gin.RouterGroup) {}

func (m *ExampleModule) RegisterProtectedRoutes(r *gin.RouterGroup) {
	RegisterRoutes(r, m.provider.Handler())
}
