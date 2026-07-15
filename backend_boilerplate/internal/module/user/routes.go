package user

import (
	"backend/internal/middleware"
	"backend/internal/module/user/delivery"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, handler *delivery.UserHandler) {
	r.GET("/users", middleware.RequireRoles("Admin"), handler.GetUsers)
}

func (m *UserModule) RegisterPublicRoutes(r *gin.RouterGroup) {}

func (m *UserModule) RegisterProtectedRoutes(r *gin.RouterGroup) {
	RegisterRoutes(r, m.provider.Handler())
}
