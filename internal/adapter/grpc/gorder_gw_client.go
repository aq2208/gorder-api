package grpc

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	gwpb "github.com/aq2208/gorder-api/internal/generated"
)

// OrderGWClient implements the port used by your order_created_handler.
type OrderGWClient struct {
	cli     gwpb.OrderServiceClient
	timeout time.Duration
	ua      string
}

func NewOrderGWClientFromConn(conn *grpc.ClientConn, timeout time.Duration, userAgent string) *OrderGWClient {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &OrderGWClient{
		cli:     gwpb.NewOrderServiceClient(conn),
		timeout: timeout,
		ua:      userAgent,
	}
}

func (c *OrderGWClient) CreateOrder(ctx context.Context, orderID, userID string, cents int64, currency string) error {
	// ensure per-call timeout if caller didn't set one
	if _, ok := ctx.Deadline(); !ok && c.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	// optional metadata (helpful for tracing)
	if c.ua != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "user-agent", c.ua)
	}

	_, err := c.cli.CreateOrder(ctx, &gwpb.CreateOrderRequest{
		OrderId:     orderID,
		UserId:      userID,
		AmountCents: cents,
		Currency:    currency,
	})
	return err
}

// Ensure interface match at compile time (optional)
var _ interface {
	CreateOrder(ctx context.Context, orderID, userID string, cents int64, currency string) error
} = (*OrderGWClient)(nil)

// If you prefer to re-use the same method signature as in your handler port:
//
// var _ handlers.OrderGateway = (*OrderGWClient)(nil)
