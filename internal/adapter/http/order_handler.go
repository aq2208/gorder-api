package http

import (
	"context"
	"errors"
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

type createOrderReq struct {
	UserID string `json:"userId" binding:"required"`

	Amount struct {
		Cents    int64  `json:"cents" binding:"required,gt=0"`
		Currency string `json:"currency" binding:"required"`
	} `json:"amount" binding:"required"`

	Items string `json:"items" binding:"required"`
}

type createOrderResp struct {
	OrderID string `json:"orderId"`
	Status  string `json:"status"`
}

// CreateOrder handler: translate to use case input
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req createOrderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request"})
		return
	}

	idemKey := c.GetHeader("X-Idempotency-Key") // prevent duplicated requests

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	out, err := h.create.Execute(ctx, usecase.CreateOrderInput{
		UserID:         req.UserID,
		IdempotencyKey: idemKey,
		AmountCents:    req.Amount.Cents,
		Currency:       req.Amount.Currency,
		ItemsJSON:      req.Items,
	})

	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, usecase.ErrDuplicate) {
			status = http.StatusConflict
		}
		if errors.Is(err, usecase.ErrInvalidAmount) {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, createOrderResp{
		OrderID: out.OrderID,
		Status:  out.Status,
	})
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
