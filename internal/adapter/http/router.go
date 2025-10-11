package http

import "github.com/gin-gonic/gin"

func NewRouter(h *OrderHandler) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	v1 := r.Group("/v1")
	{
		v1.POST("/orders", h.CreateOrder)
		v1.GET("/orders/:id", h.GetOrderByID)
	}

	return r
}
