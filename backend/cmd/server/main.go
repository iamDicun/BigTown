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
	// godotenv.Load() không có arg dùng os.Getwd() — trên Windows/IDE path có thể khác
	// thư mục backend/. Thử load từ thư mục chứa source file này trước, rồi fallback
	// về working directory.
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
