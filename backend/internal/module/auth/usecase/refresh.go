package usecase

import (
	"context"
	"database/sql"
	"errors"

	"backend/internal/apperror"
	"backend/internal/security"
)

type RefreshInput struct {
	RefreshToken string
}

type RefreshOutput struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresIn    int64
}

func (u *AuthUsecase) Refresh(ctx context.Context, input RefreshInput) (*RefreshOutput, error) {
	refreshTokenHash := security.HashToken(input.RefreshToken)
	storedToken, err := u.authRepo.FindRefreshTokenByHash(ctx, refreshTokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperror.RefreshTokenInvalid("Refresh token không hợp lệ", err)
		}
		return nil, apperror.Internal(err)
	}

	if storedToken.RevokedAt.Valid {
		return nil, apperror.RefreshTokenRevoked("Refresh token đã bị thu hồi", nil)
	}
	if timeNow().After(storedToken.ExpiresAt) {
		return nil, apperror.RefreshTokenExpired("Refresh token đã hết hạn", nil)
	}

	user, err := u.userReader.FindByID(ctx, storedToken.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperror.RefreshTokenInvalid("Refresh token không hợp lệ", err)
		}
		return nil, apperror.Internal(err)
	}

	accessToken, err := security.GenerateToken(user.ID, user.Role, u.jwtSecret, accessTokenTTL)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	newRefreshToken, err := security.GenerateRandomToken()
	if err != nil {
		return nil, apperror.Internal(err)
	}

	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	defer tx.Rollback()

	if err := u.authRepo.RevokeRefreshTokenWithTx(ctx, tx, refreshTokenHash); err != nil {
		return nil, apperror.Internal(err)
	}

	if err := u.authRepo.CreateRefreshTokenWithTx(ctx, tx, user.ID, security.HashToken(newRefreshToken), timeNow().Add(refreshTokenTTL)); err != nil {
		return nil, apperror.Internal(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, apperror.Internal(err)
	}

	return &RefreshOutput{AccessToken: accessToken, RefreshToken: newRefreshToken, TokenType: "Bearer", ExpiresIn: int64(accessTokenTTL.Seconds())}, nil
}
