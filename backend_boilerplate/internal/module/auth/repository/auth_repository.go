package repository

import (
	"context"
	"database/sql"
	"time"

	"backend/internal/module/auth/entity"
	"backend/internal/module/auth/port"
)

type AuthRepository struct {
	db *sql.DB
}

func NewAuthRepository(db *sql.DB) *AuthRepository {
	return &AuthRepository{db: db}
}

var _ port.AuthRepository = (*AuthRepository)(nil)

func (r *AuthRepository) CreateCredentialWithTx(ctx context.Context, tx *sql.Tx, userID string, passwordHash string) error {
	_, err := tx.ExecContext(
		ctx,
		`INSERT INTO credential (user_id, password_hash) VALUES ($1, $2)`,
		userID,
		passwordHash,
	)
	return err
}

func (r *AuthRepository) FindCredentialByUserID(ctx context.Context, userID string) (*entity.Credential, error) {
	var credential entity.Credential
	err := r.db.QueryRowContext(
		ctx,
		`SELECT user_id::text, password_hash FROM credential WHERE user_id = $1`,
		userID,
	).Scan(&credential.UserID, &credential.PasswordHash)
	if err != nil {
		return nil, err
	}
	return &credential, nil
}

func (r *AuthRepository) UpdatePasswordHash(ctx context.Context, userID string, passwordHash string) error {
	_, err := r.db.ExecContext(
		ctx,
		`UPDATE credential SET password_hash = $2, updated_at = CURRENT_TIMESTAMP WHERE user_id = $1`,
		userID,
		passwordHash,
	)
	return err
}

func (r *AuthRepository) CreateRefreshToken(ctx context.Context, userID string, tokenHash string, expiresAt time.Time) error {
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO refresh_token (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID,
		tokenHash,
		expiresAt,
	)
	return err
}

func (r *AuthRepository) CreateRefreshTokenWithTx(ctx context.Context, tx *sql.Tx, userID string, tokenHash string, expiresAt time.Time) error {
	_, err := tx.ExecContext(
		ctx,
		`INSERT INTO refresh_token (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID,
		tokenHash,
		expiresAt,
	)
	return err
}

func (r *AuthRepository) FindRefreshTokenByHash(ctx context.Context, tokenHash string) (*entity.RefreshToken, error) {
	var token entity.RefreshToken
	err := r.db.QueryRowContext(
		ctx,
		`SELECT id::text, user_id::text, token_hash, expires_at, revoked_at
		 FROM refresh_token
		 WHERE token_hash = $1`,
		tokenHash,
	).Scan(&token.ID, &token.UserID, &token.TokenHash, &token.ExpiresAt, &token.RevokedAt)
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *AuthRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	_, err := r.db.ExecContext(
		ctx,
		`UPDATE refresh_token SET revoked_at = CURRENT_TIMESTAMP WHERE token_hash = $1 AND revoked_at IS NULL`,
		tokenHash,
	)
	return err
}

func (r *AuthRepository) RevokeRefreshTokenWithTx(ctx context.Context, tx *sql.Tx, tokenHash string) error {
	_, err := tx.ExecContext(
		ctx,
		`UPDATE refresh_token SET revoked_at = CURRENT_TIMESTAMP WHERE token_hash = $1 AND revoked_at IS NULL`,
		tokenHash,
	)
	return err
}

func (r *AuthRepository) BlacklistAccessToken(ctx context.Context, tokenHash string, expiresAt time.Time) error {
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO token_blacklist (token_hash, expires_at)
		 VALUES ($1, $2)
		 ON CONFLICT (token_hash) DO NOTHING`,
		tokenHash,
		expiresAt,
	)
	return err
}

func (r *AuthRepository) IsAccessTokenBlacklisted(tokenHash string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(
		`SELECT EXISTS(
			SELECT 1 FROM token_blacklist
			WHERE token_hash = $1 AND expires_at > CURRENT_TIMESTAMP
		)`,
		tokenHash,
	).Scan(&exists)
	return exists, err
}
