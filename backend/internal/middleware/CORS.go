package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func CORSMiddleware(allowedOrigins []string) gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins: allowedOrigins, // FE nào được phép gọi API
		AllowMethods: []string{ // method nào được phép dùng
			"GET",
			"POST",
			"PUT",
			"PATCH",
			"DELETE",
			"OPTIONS",
		},
		AllowHeaders: []string{ // Request header nào Fe được phép gửi
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
		},
		ExposeHeaders: []string{ // response header nào FE được phép đọc
			"Content-Length",
		},
		AllowCredentials: true,           // có cho gửi cookie/auth credential không
		MaxAge:           12 * time.Hour, // browser casche kết quả preflight bao lâu
	})
}
