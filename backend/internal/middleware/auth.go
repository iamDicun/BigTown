package middleware

import (
	"errors"
	"strings"

	"backend/internal/apperror"
	"backend/internal/security"

	"github.com/gin-gonic/gin"
)

type TokenBlacklistChecker interface {
	IsAccessTokenBlacklisted(tokenHash string) (bool, error)
}

func AuthMiddleware(jwtSecret string, blacklistChecker TokenBlacklistChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		authorization := c.GetHeader("Authorization")
		if authorization == "" {
			c.Error(apperror.TokenMissing("Thiếu token", nil))
			c.Abort()
			return
		}

		parts := strings.SplitN(authorization, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.Error(apperror.TokenInvalid("Token không hợp lệ", nil))
			c.Abort()
			return
		}

		accessToken := parts[1]
		claims, err := security.ParseToken(accessToken, jwtSecret)
		if err != nil {
			if errors.Is(err, security.ErrTokenExpired) {
				c.Error(apperror.TokenExpired("Token đã hết hạn", err))
			} else {
				c.Error(apperror.TokenInvalid("Token không hợp lệ", err))
			}
			c.Abort()
			return
		}

		blacklisted, err := blacklistChecker.IsAccessTokenBlacklisted(security.HashToken(accessToken))
		if err != nil {
			c.Error(apperror.Internal(err))
			c.Abort()
			return
		}
		if blacklisted {
			c.Error(apperror.TokenRevoked("Token đã bị thu hồi", nil))
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Set("access_token", accessToken)
		if claims.ExpiresAt != nil {
			c.Set("access_token_expires_at", claims.ExpiresAt.Time)
		}
		c.Next()
	}
}

func RequireRoles(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, ok := c.Get("role")
		if !ok {
			c.Error(apperror.Forbidden("Không có quyền truy cập", nil))
			c.Abort()
			return
		}

		userRole, ok := role.(string)
		if !ok {
			c.Error(apperror.Forbidden("Role không hợp lệ", nil))
			c.Abort()
			return
		}

		for _, allowedRole := range allowedRoles {
			if userRole == allowedRole {
				c.Next()
				return
			}
		}

		c.Error(apperror.Forbidden("Không có quyền truy cập", nil))
		c.Abort()
	}
}
