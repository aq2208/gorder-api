package queue

import (
	"context"

	"github.com/aq2208/gorder-api/internal/usecase"
)

// OrderGateway is the port to your order-gw (gRPC) client.
// Implement this in your gateways adapter (e.g., using generated protobuf client).
type OrderGateway interface {
	CreateOrder(ctx context.Context, orderID, userID string, cents int64, currency string) error
}

// OrderCreatedHandler forwards the event to order-gw via gRPC.
type OrderCreatedHandler struct {
	GW OrderGateway
}

func NewOrderCreatedHandler(gw OrderGateway) *OrderCreatedHandler {
	return &OrderCreatedHandler{GW: gw}
}

// HandleCreate is intended to be used with the JSON adapter (queue.JSONHandler[CreatedMsg]).
func (h *OrderCreatedHandler) HandleCreate(ctx context.Context, msg usecase.CreatedMsg) error {
	// Single responsibility: call the downstream gRPC.
	return h.GW.CreateOrder(ctx, msg.OrderID, msg.UserID, msg.Cents, msg.Currency)
}
