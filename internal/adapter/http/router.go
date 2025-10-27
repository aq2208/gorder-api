package http

import (
	"github.com/aq2208/gorder-api/internal/adapter/http/middleware"
	"github.com/aq2208/gorder-api/internal/logging"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewRouter(h *OrderHandler, th *TokenHandler, authz *middleware.Authz, cv *middleware.CryptoVerify) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), middleware.MetricsMiddleware())

	logging.Init("order-api", "./logs/app.log")
	l := logging.New("http")
	r.Use(middleware.Logging(l))

	r.GET("/healthz", func(c *gin.Context) {
		logging.From(c).Info("health check")
		c.JSON(200, gin.H{"ok": true})
	})
	// Prometheus endpoint (scraped by Prometheus)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	r.POST("/v1/token", th.IssueToken)

	r.POST("/_test/encrypt-sign", cv.EncryptAndSign())
	r.POST("/_test/encrypt-sign-text", cv.EncryptAndSignText())

	v1 := r.Group("/v1")
	{
		v1.POST("/orders", authz.Require("orders.write"), cv.CryptoVerify(), h.CreateOrder)
		v1.GET("/orders/:id", authz.Require("orders.read"), cv.CryptoVerify(), h.GetOrderByID)
	}

	return r
}
