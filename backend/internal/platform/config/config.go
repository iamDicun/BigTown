package config

import "os"

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Auth     AuthConfig
	Teams    TeamsConfig
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
	}
}

func getEnv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
