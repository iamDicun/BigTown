package main

import (
	"log"
	"os"
	"path/filepath"
	"runtime"

	"backend/internal/app"
	"backend/internal/database"
	"backend/internal/platform/config"

	"github.com/joho/godotenv"
)

func main() {
	_, b, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(b), "..", "..")
	envPath := filepath.Join(projectRoot, ".env")

	if _, statErr := os.Stat(envPath); statErr == nil {
		if err := godotenv.Load(envPath); err != nil {
			log.Println("godotenv: cannot load", envPath, err)
		}
	} else {
		_ = godotenv.Load()
	}

	cfg := config.Load()
	pg := database.NewPostgresDB(cfg.Database)
	defer pg.Close() // Tránh gọi xuyên tầng

	server := app.New(&app.Container{Config: cfg, DB: pg.DB})
	server.Run(":" + cfg.Server.Port)
}
