package port

import (
	"context"
	"database/sql"
	"time"

	"backend/internal/module/auth/entity"
)

type AuthRepository interface {
	CreateCredentialWithTx(ctx context.Context, tx *sql.Tx, userID string, passwordHash string) error
	FindCredentialByUserID(ctx context.Context, userID string) (*entity.Credential, error)
	UpdatePasswordHash(ctx context.Context, userID string, passwordHash string) error
	CreateRefreshToken(ctx context.Context, userID string, tokenHash string, expiresAt time.Time) error
	CreateRefreshTokenWithTx(ctx context.Context, tx *sql.Tx, userID string, tokenHash string, expiresAt time.Time) error
	FindRefreshTokenByHash(ctx context.Context, tokenHash string) (*entity.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	RevokeRefreshTokenWithTx(ctx context.Context, tx *sql.Tx, tokenHash string) error
	BlacklistAccessToken(ctx context.Context, tokenHash string, expiresAt time.Time) error
	IsAccessTokenBlacklisted(tokenHash string) (bool, error)
}
