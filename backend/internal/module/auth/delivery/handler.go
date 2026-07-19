package delivery

import (
	"net/http"
	"time"

	"backend/internal/apperror"
	"backend/internal/module/auth/usecase"
	"backend/internal/response"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	usecase      *usecase.AuthUsecase
	cookieConfig CookieConfig
}

type CookieConfig struct {
	Secure   bool
	SameSite string
}

const refreshTokenCookieName = "refresh_token"
const refreshTokenCookieMaxAge = 7 * 24 * 60 * 60

func NewAuthHandler(usecase *usecase.AuthUsecase, cookieConfig CookieConfig) *AuthHandler {
	return &AuthHandler{usecase: usecase, cookieConfig: cookieConfig}
}

func (c *AuthHandler) Register(ctx *gin.Context) {
	var input RegisterRequest
	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.Error(apperror.BadRequest("Dữ liệu đăng ký không hợp lệ", err))
		return
	}

	result, err := c.usecase.Register(ctx.Request.Context(), usecase.RegisterInput{
		FullName: input.FullName,
		Email:    input.Email,
		Password: input.Password,
	})
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusCreated, response.SuccessResponse[*RegisterResponse]{
		Success: true,
		Data: &RegisterResponse{
			ID:       result.ID,
			FullName: result.FullName,
			Email:    result.Email,
			Role:     result.Role,
		},
	})
}

func (c *AuthHandler) Login(ctx *gin.Context) {
	var input LoginRequest
	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.Error(apperror.BadRequest("Dữ liệu đăng nhập không hợp lệ", err))
		return
	}

	result, err := c.usecase.Login(ctx.Request.Context(), usecase.LoginInput{Email: input.Email, Password: input.Password})
	if err != nil {
		ctx.Error(err)
		return
	}

	c.setRefreshTokenCookie(ctx, result.RefreshToken)

	ctx.JSON(http.StatusOK, response.SuccessResponse[*LoginResponse]{
		Success: true,
		Data: &LoginResponse{
			AccessToken: result.AccessToken,
			TokenType:   result.TokenType,
			ExpiresIn:   result.ExpiresIn,
		},
	})
}

func (c *AuthHandler) TeamsLogin(ctx *gin.Context) {
	var input TeamsLoginRequest
	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.Error(apperror.BadRequest("Dữ liệu Teams SSO không hợp lệ", err))
		return
	}

	result, err := c.usecase.TeamsLogin(ctx.Request.Context(), usecase.TeamsLoginInput{SSOToken: input.SSOToken})
	if err != nil {
		ctx.Error(err)
		return
	}

	c.setRefreshTokenCookie(ctx, result.RefreshToken)

	ctx.JSON(http.StatusOK, response.SuccessResponse[*LoginResponse]{
		Success: true,
		Data: &LoginResponse{
			AccessToken: result.AccessToken,
			TokenType:   result.TokenType,
			ExpiresIn:   result.ExpiresIn,
		},
	})
}

func (c *AuthHandler) Refresh(ctx *gin.Context) {
	refreshToken, err := readRefreshTokenCookie(ctx)
	if err != nil {
		ctx.Error(err)
		return
	}

	result, err := c.usecase.Refresh(ctx.Request.Context(), usecase.RefreshInput{RefreshToken: refreshToken})
	if err != nil {
		ctx.Error(err)
		return
	}

	c.setRefreshTokenCookie(ctx, result.RefreshToken)

	ctx.JSON(http.StatusOK, response.SuccessResponse[*RefreshResponse]{
		Success: true,
		Data: &RefreshResponse{
			AccessToken: result.AccessToken,
			TokenType:   result.TokenType,
			ExpiresIn:   result.ExpiresIn,
		},
	})
}

func (c *AuthHandler) Logout(ctx *gin.Context) {
	refreshToken, err := readRefreshTokenCookie(ctx)
	if err != nil {
		ctx.Error(err)
		return
	}

	accessTokenValue, ok := ctx.Get("access_token")
	if !ok {
		ctx.Error(apperror.Unauthorized("Thiếu access token", nil))
		return
	}

	accessToken, ok := accessTokenValue.(string)
	if !ok {
		ctx.Error(apperror.Unauthorized("Access token không hợp lệ", nil))
		return
	}

	expiresAtValue, ok := ctx.Get("access_token_expires_at")
	if !ok {
		ctx.Error(apperror.Unauthorized("Access token không hợp lệ", nil))
		return
	}

	expiresAt, ok := expiresAtValue.(time.Time)
	if !ok {
		ctx.Error(apperror.Unauthorized("Access token không hợp lệ", nil))
		return
	}

	result, err := c.usecase.Logout(ctx.Request.Context(), usecase.LogoutInput{RefreshToken: refreshToken}, accessToken, expiresAt)
	if err != nil {
		ctx.Error(err)
		return
	}

	c.clearRefreshTokenCookie(ctx)

	ctx.JSON(http.StatusOK, response.SuccessResponse[*LogoutResponse]{
		Success: true,
		Data:    &LogoutResponse{Message: result.Message},
	})
}

func (c *AuthHandler) setRefreshTokenCookie(ctx *gin.Context, refreshToken string) {
	ctx.SetSameSite(c.cookieSameSite())
	ctx.SetCookie(refreshTokenCookieName, refreshToken, refreshTokenCookieMaxAge, "/api/auth", "", c.cookieConfig.Secure, true)
}

func (c *AuthHandler) clearRefreshTokenCookie(ctx *gin.Context) {
	ctx.SetSameSite(c.cookieSameSite())
	ctx.SetCookie(refreshTokenCookieName, "", -1, "/api/auth", "", c.cookieConfig.Secure, true)
}

func (c *AuthHandler) cookieSameSite() http.SameSite {
	switch c.cookieConfig.SameSite {
	case "Strict", "strict":
		return http.SameSiteStrictMode
	case "None", "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

func readRefreshTokenCookie(ctx *gin.Context) (string, error) {
	refreshToken, err := ctx.Cookie(refreshTokenCookieName)
	if err != nil {
		return "", apperror.RefreshTokenInvalid("Thiếu refresh token", err)
	}
	return refreshToken, nil
}
