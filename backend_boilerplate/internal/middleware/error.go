package middleware

import (
	"errors"
	"log"

	"github.com/gin-gonic/gin"

	"backend/internal/apperror"
	"backend/internal/response"
)

func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		err := c.Errors.Last().Err

		var appErr *apperror.AppError
		if !errors.As(err, &appErr) {
			appErr = apperror.Internal(err)
		}

		requestID, _ := c.Get("requestID")

		if appErr.Err != nil {
			log.Printf(
				"level=ERROR request_id=%v code=%s err=%v",
				requestID,
				appErr.Code,
				appErr.Err,
			)
		}

		c.JSON(appErr.HTTPStatus, response.ErrorResponse{
			Success:   false,
			Code:      appErr.Code,
			Message:   appErr.Message,
			RequestID: requestID,
		})
	}
}
