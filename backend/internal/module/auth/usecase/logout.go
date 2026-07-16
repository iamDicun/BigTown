package usecase

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"backend/internal/apperror"
	"backend/internal/security"
)

type LogoutInput struct {
	RefreshToken string
}

type LogoutOutput struct {
	Message string
}

func (u *AuthUsecase) Logout(ctx context.Context, input LogoutInput, accessToken string, accessTokenExpiresAt time.Time) (*LogoutOutput, error) {
	if accessToken == "" {
		return nil, apperror.TokenMissing("Thiếu access token", nil)
	}

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

	if err := u.authRepo.BlacklistAccessToken(ctx, security.HashToken(accessToken), accessTokenExpiresAt); err != nil {
		return nil, apperror.Internal(err)
	}

	if err := u.authRepo.RevokeRefreshToken(ctx, refreshTokenHash); err != nil {
		return nil, apperror.Internal(err)
	}

	return &LogoutOutput{Message: "Đăng xuất thành công"}, nil
}
