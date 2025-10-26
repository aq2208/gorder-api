package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aq2208/gorder-api/internal/usecase"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	exchangeName = "order.events"
	routingKey   = "order.created"
	queueName    = "order.created.q"
)

// RabbitProducer implements usecase.OrderQueue
type RabbitProducer struct {
	ch *amqp.Channel
}

// NewRabbitProducer sets up the exchange, queue, and binding once at startup.
func NewRabbitProducer(ch *amqp.Channel) (*RabbitProducer, error) {
	// 1. declare exchange (topic type, durable)
	if err := ch.ExchangeDeclare(
		exchangeName,
		"topic",
		true,  // durable
		false, // auto-delete
		false, // internal
		false, // no-wait
		nil,
	); err != nil {
		return nil, fmt.Errorf("declare exchange: %w", err)
	}

	// 2. declare queue
	q, err := ch.QueueDeclare(
		queueName,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("declare queue: %w", err)
	}

	// 3. bind queue → exchange
	if err := ch.QueueBind(
		q.Name,
		routingKey,
		exchangeName,
		false, // no-wait
		nil,
	); err != nil {
		return nil, fmt.Errorf("queue bind: %w", err)
	}

	// 4. enable publisher confirms (optional but recommended)
	if err := ch.Confirm(false); err != nil {
		return nil, fmt.Errorf("enable confirm mode: %w", err)
	}

	return &RabbitProducer{ch: ch}, nil
}

// PublishCreated sends an "order.created" event to the exchange.
func (p *RabbitProducer) PublishCreated(ctx context.Context, msg usecase.CreatedMsg) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	pub := amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent, // survive broker restarts
		Body:         body,
	}

	// Publish with context-aware cancellation
	if err := p.ch.PublishWithContext(
		ctx,
		exchangeName, // exchange
		routingKey,   // routing key
		false,        // mandatory
		false,        // immediate
		pub,
	); err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	return nil
}
