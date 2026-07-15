package auth

import (
	"backend/internal/module/auth/delivery"

	"github.com/gin-gonic/gin"
)

func RegisterPublicRoutes(r *gin.RouterGroup, handler *delivery.AuthHandler) {
	r.POST("/auth/register", handler.Register)
	r.POST("/auth/login", handler.Login)
	r.POST("/auth/refresh", handler.Refresh)
}

func RegisterProtectedRoutes(r *gin.RouterGroup, handler *delivery.AuthHandler) {
	r.POST("/auth/logout", handler.Logout)
}

func (m *AuthModule) RegisterPublicRoutes(r *gin.RouterGroup) {
	RegisterPublicRoutes(r, m.provider.Handler())
}

func (m *AuthModule) RegisterProtectedRoutes(r *gin.RouterGroup) {
	RegisterProtectedRoutes(r, m.provider.Handler())
}
