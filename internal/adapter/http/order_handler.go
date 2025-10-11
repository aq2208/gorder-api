package http

import (
	"context"
	"net/http"
	"time"

	"github.com/aq2208/gorder-api/internal/usecase"
	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	create *usecase.CreateOrder
	query  usecase.OrderRepo
}

func NewOrderHandler(create *usecase.CreateOrder, query usecase.OrderRepo) *OrderHandler {
	return &OrderHandler{create: create, query: query}
}

type createReq struct {
	AmountCents int64  `json:"amount_cents" binding:"required,gt=0"`
	Currency    string `json:"currency" binding:"required"`
	ItemsJSON   string `json:"items_json" binding:"required"`
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	// (Later) get user from JWT middleware; for now use header for demo
	userID := c.GetHeader("X-Demo-User")
	idemKey := c.GetHeader("X-Idempotency-Key")

	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	out, err := h.create.Execute(ctx, usecase.CreateOrderInput{
		UserID:         userID,
		IdempotencyKey: idemKey,
		AmountCents:    req.AmountCents,
		Currency:       req.Currency,
		ItemsJSON:      req.ItemsJSON,
	})
	if err != nil {
		status := http.StatusInternalServerError
		if err == usecase.ErrDuplicate {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"order_id": out.OrderID, "status": out.Status})
}

func (h *OrderHandler) GetOrderByID(c *gin.Context) {
	id := c.Param("id")
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	rec, err := h.query.GetByID(ctx, id)
	if err != nil || rec == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":           rec.ID,
		"user_id":      rec.UserID,
		"status":       rec.Status,
		"amount_cents": rec.AmountCents,
		"currency":     rec.Currency,
		"items_json":   rec.ItemsJSON,
	})
}
