package apperror

import (
	"fmt"
	"net/http"
)

type AppError struct {
	Code       string
	Message    string
	HTTPStatus int
	Err        error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Code, e.Err)
	}
	return e.Code
}

func New(code string, message string, httpStatus int, err error) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		Err:        err,
	}
}

func BadRequest(message string, err error) *AppError {
	return New("BAD_REQUEST", message, http.StatusBadRequest, err)
}

func Unauthorized(message string, err error) *AppError {
	return New("UNAUTHORIZED", message, http.StatusUnauthorized, err)
}

func TokenMissing(message string, err error) *AppError {
	return New("TOKEN_MISSING", message, http.StatusUnauthorized, err)
}

func TokenInvalid(message string, err error) *AppError {
	return New("TOKEN_INVALID", message, http.StatusUnauthorized, err)
}

func TokenExpired(message string, err error) *AppError {
	return New("TOKEN_EXPIRED", message, http.StatusUnauthorized, err)
}

func TokenRevoked(message string, err error) *AppError {
	return New("TOKEN_REVOKED", message, http.StatusUnauthorized, err)
}

func RefreshTokenInvalid(message string, err error) *AppError {
	return New("REFRESH_TOKEN_INVALID", message, http.StatusUnauthorized, err)
}

func RefreshTokenExpired(message string, err error) *AppError {
	return New("REFRESH_TOKEN_EXPIRED", message, http.StatusUnauthorized, err)
}

func RefreshTokenRevoked(message string, err error) *AppError {
	return New("REFRESH_TOKEN_REVOKED", message, http.StatusUnauthorized, err)
}

func Forbidden(message string, err error) *AppError {
	return New("FORBIDDEN", message, http.StatusForbidden, err)
}

func NotFound(message string, err error) *AppError {
	return New("NOT_FOUND", message, http.StatusNotFound, err)
}

func Internal(err error) *AppError {
	return New("INTERNAL_SERVER_ERROR", "Internal server error", http.StatusInternalServerError, err)
}
