package auth

import (
	"database/sql"

	"backend/internal/module/auth/delivery"
	"backend/internal/platform/config"
)

type AuthModule struct {
	provider *Provider
}

func NewAuthModule(db *sql.DB, jwtSecret string, teamsClientID string, teamsTenantID string, cookieConfig config.CookieConfig) *AuthModule {
	return &AuthModule{provider: NewProvider(db, jwtSecret, teamsClientID, teamsTenantID, delivery.CookieConfig{
		Secure:   cookieConfig.Secure,
		SameSite: cookieConfig.SameSite,
	})}
}
