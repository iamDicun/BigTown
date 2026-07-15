package main

import (
	"backend/internal/app"
	"backend/internal/database"
	"backend/internal/platform/config"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()
	pg := database.NewPostgresDB(cfg.Database)
	defer pg.Close() // Tránh gọi xuyên tầng

	server := app.New(&app.Container{Config: cfg, DB: pg.DB})
	server.Run(":" + cfg.Server.Port)
}
