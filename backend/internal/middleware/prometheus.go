package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var httpRequestsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests processed by the backend.",
	},
	[]string{"method", "path", "status"},
)

func PrometheusMetrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		httpRequestsTotal.WithLabelValues(c.Request.Method, c.FullPath(), httpStatus(c.Writer.Status())).Inc()
	}
}

func httpStatus(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	default:
		return "5xx"
	}
}
