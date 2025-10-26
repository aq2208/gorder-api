package queue

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Handler processes a single delivery. It should be idempotent.
// Return nil => ACK; return error => NACK (requeue behavior controlled by Router).
type Handler interface {
	Handle(ctx context.Context, d amqp.Delivery) error
}
