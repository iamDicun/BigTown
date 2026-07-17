package config

import "os"

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Auth     AuthConfig
	Teams    TeamsConfig
	Game     GameConfig
}

type ServerConfig struct {
	Port string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
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
			Port: getEnv("SERVER_PORT", "8080"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			Name:     getEnv("DB_NAME", "asset_management"),
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
	}
}

func getEnv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
