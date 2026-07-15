package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()

		level := "INFO"
		if status >= 500 {
			level = "ERROR"
		} else if status >= 400 {
			level = "WARN"
		}

		requestID, _ := c.Get("requestID")
		userID, _ := c.Get("user_id")

		fmt.Printf(
			"level=%s request_id=%v method=%s path=%s status=%d duration_ms=%d user_id=%v ip=%s\n",
			level,
			requestID,
			c.Request.Method,
			c.FullPath(),
			status,
			duration.Milliseconds(),
			userID,
			c.ClientIP(),
		)
	}
}
