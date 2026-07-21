package config

import (
	"os"
	"strings"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Auth     AuthConfig
	Teams    TeamsConfig
	Game     GameConfig
	Web      WebConfig
	Cookie   CookieConfig
}

type ServerConfig struct {
	Port string
}

type WebConfig struct {
	AllowedOrigins []string
}

type CookieConfig struct {
	Secure   bool
	SameSite string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type AuthConfig struct {
	JWTSecret string
}

type TeamsConfig struct {
	ClientID string
	TenantID string
}

// GameConfig.DefaultMapCode là điểm cấu hình duy nhất để đổi map hiện tại của MVP
// (xem docs/Architecture.md mục 9.1). Character mới/cũ đều được đồng bộ map_id theo giá trị này.
type GameConfig struct {
	DefaultMapCode string
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", getEnv("SERVER_PORT", "8080")),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5433"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			Name:     getEnv("DB_NAME", "app_db"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Auth: AuthConfig{
			JWTSecret: getEnv("JWT_SECRET", "dev-secret-change-me"),
		},
		Teams: TeamsConfig{
			ClientID: getEnv("TEAMS_CLIENT_ID", ""),
			TenantID: getEnv("TEAMS_TENANT_ID", "common"),
		},
		Game: GameConfig{
			DefaultMapCode: getEnv("GAME_DEFAULT_MAP_CODE", "village_adventure"),
		},
		Web: WebConfig{
			AllowedOrigins: getCSVEnv("CORS_ALLOWED_ORIGINS", []string{"http://localhost:5173"}),
		},
		Cookie: CookieConfig{
			Secure:   getBoolEnv("COOKIE_SECURE", false),
			SameSite: getEnv("COOKIE_SAME_SITE", "Lax"),
		},
	}
}

func getEnv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getCSVEnv(key string, fallback []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			items = append(items, item)
		}
	}
	if len(items) == 0 {
		return fallback
	}
	return items
}

func getBoolEnv(key string, fallback bool) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	return value == "true" || value == "1" || value == "yes"
}
