package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_ms",
			Help:    "Duration of HTTP requests in ms",
			Buckets: []float64{5, 10, 25, 50, 100, 200, 400, 800, 1600},
		},
		[]string{"method", "path"},
	)
)

func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := float64(time.Since(start).Milliseconds())
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		httpRequests.WithLabelValues(c.Request.Method, path,
			http.StatusText(c.Writer.Status())).Inc()
		httpDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
	}
}
