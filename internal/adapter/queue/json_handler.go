package queue

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
)

// JSONHandler adapts a typed function into a raw Delivery handler.
// It unmarshals d.Body into T and calls HandleFunc(ctx, T).
type JSONHandler[T any] struct {
	HandleFunc func(ctx context.Context, msg T) error
}

func (h JSONHandler[T]) Handle(ctx context.Context, d amqp.Delivery) error {
	var v T
	if err := json.Unmarshal(d.Body, &v); err != nil {
		return err
	}
	return h.HandleFunc(ctx, v)
}
