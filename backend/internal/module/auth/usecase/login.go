package usecase

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"backend/internal/apperror"
	"backend/internal/security"
)

type LoginInput struct {
	Email    string
	Password string
}

type LoginOutput struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresIn    int64
}

func (u *AuthUsecase) Login(ctx context.Context, input LoginInput) (*LoginOutput, error) {
	user, err := u.userReader.FindByEmail(ctx, strings.ToLower(strings.TrimSpace(input.Email)))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperror.Unauthorized("Email hoặc mật khẩu không đúng", err)
		}
		return nil, apperror.Internal(err)
	}

	credential, err := u.authRepo.FindCredentialByUserID(ctx, user.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperror.Unauthorized("Email hoặc mật khẩu không đúng", err)
		}
		return nil, apperror.Internal(err)
	}

	if err := security.CheckPassword(input.Password, credential.PasswordHash); err != nil {
		return nil, apperror.Unauthorized("Email hoặc mật khẩu không đúng", err)
	}

	tokenPair, err := u.generateTokenPair(ctx, user.ID, user.Role)
	if err != nil {
		return nil, err
	}

	return &LoginOutput{AccessToken: tokenPair.AccessToken, RefreshToken: tokenPair.RefreshToken, TokenType: "Bearer", ExpiresIn: tokenPair.ExpiresIn}, nil
}

func (u *AuthUsecase) generateTokenPair(ctx context.Context, userID string, role string) (*TokenPair, error) {
	accessToken, err := security.GenerateToken(userID, role, u.jwtSecret, accessTokenTTL)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	refreshToken, err := security.GenerateRandomToken()
	if err != nil {
		return nil, apperror.Internal(err)
	}

	if err := u.authRepo.CreateRefreshToken(ctx, userID, security.HashToken(refreshToken), timeNow().Add(refreshTokenTTL)); err != nil {
		return nil, apperror.Internal(err)
	}

	return &TokenPair{AccessToken: accessToken, RefreshToken: refreshToken, ExpiresIn: int64(accessTokenTTL.Seconds())}, nil
}
