package usecase

import (
	"database/sql"
	"time"

	"backend/internal/module/auth/port"
)

const accessTokenTTL = 15 * time.Minute
const refreshTokenTTL = 7 * 24 * time.Hour

type AuthUsecase struct {
	db         *sql.DB
	authRepo   port.AuthRepository
	userReader port.UserReader
	jwtSecret  string
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

func NewAuthUsecase(db *sql.DB, authRepo port.AuthRepository, userReader port.UserReader, jwtSecret string) *AuthUsecase {
	return &AuthUsecase{db: db, authRepo: authRepo, userReader: userReader, jwtSecret: jwtSecret}
}
