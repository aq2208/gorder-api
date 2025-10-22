package http

import (
	"github.com/aq2208/gorder-api/internal/adapter/http/middleware"
	"github.com/gin-gonic/gin"
)

func NewRouter(h *OrderHandler, th *TokenHandler, authz *middleware.Authz) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
	r.POST("/v1/token", th.IssueToken)

	v1 := r.Group("/v1")
	{
		v1.POST("/orders", authz.Require("orders.write"), h.CreateOrder)
		v1.GET("/orders/:id", authz.Require("orders.read"), h.GetOrderByID)
	}

	return r
}
