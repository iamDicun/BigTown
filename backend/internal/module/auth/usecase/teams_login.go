package usecase

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"backend/internal/apperror"
	"backend/internal/module/auth/entity"
)

const teamsIdentityProvider = "teams"

type TeamsLoginInput struct {
	SSOToken string
}

type TeamsLoginOutput struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresIn    int64
}

func (u *AuthUsecase) TeamsLogin(ctx context.Context, input TeamsLoginInput) (*TeamsLoginOutput, error) {
	ssoToken := strings.TrimSpace(input.SSOToken)
	if ssoToken == "" {
		return nil, apperror.BadRequest("Thiếu Teams SSO token", nil)
	}

	claims, err := u.teamsTokenVerifier.Verify(ctx, ssoToken)
	if err != nil {
		return nil, apperror.Unauthorized("Teams SSO token không hợp lệ", err)
	}

	identity, err := u.authRepo.FindUserIdentity(ctx, teamsIdentityProvider, claims.TenantID, claims.ExternalSubject)
	if err == nil {
		user, err := u.userReader.FindByID(ctx, identity.UserID)
		if err != nil {
			return nil, apperror.Internal(err)
		}
		return u.loginOutputFromUser(ctx, user.ID, user.Role)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.Internal(err)
	}

	email := strings.ToLower(strings.TrimSpace(claims.Email))
	if email == "" {
		return nil, apperror.BadRequest("Teams token không có email hợp lệ", nil)
	}

	fullName := strings.TrimSpace(claims.FullName)
	if fullName == "" {
		fullName = email
	}

	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	defer tx.Rollback()

	user, err := u.userReader.FindByEmail(ctx, email)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, apperror.Internal(err)
		}

		user, err = u.userReader.CreateWithTx(ctx, tx, fullName, email)
		if err != nil {
			return nil, apperror.BadRequest("Không thể tạo user từ Teams SSO", err)
		}
	}

	if err := u.authRepo.CreateUserIdentityWithTx(ctx, tx, entity.UserIdentity{
		UserID:          user.ID,
		Provider:        teamsIdentityProvider,
		ExternalSubject: claims.ExternalSubject,
		TenantID:        claims.TenantID,
		Email:           email,
	}); err != nil {
		return nil, apperror.Internal(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, apperror.Internal(err)
	}

	return u.loginOutputFromUser(ctx, user.ID, user.Role)
}

func (u *AuthUsecase) loginOutputFromUser(ctx context.Context, userID string, role string) (*TeamsLoginOutput, error) {
	tokenPair, err := u.generateTokenPair(ctx, userID, role)
	if err != nil {
		return nil, err
	}

	return &TeamsLoginOutput{AccessToken: tokenPair.AccessToken, RefreshToken: tokenPair.RefreshToken, TokenType: "Bearer", ExpiresIn: tokenPair.ExpiresIn}, nil
}
