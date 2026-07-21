package app

import (
	"backend/internal/middleware"
	"backend/internal/module/auth"
	authrepo "backend/internal/module/auth/repository"
	"backend/internal/module/character"
	"backend/internal/module/chat"
	"backend/internal/module/leaderboard"
	"backend/internal/module/realtime"
	"backend/internal/module/user"
	userrepo "backend/internal/module/user/repository"

	"github.com/gin-gonic/gin"
)

type App struct {
	container *Container
	router    *gin.Engine
}

func New(container *Container) *App {
	router := gin.Default()

	a := &App{container: container, router: router}
	a.registerMiddleware()
	a.registerModules()

	return a
}

func (a *App) Router() *gin.Engine {
	return a.router
}

func (a *App) Run(addr string) error {
	return a.router.Run(addr)
}

func (a *App) registerMiddleware() {
	a.router.Use(middleware.RequestIDMiddleware())
	a.router.Use(middleware.LoggerMiddleware())
	a.router.Use(middleware.CORSMiddleware(a.container.Config.Web.AllowedOrigins))
	a.router.Use(middleware.ErrorHandlerMiddleware())
	a.router.Use(middleware.RecoveryMiddleware())
}

func (a *App) registerModules() {
	defaultMapCode := a.container.Config.Game.DefaultMapCode

	authModule := auth.NewAuthModule(a.container.DB, a.container.Config.Auth.JWTSecret, a.container.Config.Teams.ClientID, a.container.Config.Teams.TenantID, defaultMapCode, a.container.Config.Cookie)

	// characterModule đứng trước realtimeModule vì RealtimeUsecase cần characterModule.Usecase()
	// (implement port.MapReader) để trả bootstrap map thật thay vì hardcode — xem
	// docs/Architecture.md mục 9.1. characterModule.RegisterProtectedRoutes() vẫn gọi ở cuối,
	// route registration không phụ thuộc thứ tự construction.
	//
	// userrepo truyền vào để CharacterUsecase lấy full_name thật của user khi tạo character qua
	// đường an toàn dự phòng (GetOrCreateForUser) — xem character/port/user_reader.go.
	characterModule := character.NewCharacterModule(a.container.DB, userrepo.NewUserRepository(a.container.DB), defaultMapCode)

	// characterModule.Usecase() thỏa mãn cả port.MapReader (GetDefaultMap) lẫn
	// port.CharacterResolver (GetOrCreateForUser) — dùng chung 1 instance cho cả bootstrap
	// map lẫn resolve character khi join room/movement.
	realtimeModule := realtime.NewRealtimeModule(a.container.Config.Auth.JWTSecret, a.container.Config.Web.AllowedOrigins, characterModule.Usecase(), characterModule.Usecase())
	realtimeModule.RegisterConnectionRoute(a.router)

	publicAPI := a.router.Group("/api")
	authModule.RegisterPublicRoutes(publicAPI)

	api := a.router.Group("/api")
	blacklistChecker := authrepo.NewAuthRepository(a.container.DB)
	api.Use(middleware.AuthMiddleware(a.container.Config.Auth.JWTSecret, blacklistChecker))

	chatModule := chat.NewChatModule(a.container.DB, realtimeModule.Transport(), characterModule.Usecase())

	authModule.RegisterProtectedRoutes(api)
	user.NewUserModule(a.container.DB).RegisterProtectedRoutes(api)
	leaderboard.NewLeaderboardModule(a.container.DB).RegisterProtectedRoutes(api)
	realtimeModule.RegisterProtectedRoutes(api)
	characterModule.RegisterProtectedRoutes(api)
	chatModule.RegisterProtectedRoutes(api)
}
