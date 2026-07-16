package realtime

import (
	"backend/internal/module/realtime/delivery"
	"backend/internal/module/realtime/transport"

	"github.com/gin-gonic/gin"
)

func RegisterConnectionRoute(r *gin.Engine, realtimeTransport *transport.CentrifugeTransport) {
	r.GET("/connection/websocket", gin.WrapH(realtimeTransport.Handler()))
}

func RegisterProtectedRoutes(r *gin.RouterGroup, handler *delivery.RealtimeHandler) {
	realtime := r.Group("/realtime")
	realtime.GET("/bootstrap", handler.GetBootstrap)
}

func (m *RealtimeModule) RegisterConnectionRoute(r *gin.Engine) {
	RegisterConnectionRoute(r, m.provider.Transport())
}

func (m *RealtimeModule) RegisterPublicRoutes(r *gin.RouterGroup) {}

func (m *RealtimeModule) RegisterProtectedRoutes(r *gin.RouterGroup) {
	RegisterProtectedRoutes(r, m.provider.Handler())
}
