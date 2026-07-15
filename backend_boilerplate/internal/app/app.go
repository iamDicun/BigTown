package app

import (
	"backend/internal/middleware"
	"backend/internal/module/auth"
	authrepo "backend/internal/module/auth/repository"
	"backend/internal/module/game"
	"backend/internal/module/user"

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
	a.router.Use(middleware.CORSMiddleware())
	a.router.Use(middleware.ErrorHandlerMiddleware())
	a.router.Use(middleware.RecoveryMiddleware())
}

func (a *App) registerModules() {
	authModule := auth.NewAuthModule(a.container.DB, a.container.Config.Auth.JWTSecret)

	publicAPI := a.router.Group("/api")
	authModule.RegisterPublicRoutes(publicAPI)

	api := a.router.Group("/api")
	blacklistChecker := authrepo.NewAuthRepository(a.container.DB)
	api.Use(middleware.AuthMiddleware(a.container.Config.Auth.JWTSecret, blacklistChecker))

	authModule.RegisterProtectedRoutes(api)
	user.NewUserModule(a.container.DB).RegisterProtectedRoutes(api)
	game.NewGameModule(a.container.DB).RegisterProtectedRoutes(api)
}
