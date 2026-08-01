package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

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
	defer pg.Close()

	server := app.New(&app.Container{Config: cfg, DB: pg.DB})

	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: server.Router(),
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	log.Printf("Server is running on port %s", cfg.Server.Port)

	// Chờ tín hiệu graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Tín hiệu tắt server nhận được, đang graceful shutdown...")

	// Dừng nhận HTTP request mới
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Lỗi khi shutdown HTTP server: %v", err)
	}

	// Shutdown application (flush RoomManager/Writer)
	server.Shutdown()
	log.Println("Graceful shutdown hoàn tất!")
}
