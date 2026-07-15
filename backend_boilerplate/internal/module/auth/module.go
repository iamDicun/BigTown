package auth

import (
	"database/sql"
)

type AuthModule struct {
	provider *Provider
}

func NewAuthModule(db *sql.DB, jwtSecret string) *AuthModule {
	return &AuthModule{provider: NewProvider(db, jwtSecret)}
}
