package middleware

import (
	"backend/internal/apperror"
	"fmt"
	"log"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic=%v\n%s", err, debug.Stack())

				c.Error(apperror.Internal(fmt.Errorf("%v", err)))
				c.Abort()
			}
		}()

		c.Next()
	}
}
