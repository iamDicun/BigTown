package auth

import (
	"database/sql"
)

type AuthModule struct {
	provider *Provider
}

func NewAuthModule(db *sql.DB, jwtSecret string, teamsClientID string, teamsTenantID string, defaultMapCode string) *AuthModule {
	return &AuthModule{provider: NewProvider(db, jwtSecret, teamsClientID, teamsTenantID, defaultMapCode)}
}
